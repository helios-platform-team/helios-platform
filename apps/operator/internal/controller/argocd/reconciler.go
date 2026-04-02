package argocd

import (
	"cmp"
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

// Reconciler handles ArgoCD Application and RBAC reconciliation.
type Reconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// NewReconciler creates a new ArgoCD Reconciler.
func NewReconciler(c client.Client, scheme *runtime.Scheme) *Reconciler {
	return &Reconciler{Client: c, Scheme: scheme}
}

// Reconcile ensures the ArgoCD Application and sync RBAC exist.
func (r *Reconciler) Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	log.Info("Ensuring ArgoCD Application exists")
	argoApp, err := GenerateArgoApplication(app)
	if err != nil {
		log.Error(err, "Failed to generate ArgoCD Application manifest")
		return fmt.Errorf("failed to generate ArgoCD Application: %w", err)
	}

	argoApp.SetGroupVersionKind(argoApp.GroupVersionKind())

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
		log.Info("ArgoCD Application already exists", "name", argoApp.GetName())
	}

	if err := r.ensureSyncRBAC(ctx, app); err != nil {
		return fmt.Errorf("failed to ensure sync RBAC: %w", err)
	}

	return nil
}

// ensureSyncRBAC grants the pipeline ServiceAccount permission to patch
// the Argo CD Application so the argocd-sync Tekton task can use kubectl
// (in-cluster) without an Argo CD API token.
func (r *Reconciler) ensureSyncRBAC(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)
	argoNS := cmp.Or(app.Spec.ArgoCDNamespace, "argocd")

	role := GenerateSyncRole(app)
	if err := r.applyUnstructured(ctx, role); err != nil {
		log.Error(err, "Failed to reconcile Argo CD sync Role", "namespace", argoNS)
		return fmt.Errorf("failed to apply sync Role: %w", err)
	}

	rb := GenerateSyncRoleBinding(app)
	if err := r.applyUnstructured(ctx, rb); err != nil {
		log.Error(err, "Failed to reconcile Argo CD sync RoleBinding", "namespace", argoNS)
		return fmt.Errorf("failed to apply sync RoleBinding: %w", err)
	}

	return nil
}

func (r *Reconciler) applyUnstructured(ctx context.Context, obj *unstructured.Unstructured) error {
	key := client.ObjectKeyFromObject(obj)
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GroupVersionKind())
	err := r.Client.Get(ctx, key, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Client.Create(ctx, obj)
		}
		return err
	}
	obj.SetResourceVersion(found.GetResourceVersion())
	return r.Client.Update(ctx, obj)
}
