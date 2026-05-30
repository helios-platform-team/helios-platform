package tekton

import (
	"context"
	"fmt"

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
