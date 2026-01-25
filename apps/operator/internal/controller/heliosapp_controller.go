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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
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
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

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

	// 3. Render via CUE Engine (Single call - NO looping in Go)
	objects, err := r.CueEngine.RenderToObjects(appModel)
	if err != nil {
		log.Error(err, "Failed to render application via CUE")
		r.updateStatus(ctx, &heliosApp, "Failed", fmt.Sprintf("CUE rendering failed: %v", err))
		return ctrl.Result{}, err
	}

	log.Info("CUE rendered Kubernetes objects", "count", len(objects))

	// 4. Apply rendered resources
	var resourcesCreated []appv1alpha1.ResourceRef
	for _, obj := range objects {
		ref, err := r.applyResource(ctx, &heliosApp, obj)
		if err != nil {
			log.Error(err, "Failed to apply resource", "kind", obj["kind"], "name", obj["metadata"].(map[string]interface{})["name"])
			return ctrl.Result{}, err
		}
		resourcesCreated = append(resourcesCreated, ref)
	}

	// 5. Update status
	heliosApp.Status.Phase = "Ready"
	heliosApp.Status.Message = fmt.Sprintf("Successfully deployed %d resources", len(resourcesCreated))
	heliosApp.Status.ResourcesCreated = resourcesCreated
	if err := r.Status().Update(ctx, &heliosApp); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled HeliosApp", "resources", len(resourcesCreated))
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

// applyResource applies a single Kubernetes resource
func (r *HeliosAppReconciler) applyResource(ctx context.Context, owner *appv1alpha1.HeliosApp, obj map[string]interface{}) (appv1alpha1.ResourceRef, error) {
	log := logf.FromContext(ctx)

	// Extract metadata
	kind := obj["kind"].(string)
	apiVersion := obj["apiVersion"].(string)
	metadata := obj["metadata"].(map[string]interface{})
	name := metadata["name"].(string)
	namespace := owner.Namespace

	ref := appv1alpha1.ResourceRef{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
	}

	// Convert to unstructured
	u := &unstructured.Unstructured{Object: obj}
	u.SetNamespace(namespace)

	// Set owner reference
	u.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: owner.APIVersion,
		Kind:       owner.Kind,
		Name:       owner.Name,
		UID:        owner.UID,
		Controller: func() *bool { b := true; return &b }(),
	}})

	// Check if resource exists
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(u.GroupVersionKind())
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing)

	if errors.IsNotFound(err) {
		// Create new resource
		if err := r.Create(ctx, u); err != nil {
			return ref, fmt.Errorf("failed to create %s/%s: %w", kind, name, err)
		}
		log.Info("Created resource", "kind", kind, "name", name)
	} else if err != nil {
		return ref, fmt.Errorf("failed to get %s/%s: %w", kind, name, err)
	} else {
		// Update existing resource
		u.SetResourceVersion(existing.GetResourceVersion())
		if err := r.Update(ctx, u); err != nil {
			return ref, fmt.Errorf("failed to update %s/%s: %w", kind, name, err)
		}
		log.Info("Updated resource", "kind", kind, "name", name)
	}

	return ref, nil
}

// updateStatus updates the HeliosApp status
func (r *HeliosAppReconciler) updateStatus(ctx context.Context, app *appv1alpha1.HeliosApp, phase, message string) {
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
