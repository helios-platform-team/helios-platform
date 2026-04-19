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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/argocd"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/database"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/gitopssync"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/tekton"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

// HeliosAppReconciler reconciles a HeliosApp object.
type HeliosAppReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	CueEngine heliosCue.CueEngineInterface

	Tekton   TektonReconciler
	ArgoCD   ArgoCDReconciler
	Database DatabaseReconciler
	GitOps   GitOpsSyncReconciler
}

// NewHeliosAppReconciler wires all sub-reconcilers and returns a ready-to-use reconciler.
func NewHeliosAppReconciler(
	c client.Client,
	scheme *runtime.Scheme,
	cueEngine heliosCue.CueEngineInterface,
	tektonRenderer heliosCue.TektonRendererInterface,
	gitFactory func(string, string, string) gitops.GitOpsClientInterface,
) *HeliosAppReconciler {
	return &HeliosAppReconciler{
		Client:    c,
		Scheme:    scheme,
		CueEngine: cueEngine,
		Tekton:    tekton.NewReconciler(c, scheme, tektonRenderer),
		ArgoCD:    argocd.NewReconciler(c, scheme),
		Database:  database.NewReconciler(c, scheme),
		GitOps:    gitopssync.NewReconciler(c, scheme, gitFactory),
	}
}

// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.helios.io,resources=heliosapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for HeliosApp.
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

	// Pre-flight validation: Check if all referenced secrets exist
	if err := r.validateSecretReferences(ctx, &heliosApp); err != nil {
		log.Error(err, "Pre-flight validation failed: referenced secret does not exist")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Configuration error: %v", err))
		return ctrl.Result{}, err
	}

	// 2. Map CRD to Application Model
	appModel, err := mapCRDToModel(&heliosApp)
	if err != nil {
		log.Error(err, "Failed to map CRD to application model")
		return ctrl.Result{}, err
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
	if err := r.Tekton.Reconcile(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile Tekton resources")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Tekton reconciliation failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE 0.5: Database Credential Secrets
	// Generate and store secure credentials for components with database traits.
	// Secrets are created BEFORE GitOps sync to ensure credentials exist
	// when the application is deployed.
	// ------------------------------------------------------------------
	if err := r.Database.ReconcileSecrets(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile database secrets")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Database secret creation failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE 0.7: Database Instance Provisioning
	// Provision StatefulSets and headless Services for database traits.
	// Runs AFTER secrets so that the credential Secret already exists
	// when the database pod starts.
	// ------------------------------------------------------------------
	if err := r.Database.ReconcileInstances(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile database instance")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Database instance provisioning failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE 0.9: Inject Database Credentials into Backend Deployment
	// Patches the live Deployment (deployed by ArgoCD) to add DB_HOST,
	// DB_USER, DB_PASS, DB_PORT, DB_NAME, and DATABASE_URL (via $(VAR) expansion).
	// Runs AFTER secrets and instances so the Secret already exists.
	// ------------------------------------------------------------------
	dbInjectionPending, err := r.Database.ReconcileInjection(ctx, &heliosApp)
	if err != nil {
		log.Error(err, "Failed to inject database secrets into Deployment")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Database secret injection failed: %v", err))
		return ctrl.Result{}, err
	}

	for _, comp := range appModel.App.Components {
		if img, ok := comp.Properties["image"].(string); !ok || img == "" {
			msg := fmt.Sprintf("Component '%s' is waiting for image (likely building). Status: Pending.", comp.Name)
			log.Info(msg)
			r.updateStatus(ctx, &heliosApp, appv1alpha1.PhasePending, msg)
			return ctrl.Result{}, nil
		}
	}

	if err := r.Tekton.ReconcileInitialPipelineRun(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile initial PipelineRun")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("Initial PipelineRun failed: %v", err))
		return ctrl.Result{}, err
	}

	// ------------------------------------------------------------------
	// PHASE 1: Render & GitOps
	// ------------------------------------------------------------------

	if err := r.GitOps.Reconcile(ctx, &heliosApp, manifestBytes); err != nil {
		log.Error(err, "GitOps sync failed")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("GitOps sync failed: %v", err))
		return ctrl.Result{}, err
	}

	if err := r.ArgoCD.Reconcile(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to reconcile ArgoCD Application")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("ArgoCD reconciliation failed: %v", err))
		return ctrl.Result{}, err
	}

	if dbInjectionPending {
		log.Info("Database secret injection pending, requeuing")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// updateStatus updates the HeliosApp status.
func (r *HeliosAppReconciler) updateStatus(ctx context.Context, app *appv1alpha1.HeliosApp, phase appv1alpha1.HeliosAppPhase, message string) {
	app.Status.Phase = phase
	app.Status.Message = message
	if err := r.Status().Update(ctx, app); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to update status")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeliosAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.HeliosApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
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
			app.Spec.WebhookSecret == obj.GetName() ||
			app.Spec.DatabaseSecretRef == obj.GetName() {
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

// validateSecretReferences checks if all referenced secrets exist in the cluster.
// This is a pre-flight validation to catch configuration errors early.
// Note: Database secrets are NOT validated here because they are auto-created
// by the operator in Phase 0.5 if database traits are present.
func (r *HeliosAppReconciler) validateSecretReferences(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	secretsToValidate := map[string]string{
		"webhook secret": app.Spec.WebhookSecret,
		"GitOps secret":  app.Spec.GitOpsSecretRef,
		// Note: database secret is NOT validated here - it's auto-created in Phase 0.5
	}

	for secretType, secretName := range secretsToValidate {
		if secretName == "" {
			continue // Skip empty references
		}

		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: app.Namespace}, &secret); err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("%s '%s' not found in namespace '%s'", secretType, secretName, app.Namespace)
			}
			return fmt.Errorf("failed to validate %s '%s': %w", secretType, secretName, err)
		}
	}

	return nil
}
