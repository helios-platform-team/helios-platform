package argocd

import (
	"context"
	"encoding/json"
	"fmt"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// preSyncFinalizerKey is used to ensure cluster-scoped RBAC resources
	// (ClusterRole and ClusterRoleBinding) are properly cleaned up when a HeliosApp is deleted.
	preSyncFinalizerKey = "argocd.helios.io/presync-cleanup"

	// databaseTraitType is the type identifier for database traits in components.
	databaseTraitType = "database"
)

// PreSyncReconciler creates PreSync Jobs and supporting resources for database migrations.
type PreSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewPreSyncReconciler creates a new PreSyncReconciler.
func NewPreSyncReconciler(c client.Client, scheme *runtime.Scheme) *PreSyncReconciler {
	return &PreSyncReconciler{
		Client: c,
		Scheme: scheme,
	}
}

// ReconcilePreSyncResources creates PreSync Job, EventListener, and supporting RBAC
// when a HeliosApp has a database trait. This enables automatic database migrations
// before ArgoCD deploys the application.
func (r *PreSyncReconciler) ReconcilePreSyncResources(
	ctx context.Context,
	heliosApp *appv1alpha1.HeliosApp,
) error {
	log := logf.FromContext(ctx)

	// Check if any component has database trait
	hasDatabaseTrait := false
	for _, comp := range heliosApp.Spec.Components {
		for _, trait := range comp.Traits {
			if trait.Type == databaseTraitType {
				hasDatabaseTrait = true
				break
			}
		}
	}

	if !hasDatabaseTrait {
		log.Info("No database trait found, skipping PreSync resource creation")
		return nil
	}

	log.Info("Creating PreSync resources for database migrations", "app", heliosApp.Name)

	// Add finalizer to ensure cluster-scoped RBAC resources are cleaned up on deletion
	if err := r.AddPreSyncFinalizer(ctx, heliosApp); err != nil {
		return fmt.Errorf("failed to add presync finalizer: %w", err)
	}

	// 1. Create ServiceAccount for PreSync Job execution
	if err := r.reconcileServiceAccount(ctx, heliosApp); err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// 2. Create ClusterRole for Job management
	if err := r.reconcileRole(ctx, heliosApp); err != nil {
		return fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	// 3. Create ClusterRoleBinding
	if err := r.reconcileRoleBinding(ctx, heliosApp); err != nil {
		return fmt.Errorf("failed to create RoleBinding: %w", err)
	}

	// 4. Create PreSync Job template (not executed yet, just stored for ArgoCD)
	if err := r.reconcilePreSyncJobconfig(ctx, heliosApp); err != nil {
		return fmt.Errorf("failed to create PreSync Job config: %w", err)
	}

	log.Info("PreSync resources created successfully", "app", heliosApp.Name)
	return nil
}

// reconcileServiceAccount creates a ServiceAccount for PreSync Job execution.
func (r *PreSyncReconciler) reconcileServiceAccount(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-migrator", heliosApp.Name),
			Namespace: heliosApp.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/component":  "database-migration",
				"app.kubernetes.io/managed-by": "helios-operator",
			},
		},
	}

	if err := ctrl.SetControllerReference(heliosApp, sa, r.Scheme); err != nil {
		return err
	}

	if err := r.Create(ctx, sa); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil // Already exists, no error
		}
		return err
	}

	return nil
}

// reconcileRole creates a ClusterRole for Job and Pod management.
func (r *PreSyncReconciler) reconcileRole(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-presync-job-role", heliosApp.Name),
			Labels: map[string]string{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/managed-by": "helios-operator",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/log"},
				Verbs:     []string{"get"},
			},
		},
	}

	if err := r.Create(ctx, role); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

// reconcileRoleBinding creates a ClusterRoleBinding for the ServiceAccount.
func (r *PreSyncReconciler) reconcileRoleBinding(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-presync-job-binding", heliosApp.Name),
			Labels: map[string]string{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/managed-by": "helios-operator",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     fmt.Sprintf("%s-presync-job-role", heliosApp.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      fmt.Sprintf("%s-migrator", heliosApp.Name),
				Namespace: heliosApp.Namespace,
			},
		},
	}

	if err := r.Create(ctx, binding); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

// reconcilePreSyncJobconfig stores the PreSync Job configuration as a ConfigMap.
// This config will be referenced by the ArgoCD Application and executed as a PreSync hook.
func (r *PreSyncReconciler) reconcilePreSyncJobconfig(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	// Find the first component with database trait to get the correct database secret
	var databaseComponentName string
	for _, comp := range heliosApp.Spec.Components {
		for _, trait := range comp.Traits {
			if trait.Type == databaseTraitType {
				databaseComponentName = comp.Name
				break
			}
		}
		if databaseComponentName != "" {
			break
		}
	}

	if databaseComponentName == "" {
		log.V(1).Info("No database component found, skipping PreSync Job config creation")
		return nil
	}

	// The database secret is named {componentName}-db-secret following the operator's convention
	databaseSecretName := fmt.Sprintf("%s-db-secret", databaseComponentName)

	// Find migration image reference from components
	// For now, use the standard naming convention: <app-name>-migrate:latest
	migrateImage := fmt.Sprintf("index.docker.io/{{.Values.dockerOrg}}/%s-migrate:latest", heliosApp.Name)

	// Build PreSync Job YAML
	preSyncJob := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-db-migrate-presync", heliosApp.Name),
			"namespace": heliosApp.Namespace,
			"labels": map[string]interface{}{
				"app":                     heliosApp.Name,
				"job-type":                "db-migration",
				"argocd.argoproj.io/hook": "PreSync",
				"argocd.argoproj.io/hook-deletion-policy": "BeforeHookCreation",
			},
		},
		"spec": map[string]interface{}{
			"backoffLimit":            3,
			"ttlSecondsAfterFinished": 3600,
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":      heliosApp.Name,
						"job-type": "db-migration",
					},
				},
				"spec": map[string]interface{}{
					"serviceAccountName": fmt.Sprintf("%s-migrator", heliosApp.Name),
					"containers": []map[string]interface{}{
						{
							"name":            "db-migrate",
							"image":           migrateImage,
							"imagePullPolicy": "Always",
							"env": []map[string]interface{}{
								{
									"name": "PGRST_DB_URI",
									"valueFrom": map[string]interface{}{
										"secretKeyRef": map[string]interface{}{
											"name": databaseSecretName,
											"key":  "PGRST_DB_URI",
										},
									},
								},
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "128Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "500m",
									"memory": "512Mi",
								},
							},
							"securityContext": map[string]interface{}{
								"readOnlyRootFilesystem": true,
							},
						},
					},
					"restartPolicy": "Never",
					"securityContext": map[string]interface{}{
						"runAsNonRoot": true,
						"runAsUser":    1000,
					},
				},
			},
		},
	}

	// Store as annotation on HeliosApp for ArgoCD to reference
	jobBytes, err := json.Marshal(preSyncJob)
	if err != nil {
		return fmt.Errorf("failed to marshal PreSync Job: %w", err)
	}

	// Update HeliosApp with PreSync Job definition as annotation
	heliosAppCopy := heliosApp.DeepCopy()
	if heliosAppCopy.Annotations == nil {
		heliosAppCopy.Annotations = make(map[string]string)
	}
	heliosAppCopy.Annotations["helios.io/presync-job"] = string(jobBytes)
	heliosAppCopy.Annotations["helios.io/has-database-trait"] = "true"

	if err := r.Update(ctx, heliosAppCopy); err != nil {
		return fmt.Errorf("failed to update HeliosApp with PreSync Job config: %w", err)
	}

	return nil
}

// HasDatabaseTrait checks if the HeliosApp has any component with database trait.
func HasDatabaseTrait(heliosApp *appv1alpha1.HeliosApp) bool {
	for _, comp := range heliosApp.Spec.Components {
		for _, trait := range comp.Traits {
			if trait.Type == databaseTraitType {
				return true
			}
		}
	}
	return false
}

// AddPreSyncFinalizer adds the presync cleanup finalizer to the HeliosApp.
// This finalizer ensures cluster-scoped RBAC resources are properly cleaned up
// when the HeliosApp is deleted.
func (r *PreSyncReconciler) AddPreSyncFinalizer(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	// Check if finalizer already exists
	for _, finalizer := range heliosApp.Finalizers {
		if finalizer == preSyncFinalizerKey {
			return nil // Finalizer already added
		}
	}

	// Add finalizer
	heliosAppCopy := heliosApp.DeepCopy()
	heliosAppCopy.Finalizers = append(heliosAppCopy.Finalizers, preSyncFinalizerKey)

	if err := r.Update(ctx, heliosAppCopy); err != nil {
		return fmt.Errorf("failed to add presync finalizer: %w", err)
	}

	return nil
}

// HandlePreSyncCleanup cleans up cluster-scoped RBAC resources when HeliosApp is deleted.
// This method should be called when the HeliosApp has a deletion timestamp and the
// presync cleanup finalizer is present.
func (r *PreSyncReconciler) HandlePreSyncCleanup(ctx context.Context, heliosApp *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	// Delete ClusterRole
	roleName := fmt.Sprintf("%s-presync-job-role", heliosApp.Name)
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
	}

	if err := r.Delete(ctx, role); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ClusterRole '%s': %w", roleName, err)
		}
		log.Info("ClusterRole not found, skipping deletion", "name", roleName)
	} else {
		log.Info("Deleted ClusterRole", "name", roleName)
	}

	// Delete ClusterRoleBinding
	bindingName := fmt.Sprintf("%s-presync-job-binding", heliosApp.Name)
	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
	}

	if err := r.Delete(ctx, binding); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ClusterRoleBinding '%s': %w", bindingName, err)
		}
		log.Info("ClusterRoleBinding not found, skipping deletion", "name", bindingName)
	} else {
		log.Info("Deleted ClusterRoleBinding", "name", bindingName)
	}

	// Remove finalizer after cleanup
	heliosAppCopy := heliosApp.DeepCopy()
	finalizers := []string{}
	for _, finalizer := range heliosAppCopy.Finalizers {
		if finalizer != preSyncFinalizerKey {
			finalizers = append(finalizers, finalizer)
		}
	}
	heliosAppCopy.Finalizers = finalizers

	if err := r.Update(ctx, heliosAppCopy); err != nil {
		return fmt.Errorf("failed to remove presync finalizer: %w", err)
	}

	log.Info("Presync cleanup completed and finalizer removed", "app", heliosApp.Name)
	return nil
}

// HasPreSyncFinalizer checks if the HeliosApp has the presync cleanup finalizer.
func HasPreSyncFinalizer(heliosApp *appv1alpha1.HeliosApp) bool {
	for _, finalizer := range heliosApp.Finalizers {
		if finalizer == preSyncFinalizerKey {
			return true
		}
	}
	return false
}

// GetPreSyncFinalizerKey returns the presync cleanup finalizer key.
// This is exported for use by the main controller.
func GetPreSyncFinalizerKey() string {
	return preSyncFinalizerKey
}
