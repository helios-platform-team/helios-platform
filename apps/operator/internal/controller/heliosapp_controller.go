/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

// HeliosAppReconciler reconciles a HeliosApp object
type HeliosAppReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	CueEngine heliosCue.CueEngineInterface
	// TektonRenderer renders Tekton CI/CD resources via CUE engine.
	TektonRenderer heliosCue.TektonRendererInterface
	// GitFactory allows injecting a custom GitOps client (e.g. for testing)
	GitFactory func(string, string, string) gitops.GitOpsClientInterface
}

// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile handles the reconciliation loop for HeliosApp
// Controller does NOT iterate components/traits - all orchestration is in CUE
func (r *HeliosAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch HeliosApp CRD
	var heliosApp appv1alpha1.HeliosApp
	if err := r.Get(ctx, req.NamespacedName, &heliosApp); err != nil {
		if errors.IsNotFound(err) {
			log.Info("HeliosApp resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Reconciling HeliosApp", "name", heliosApp.Name, "namespace", heliosApp.Namespace)

	// 2. Map CRD to Application Model
	appModel, err := r.mapCRDToModel(&heliosApp)
	if err != nil {
		log.Error(err, "Failed to map CRD to application model")
		return ctrl.Result{}, err
	}

	// VALIDATION: Ensure image is present (Fix "First Commit Missing Image")
	for _, comp := range appModel.App.Components {
		// We can add more specific checks here based on component type
		// For now, checks if 'image' property exists and is not empty for all components
		// assuming all workloads need an image.
		if img, ok := comp.Properties["image"].(string); !ok || img == "" {
			msg := fmt.Sprintf("Component '%s' is waiting for image (likely building). Status: Pending.", comp.Name)
			log.Info(msg)
			r.updateStatus(ctx, &heliosApp, appv1alpha1.PhasePending, msg)
			return ctrl.Result{}, nil // Wait for next update (CI/CD will update CR with image)
		}
	}

	// 3. Render via CUE Engine
	manifestBytes, err := r.CueEngine.Render(appModel)
	if err != nil {
		log.Error(err, "Failed to render application via CUE")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("CUE rendering failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE -1 & 0: Tekton CI/CD Resources (Tasks, Pipeline, Triggers)
	// All Tekton resources are rendered via CUE engine.
	// ------------------------------------------------------------------
	if err := r.reconcileTektonResourcesCue(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile Tekton resources via CUE")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("CUE Tekton rendering failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE 0.6: Trigger Initial PipelineRun (if not already done)
	// ------------------------------------------------------------------
	if !heliosApp.Status.InitialBuildTriggered {
		log.Info("Triggering initial PipelineRun for new HeliosApp")

		pipelineName := heliosApp.Spec.PipelineName
		if pipelineName == "" {
			pipelineName = "from-code-to-cluster"
		}
		pr, err := GeneratePipelineRunForManifestGeneration(&heliosApp, pipelineName)
		if err != nil {
			log.Error(err, "Failed to generate initial PipelineRun")
		} else {
			if err := ctrl.SetControllerReference(&heliosApp, pr, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference for PipelineRun")
			}

			if err := r.Client.Create(ctx, pr); err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Error(err, "Failed to create initial PipelineRun")
				}
			} else {
				log.Info("Created initial PipelineRun", "name", pr.GetName())
			}

			// Mark as triggered to avoid creating multiple PipelineRuns
			heliosApp.Status.InitialBuildTriggered = true
			if err := r.Status().Update(ctx, &heliosApp); err != nil {
				log.Error(err, "Failed to update InitialBuildTriggered status")
			}
		}
	}

	// ------------------------------------------------------------------
	// PHASE 1: Render & GitOps (Moved below)
	// ------------------------------------------------------------------

	// 4. GitOps Helper: Get Token & Username
	token := os.Getenv("GITHUB_TOKEN")
	username := os.Getenv("GITHUB_USER")
	if username == "" {
		username = "git" // Default fallback
	}

	if heliosApp.Spec.GitOpsSecretRef != "" {
		var secret corev1.Secret
		// Explicitly log the secret lookup attempt
		if err := r.Get(ctx, types.NamespacedName{Name: heliosApp.Spec.GitOpsSecretRef, Namespace: heliosApp.Namespace}, &secret); err == nil {
			if t, ok := secret.Data["token"]; ok {
				token = string(t)
			} else if p, ok := secret.Data["password"]; ok {
				// Fallback to 'password' key (standard basic-auth secret from Tekton setup)
				token = string(p)
			} else {
				log.Info("Secret found but 'token' or 'password' key is missing", "Secret", heliosApp.Spec.GitOpsSecretRef)
				r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Secret %s missing 'token' key", heliosApp.Spec.GitOpsSecretRef))
				return ctrl.Result{}, nil
			}
			if u, ok := secret.Data["username"]; ok {
				username = string(u)
			}
		} else {
			log.Error(err, "Failed to get GitOps Secret", "Secret", heliosApp.Spec.GitOpsSecretRef)
			r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Secret %s not found", heliosApp.Spec.GitOpsSecretRef))
			return ctrl.Result{}, nil
		}
	}

	// 5. GitOps Sync

	if token == "" {
		err := fmt.Errorf("GitOps token is empty. Check Secret or GITHUB_TOKEN env var")
		log.Error(err, "Authentication failed")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, "GitOps token missing")
		return ctrl.Result{}, nil // Don't retry immediately if config is missing
	}

	// OPTIMIZATION: Check Hash
	currentHash := r.computeHash(manifestBytes)
	if heliosApp.Status.LastAppliedHash == currentHash {
		log.Info("Manifest hash unchanged, skipping GitOps sync", "hash", currentHash)

		// Still ensure status is Ready if it was previously set
		if heliosApp.Status.Phase != appv1alpha1.PhaseReady {
			heliosApp.Status.Phase = appv1alpha1.PhaseReady
			if err := r.Status().Update(ctx, &heliosApp); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Use GitFactory if available, otherwise default to NewGitOpsClient
		getGitClient := r.GitFactory
		if getGitClient == nil {
			getGitClient = func(repo, user, token string) gitops.GitOpsClientInterface {
				return gitops.NewGitOpsClient(repo, user, token)
			}
		}

		gitClient := getGitClient(heliosApp.Spec.GitOpsRepo, username, token)
		targetPath := fmt.Sprintf("%s/manifest.yaml", heliosApp.Spec.GitOpsPath)

		if err := gitClient.SyncManifest(ctx, targetPath, string(manifestBytes)); err != nil {
			log.Error(err, "GitOps sync failed")
			r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("GitOps failed: %v", err))
			return ctrl.Result{}, err
		}

		// 6. Update Status
		heliosApp.Status.Phase = appv1alpha1.PhaseReady
		heliosApp.Status.Message = fmt.Sprintf("Manifest pushed to %s/%s", heliosApp.Spec.GitOpsRepo, targetPath)
		heliosApp.Status.LastAppliedHash = currentHash
		// We clear ResourcesCreated as we are not managing them directly anymore
		heliosApp.Status.ResourcesCreated = nil

		if err := r.Status().Update(ctx, &heliosApp); err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		log.Info("Successfully reconciled HeliosApp via GitOps", "newHash", currentHash)
	}

	// 7. Ensure ArgoCD Application exists
	log.Info("Ensuring ArgoCD Application exists")
	argoApp, err := GenerateArgoApplication(&heliosApp)
	if err != nil {
		log.Error(err, "Failed to generate ArgoCD Application manifest")
		// We don't return error here to avoid loop if GitOps was successful, just log it.
		// Or maybe we should retry? Let's log and continue for now.
	} else {
		// Define ArgoCD Application identity
		argoApp.SetGroupVersionKind(argoApp.GroupVersionKind())
		// We use Sever-Side Apply or Create/Update logic
		// Since ArgoCD app is in "argocd" namespace usually, we need permissions there.
		// For simplicity/demo: Try to get, if not found create.

		foundArgoApp := &unstructured.Unstructured{}
		foundArgoApp.SetGroupVersionKind(argoApp.GroupVersionKind())

		key := client.ObjectKey{
			Name:      argoApp.GetName(),
			Namespace: argoApp.GetNamespace(),
		}

		if err := r.Client.Get(ctx, key, foundArgoApp); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating ArgoCD Application", "name", argoApp.GetName())
				if err := r.Client.Create(ctx, argoApp); err != nil {
					log.Error(err, "Failed to create ArgoCD Application")
				}
			} else {
				log.Error(err, "Failed to get ArgoCD Application")
			}
		} else {
			// Optional: Update if needed (checkout 'spec' diff)
			log.Info("ArgoCD Application already exists", "name", argoApp.GetName())
		}
	}

	// NOTE: Ingress removed - use port-forwarding for EventListener:
	// kubectl port-forward svc/el-<app>-listener 8080:8080

	return ctrl.Result{}, nil
}

// mapCRDToModel converts HeliosApp CRD to CUE Application Model
func (r *HeliosAppReconciler) mapCRDToModel(app *appv1alpha1.HeliosApp) (heliosCue.Application, error) {
	components := make([]heliosCue.Component, len(app.Spec.Components))

	for i, c := range app.Spec.Components {
		// Parse properties from RawExtension
		var props map[string]any
		if c.Properties != nil && c.Properties.Raw != nil {
			if err := json.Unmarshal(c.Properties.Raw, &props); err != nil {
				return heliosCue.Application{}, fmt.Errorf("failed to parse component properties: %w", err)
			}
		}

		// Parse traits
		traits := make([]heliosCue.Trait, len(c.Traits))
		for j, t := range c.Traits {
			var traitProps map[string]any
			if t.Properties != nil && t.Properties.Raw != nil {
				if err := json.Unmarshal(t.Properties.Raw, &traitProps); err != nil {
					return heliosCue.Application{}, fmt.Errorf("failed to parse trait properties: %w", err)
				}
			}
			traits[j] = heliosCue.Trait{
				Type:       t.Type,
				Properties: traitProps,
			}
		}

		components[i] = heliosCue.Component{
			Name:       c.Name,
			Type:       c.Type,
			Properties: props,
			Traits:     traits,
		}
	}

	return heliosCue.Application{
		App: heliosCue.AppSpec{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Owner:       app.Spec.Owner,
			Description: app.Spec.Description,
			Components:  components,
		},
	}, nil
}

// mapCRDToTektonInput converts HeliosApp CRD to TektonInput for CUE rendering.
// This is the bridge between HeliosApp spec fields and the CUE #TektonInput schema.
func (r *HeliosAppReconciler) mapCRDToTektonInput(app *appv1alpha1.HeliosApp) heliosCue.TektonInput {
	input := heliosCue.TektonInput{
		AppName:         app.Name,
		Namespace:       app.Namespace,
		GitRepo:         app.Spec.GitRepo,
		GitBranch:       app.Spec.GitBranch,
		ImageRepo:       app.Spec.ImageRepo,
		GitOpsRepo:      app.Spec.GitOpsRepo,
		GitOpsPath:      app.Spec.GitOpsPath,
		GitOpsBranch:    app.Spec.GitOpsBranch,
		GitOpsSecretRef: app.Spec.GitOpsSecretRef,
		WebhookDomain:   app.Spec.WebhookDomain,
		WebhookSecret:   app.Spec.WebhookSecret,
		PipelineName:    app.Spec.PipelineName,
		PipelineType:    app.Spec.PipelineName, // pipelineType uses same value as pipelineName
		TriggerType:     "github-push",         // Default; extend HeliosAppSpec if needed
		ServiceAccount:  app.Spec.ServiceAccount,
		PVCName:         app.Spec.PVCName,
		ContextSubpath:  app.Spec.ContextSubpath,
		Replicas:        int(app.Spec.Replicas),
		Port:            int(app.Spec.Port),
		TestCommand:     app.Spec.TestCommand,
		DockerSecret:    "docker-credentials",
		ArgoCDNamespace: app.Spec.ArgoCDNamespace,
		ArgoCDProject:   app.Spec.ArgoCDProject,
	}

	// Apply defaults for fields that may be empty
	input.GitBranch = cmp.Or(input.GitBranch, "main")
	input.GitOpsBranch = cmp.Or(input.GitOpsBranch, "main")
	input.GitOpsSecretRef = cmp.Or(input.GitOpsSecretRef, "github-credentials")
	input.WebhookSecret = cmp.Or(input.WebhookSecret, "github-webhook-secret")
	if input.PipelineName == "" {
		input.PipelineName = "from-code-to-cluster"
		input.PipelineType = "from-code-to-cluster"
	}
	input.ServiceAccount = cmp.Or(input.ServiceAccount, "default")
	if input.Replicas <= 0 {
		input.Replicas = 1
	}
	if input.Port <= 0 {
		input.Port = 8080
	}

	return input
}

// reconcileTektonResourcesCue renders Tekton resources via CUE and applies them.
// This is the NEW path that replaces all hardcoded Generate* functions.
func (r *HeliosAppReconciler) reconcileTektonResourcesCue(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	// 1. Map CRD → TektonInput
	tektonInput := r.mapCRDToTektonInput(app)

	// 2. Render via CUE
	objects, err := r.TektonRenderer.RenderTektonResources(tektonInput)
	if err != nil {
		return fmt.Errorf("CUE TektonRenderer failed: %w", err)
	}

	log.Info("CUE rendered Tekton resources", "count", len(objects))

	// 3. Apply each rendered resource
	for _, obj := range objects {
		// Set owner reference (skip cluster-scoped resources)
		if obj.GetNamespace() != "" {
			if err := ctrl.SetControllerReference(app, obj, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference", "kind", obj.GetKind(), "name", obj.GetName())
				continue
			}
		}

		// Create or update
		found := &unstructured.Unstructured{}
		found.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.Client.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating resource", "kind", obj.GetKind(), "name", obj.GetName())
				if err := r.Client.Create(ctx, obj); err != nil {
					log.Error(err, "Failed to create resource", "kind", obj.GetKind(), "name", obj.GetName())
				}
			} else {
				log.Error(err, "Failed to get resource", "kind", obj.GetKind(), "name", obj.GetName())
			}
		} else {
			// Update existing resource's spec
			found.Object["spec"] = obj.Object["spec"]
			if err := r.Client.Update(ctx, found); err != nil {
				log.Error(err, "Failed to update resource", "kind", obj.GetKind(), "name", obj.GetName())
			}
		}
	}

	// 4. Also ensure RBAC (SA, RoleBinding, ClusterRoleBinding) — these are not in CUE yet
	r.ensureTektonRBAC(ctx, app)

	return nil
}

// ensureTektonRBAC creates ServiceAccount, RoleBinding, ClusterRoleBinding.
// These are infrastructure resources not managed by CUE (they are cluster lifecycle, not app lifecycle).
func (r *HeliosAppReconciler) ensureTektonRBAC(ctx context.Context, app *appv1alpha1.HeliosApp) {
	log := logf.FromContext(ctx)

	sa := GenerateServiceAccount(app.Namespace)
	if err := ctrl.SetControllerReference(app, sa, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for ServiceAccount")
	} else {
		foundSA := &unstructured.Unstructured{}
		foundSA.SetGroupVersionKind(sa.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: sa.GetName(), Namespace: sa.GetNamespace()}, foundSA); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating ServiceAccount", "name", sa.GetName())
				r.Client.Create(ctx, sa)
			}
		}
	}

	rb := GenerateRoleBinding(app.Namespace)
	if err := ctrl.SetControllerReference(app, rb, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for RoleBinding")
	} else {
		foundRB := &unstructured.Unstructured{}
		foundRB.SetGroupVersionKind(rb.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: rb.GetName(), Namespace: rb.GetNamespace()}, foundRB); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating RoleBinding", "name", rb.GetName())
				r.Client.Create(ctx, rb)
			}
		}
	}

	crb := GenerateClusterRoleBinding(app.Namespace)
	foundCrb := &unstructured.Unstructured{}
	foundCrb.SetGroupVersionKind(crb.GroupVersionKind())
	if err := r.Client.Get(ctx, client.ObjectKey{Name: crb.GetName()}, foundCrb); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating ClusterRoleBinding", "name", crb.GetName())
			r.Client.Create(ctx, crb)
		}
	}
}

// updateStatus updates the HeliosApp status
func (r *HeliosAppReconciler) updateStatus(ctx context.Context, app *appv1alpha1.HeliosApp, phase appv1alpha1.HeliosAppPhase, message string) {
	app.Status.Phase = phase
	app.Status.Message = message
	if err := r.Status().Update(ctx, app); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to update status")
	}
}

// computeHash returns SHA256 of data
func (r *HeliosAppReconciler) computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SetupWithManager sets up the controller with the Manager
func (r *HeliosAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.HeliosApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
		).
		Named("heliosapp").
		Complete(r)
}

// findObjectsForSecret maps Secret changes to HeliosApp reconcile requests.
// This ensures the controller re-reconciles when a referenced secret changes.
func (r *HeliosAppReconciler) findObjectsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	// List all HeliosApps in the same namespace
	var heliosAppList appv1alpha1.HeliosAppList
	if err := r.List(ctx, &heliosAppList, client.InNamespace(obj.GetNamespace())); err != nil {
		log.Error(err, "Failed to list HeliosApps for secret watch")
		return nil
	}

	var requests []reconcile.Request
	for _, app := range heliosAppList.Items {
		// Check if this app references the changed secret
		if app.Spec.GitOpsSecretRef == obj.GetName() ||
			app.Spec.WebhookSecret == obj.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      app.Name,
					Namespace: app.Namespace,
				},
			})
		}
	}

	return requests
}
