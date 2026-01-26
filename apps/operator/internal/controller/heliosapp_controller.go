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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

// HeliosAppReconciler reconciles a HeliosApp object
type HeliosAppReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	CueEngine heliosCue.CueEngineInterface
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
	// PHASE 0: Setup CI/CD Triggers (Tekton)
	// ------------------------------------------------------------------

	// Define resource names
	triggerBindingName := heliosApp.Name + "-git-binding"
	defaultsBindingName := heliosApp.Name + "-defaults-binding"
	triggerTemplateName := heliosApp.Name + "-template"
	eventListenerName := heliosApp.Name + "-listener"

	pipelineName := heliosApp.Spec.PipelineName
	if pipelineName == "" {
		pipelineName = "from-code-to-cluster"
	}
	serviceAccount := heliosApp.Spec.ServiceAccount
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	webhookSecret := heliosApp.Spec.WebhookSecret
	if webhookSecret == "" {
		webhookSecret = "github-credentials" // Unified secret name
	}

	// 1. Create/Update TriggerBinding (Git Info)
	tbGit, _ := GenerateTriggerBinding(triggerBindingName, heliosApp.Namespace)
	if err := ctrl.SetControllerReference(&heliosApp, tbGit, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for TriggerBinding (Git)")
	} else {
		foundTbGit := &unstructured.Unstructured{}
		foundTbGit.SetGroupVersionKind(tbGit.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: tbGit.GetName(), Namespace: tbGit.GetNamespace()}, foundTbGit); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating TriggerBinding (Git)", "name", tbGit.GetName())
				r.Client.Create(ctx, tbGit)
			}
		}
	}

	// 2. Create/Update TriggerBinding (Defaults)
	tbDefaults, _ := GenerateDefaultsTriggerBinding(defaultsBindingName, heliosApp.Namespace, &heliosApp)
	if err := ctrl.SetControllerReference(&heliosApp, tbDefaults, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for TriggerBinding (Defaults)")
	} else {
		foundTbDefaults := &unstructured.Unstructured{}
		foundTbDefaults.SetGroupVersionKind(tbDefaults.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: tbDefaults.GetName(), Namespace: tbDefaults.GetNamespace()}, foundTbDefaults); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating TriggerBinding (Defaults)", "name", tbDefaults.GetName())
				r.Client.Create(ctx, tbDefaults)
			}
		}
	}

	// 3. Create/Update TriggerTemplate
	workspaceConfig := map[string]any{} // Placeholder
	tt, _ := GenerateTriggerTemplate(triggerTemplateName, heliosApp.Namespace, heliosApp.Name+"-run", pipelineName, serviceAccount, workspaceConfig)
	if err := ctrl.SetControllerReference(&heliosApp, tt, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for TriggerTemplate")
	} else {
		foundTt := &unstructured.Unstructured{}
		foundTt.SetGroupVersionKind(tt.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: tt.GetName(), Namespace: tt.GetNamespace()}, foundTt); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating TriggerTemplate", "name", tt.GetName())
				r.Client.Create(ctx, tt)
			}
		}
	}

	// 4. Create/Update EventListener
	el, _ := GenerateEventListener(eventListenerName, heliosApp.Namespace, "github-push", triggerBindingName, defaultsBindingName, triggerTemplateName, webhookSecret)
	if err := ctrl.SetControllerReference(&heliosApp, el, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for EventListener")
	} else {
		foundEl := &unstructured.Unstructured{}
		foundEl.SetGroupVersionKind(el.GroupVersionKind())
		if err := r.Client.Get(ctx, client.ObjectKey{Name: el.GetName(), Namespace: el.GetNamespace()}, foundEl); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating EventListener", "name", el.GetName())
				r.Client.Create(ctx, el)
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

	// 8. Ensure Ingress for EventListener (if configured)
	if heliosApp.Spec.WebhookDomain != "" {
		log.Info("Ensuring Ingress for EventListener")
		// Correctly use the EventListener name defined in Phase 0
		ing, err := GenerateIngress(&heliosApp, eventListenerName)
		if err != nil {
			log.Error(err, "Failed to generate Ingress")
		} else if ing != nil {
			ing.SetGroupVersionKind(ing.GroupVersionKind())
			foundIng := &unstructured.Unstructured{}
			foundIng.SetGroupVersionKind(ing.GroupVersionKind())
			key := client.ObjectKey{Name: ing.GetName(), Namespace: ing.GetNamespace()}
			if err := r.Client.Get(ctx, key, foundIng); err != nil {
				if errors.IsNotFound(err) {
					log.Info("Creating Ingress", "name", ing.GetName())
					r.Client.Create(ctx, ing)
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// mapCRDToModel converts HeliosApp CRD to CUE Application Model
func (r *HeliosAppReconciler) mapCRDToModel(app *appv1alpha1.HeliosApp) (heliosCue.Application, error) {
	components := make([]heliosCue.Component, len(app.Spec.Components))

	for i, c := range app.Spec.Components {
		// Parse properties from RawExtension
		var props map[string]interface{}
		if c.Properties != nil && c.Properties.Raw != nil {
			if err := json.Unmarshal(c.Properties.Raw, &props); err != nil {
				return heliosCue.Application{}, fmt.Errorf("failed to parse component properties: %w", err)
			}
		}

		// Parse traits
		traits := make([]heliosCue.Trait, len(c.Traits))
		for j, t := range c.Traits {
			var traitProps map[string]interface{}
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
		Named("heliosapp").
		Complete(r)
}
