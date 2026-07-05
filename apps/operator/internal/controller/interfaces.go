package controller

import (
	"context"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

// TektonReconciler handles Tekton CI/CD resource reconciliation.
type TektonReconciler interface {
	Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp) error
	ReconcileInitialPipelineRun(ctx context.Context, app *appv1alpha1.HeliosApp) error
}

// ArgoCDReconciler handles ArgoCD Application reconciliation.
type ArgoCDReconciler interface {
	Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp) error
}

// DatabaseReconciler handles database provisioning and secret management.
type DatabaseReconciler interface {
	ReconcileSystemSecrets(ctx context.Context, app *appv1alpha1.HeliosApp) error
	ReconcileSecrets(ctx context.Context, app *appv1alpha1.HeliosApp) error
	ReconcileInstances(ctx context.Context, app *appv1alpha1.HeliosApp) error
	ReconcileInjection(ctx context.Context, app *appv1alpha1.HeliosApp) (pending bool, err error)
}

// GitOpsSyncReconciler handles GitOps manifest sync.
type GitOpsSyncReconciler interface {
	Reconcile(ctx context.Context, app *appv1alpha1.HeliosApp, crBytes []byte) error
}
