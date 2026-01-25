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
	"encoding/json"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	CueEngine *heliosCue.Engine
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

	log.Info("CUE rendered manifest successfully")

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
			} else {
				log.Info("Secret found but 'token' key is missing", "Secret", heliosApp.Spec.GitOpsSecretRef)
			}
			if u, ok := secret.Data["username"]; ok {
				username = string(u)
			}
		} else {
			log.Error(err, "Failed to get GitOps Secret", "Secret", heliosApp.Spec.GitOpsSecretRef)
		}
	}

	// 5. GitOps Sync

	if token == "" {
		err := fmt.Errorf("GitOps token is empty. Check Secret or GITHUB_TOKEN env var")
		log.Error(err, "Authentication failed")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, "GitOps token missing")
		return ctrl.Result{}, nil // Don't retry immediately if config is missing
	}

	gitClient := gitops.NewGitOpsClient(heliosApp.Spec.GitOpsRepo, username, token)
	targetPath := fmt.Sprintf("%s/manifest.yaml", heliosApp.Spec.GitOpsPath)

	if err := gitClient.SyncManifest(ctx, targetPath, string(manifestBytes)); err != nil {
		log.Error(err, "GitOps sync failed")
		r.updateStatus(ctx, &heliosApp, appv1alpha1.PhaseFailed, fmt.Sprintf("GitOps failed: %v", err))
		return ctrl.Result{}, err
	}

	// 6. Update Status
	heliosApp.Status.Phase = appv1alpha1.PhaseReady
	heliosApp.Status.Message = fmt.Sprintf("Manifest pushed to %s/%s", heliosApp.Spec.GitOpsRepo, targetPath)
	// We clear ResourcesCreated as we are not managing them directly anymore
	heliosApp.Status.ResourcesCreated = nil

	if err := r.Status().Update(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled HeliosApp via GitOps")
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
