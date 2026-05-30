package tekton

import (
	"context"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
)

// Reconciler handles Tekton CI/CD resource reconciliation.
type Reconciler struct {
	Client         client.Client
	Scheme         *runtime.Scheme
	TektonRenderer heliosCue.TektonRendererInterface
}

// NewReconciler creates a new Tekton Reconciler.
func NewReconciler(c client.Client, scheme *runtime.Scheme, renderer heliosCue.TektonRendererInterface) *Reconciler {
	return &Reconciler{Client: c, Scheme: scheme, TektonRenderer: renderer}
}

// Reconcile renders Tekton resources via CUE and applies them, then ensures RBAC.
func (r *Reconciler) Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	tektonInput := MapCRDToTektonInput(app)

	objects, err := r.TektonRenderer.RenderTektonResources(tektonInput)
	if err != nil {
		return fmt.Errorf("CUE TektonRenderer failed: %w", err)
	}

	log.Info("CUE rendered Tekton resources", "count", len(objects))

	for _, obj := range objects {
		if obj.GetNamespace() != "" {
			if err := ctrl.SetControllerReference(app, obj, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference", "kind", obj.GetKind(), "name", obj.GetName())
				return fmt.Errorf("failed to set owner reference for %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
		}

		found := &unstructured.Unstructured{}
		found.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.Client.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Creating resource", "kind", obj.GetKind(), "name", obj.GetName())
				if err := r.Client.Create(ctx, obj); err != nil {
					log.Error(err, "Failed to create resource", "kind", obj.GetKind(), "name", obj.GetName())
					return fmt.Errorf("failed to create resource %s %s: %w", obj.GetKind(), obj.GetName(), err)
				}
			} else {
				log.Error(err, "Failed to get resource", "kind", obj.GetKind(), "name", obj.GetName())
				return fmt.Errorf("failed to get resource %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
		} else {
			found.Object["spec"] = obj.Object["spec"]
			if err := r.Client.Update(ctx, found); err != nil {
				log.Error(err, "Failed to update resource", "kind", obj.GetKind(), "name", obj.GetName())
				return fmt.Errorf("failed to update resource %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
		}
	}

	r.ensureRBAC(ctx, app)

	// Keep only the 3 most recent PipelineRuns for this application
	if err := r.PrunePipelineRuns(ctx, app.Namespace, app.Name, 3); err != nil {
		log.Error(err, "Failed to prune old PipelineRuns")
	}

	// Cancel any older concurrent PipelineRuns to prevent out-of-order deployments
	if err := r.CancelOlderPipelineRuns(ctx, app.Namespace, app.Name); err != nil {
		log.Error(err, "Failed to cancel concurrent PipelineRuns")
	}

	return nil
}

// ReconcileInitialPipelineRun triggers the first PipelineRun if not already done.
func (r *Reconciler) ReconcileInitialPipelineRun(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	if app.Status.InitialBuildTriggered {
		return nil
	}

	log.Info("Triggering initial PipelineRun for new HeliosApp")

	pipelineName := app.Spec.PipelineName
	if pipelineName == "" {
		pipelineName = defaultPipelineName
	}

	// Only trigger if triggerType is not gitea-push or if we want to force initial build.
	// If it's gitea-push, we only skip if a PipelineRun ALREADY exists (triggered by webhook).
	// This prevents missing the initial build if the webhook fails or is delayed.
	if app.Spec.TriggerType == "gitea-push" {
		prList := &unstructured.UnstructuredList{}
		prList.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "tekton.dev",
			Version: "v1",
			Kind:    "PipelineRunList",
		})
		err := r.Client.List(ctx, prList, client.InNamespace(app.Namespace), client.MatchingLabels{
			"app.kubernetes.io/name": app.Name,
		})
		if err == nil && len(prList.Items) > 0 {
			log.Info("Skipping initial PipelineRun because one already exists for gitea-push", "count", len(prList.Items))
			app.Status.InitialBuildTriggered = true
			return r.Client.Status().Update(ctx, app)
		}
		log.Info("No PipelineRun found for gitea-push app; triggering initial build as fallback")
	}

	pr, err := GeneratePipelineRun(app, pipelineName)
	if err != nil {
		log.Error(err, "Failed to generate initial PipelineRun")
		return nil
	}

	if err := ctrl.SetControllerReference(app, pr, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for PipelineRun")
	}

	if err := r.Client.Create(ctx, pr); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create initial PipelineRun: %w", err)
		}
	} else {
		log.Info("Created initial PipelineRun", "name", pr.GetName())
	}

	app.Status.InitialBuildTriggered = true
	if err := r.Client.Status().Update(ctx, app); err != nil {
		log.Error(err, "Failed to update InitialBuildTriggered status")
	}

	return nil
}

// ensureRBAC creates ServiceAccount, RoleBinding, ClusterRoleBinding for Tekton triggers.
func (r *Reconciler) ensureRBAC(ctx context.Context, app *appv1alpha1.HeliosApp) {
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
				if err := r.Client.Create(ctx, sa); err != nil {
					log.Error(err, "Failed to create ServiceAccount", "name", sa.GetName())
				}
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
				if err := r.Client.Create(ctx, rb); err != nil {
					log.Error(err, "Failed to create RoleBinding", "name", rb.GetName())
				}
			}
		}
	}

	crb := GenerateClusterRoleBinding(app.Namespace)
	foundCrb := &unstructured.Unstructured{}
	foundCrb.SetGroupVersionKind(crb.GroupVersionKind())
	if err := r.Client.Get(ctx, client.ObjectKey{Name: crb.GetName()}, foundCrb); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating ClusterRoleBinding", "name", crb.GetName())
			if err := r.Client.Create(ctx, crb); err != nil {
				log.Error(err, "Failed to create ClusterRoleBinding", "name", crb.GetName())
			}
		}
	}
}

// PrunePipelineRuns deletes older PipelineRuns, keeping only the 'keep' most recent ones.
func (r *Reconciler) PrunePipelineRuns(ctx context.Context, namespace, appName string, keep int) error {
	log := logf.FromContext(ctx)

	prList := &unstructured.UnstructuredList{}
	prList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tekton.dev",
		Version: "v1",
		Kind:    "PipelineRunList",
	})

	err := r.Client.List(ctx, prList, client.InNamespace(namespace), client.MatchingLabels{
		"app.kubernetes.io/name": appName,
	})
	if err != nil {
		return fmt.Errorf("failed to list PipelineRuns for pruning: %w", err)
	}

	if len(prList.Items) <= keep {
		return nil
	}

	// Sort oldest first
	sort.Slice(prList.Items, func(i, j int) bool {
		iTime := prList.Items[i].GetCreationTimestamp()
		jTime := prList.Items[j].GetCreationTimestamp()
		return iTime.Before(&jTime)
	})

	toDelete := len(prList.Items) - keep
	log.Info("Pruning old PipelineRuns", "count", toDelete, "appName", appName)

	for i := 0; i < toDelete; i++ {
		pr := prList.Items[i]
		log.Info("Pruning PipelineRun", "name", pr.GetName())
		if err := r.Client.Delete(ctx, &pr); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete old PipelineRun during pruning", "name", pr.GetName())
		}
	}

	return nil
}

// CancelOlderPipelineRuns cancels any active PipelineRuns except the most recent one.
func (r *Reconciler) CancelOlderPipelineRuns(ctx context.Context, namespace, appName string) error {
	log := logf.FromContext(ctx)

	prList := &unstructured.UnstructuredList{}
	prList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tekton.dev",
		Version: "v1",
		Kind:    "PipelineRunList",
	})

	err := r.Client.List(ctx, prList, client.InNamespace(namespace), client.MatchingLabels{
		"app.kubernetes.io/name": appName,
	})
	if err != nil {
		return fmt.Errorf("failed to list PipelineRuns for cancellation check: %w", err)
	}

	if len(prList.Items) <= 1 {
		return nil
	}

	// Sort newest first (jTime before iTime means iTime is newer, so sorting newest first)
	sort.Slice(prList.Items, func(i, j int) bool {
		iTime := prList.Items[i].GetCreationTimestamp()
		jTime := prList.Items[j].GetCreationTimestamp()
		return jTime.Before(&iTime)
	})

	// Find the most recent active/running PipelineRun to keep
	var activeRunToKeep *unstructured.Unstructured
	for i := range prList.Items {
		pr := &prList.Items[i]
		if isPipelineRunActive(pr) {
			activeRunToKeep = pr
			break
		}
	}

	if activeRunToKeep == nil {
		return nil
	}

	// Cancel any other running PipelineRuns that are older than the activeRunToKeep
	for i := range prList.Items {
		pr := &prList.Items[i]
		if pr.GetName() == activeRunToKeep.GetName() {
			continue
		}

		if isPipelineRunActive(pr) {
			log.Info("Cancelling older running PipelineRun to prevent race condition", "name", pr.GetName(), "appName", appName)

			patch := client.MergeFrom(pr.DeepCopy())
			if err := unstructured.SetNestedField(pr.Object, "Cancelled", "spec", "status"); err != nil {
				log.Error(err, "Failed to set status field for cancellation", "name", pr.GetName())
				continue
			}
			if err := r.Client.Patch(ctx, pr, patch); err != nil {
				log.Error(err, "Failed to cancel older PipelineRun", "name", pr.GetName())
			}
		}
	}

	return nil
}

// isPipelineRunActive checks if a PipelineRun is currently active/running.
func isPipelineRunActive(pr *unstructured.Unstructured) bool {
	conditions, found, _ := unstructured.NestedSlice(pr.Object, "status", "conditions")
	if !found || len(conditions) == 0 {
		return true // No conditions yet means it's newly created and active
	}

	for _, condObj := range conditions {
		cond, ok := condObj.(map[string]interface{})
		if !ok {
			continue
		}
		if cond["type"] == "Succeeded" {
			return cond["status"] == "Unknown"
		}
	}
	return true
}
