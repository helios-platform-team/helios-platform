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
			if trait.Type == "database" {
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

	if err := r.Client.Create(ctx, sa); err != nil {
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

	if err := r.Client.Create(ctx, role); err != nil {
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

	if err := r.Client.Create(ctx, binding); err != nil {
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
			"serviceAccountName":      fmt.Sprintf("%s-migrator", heliosApp.Name),
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":      heliosApp.Name,
						"job-type": "db-migration",
					},
				},
				"spec": map[string]interface{}{
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
											"name": fmt.Sprintf("%s-db-credentials", heliosApp.Name),
											"key":  "uri",
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
						},
					},
					"restartPolicy": "Never",
					"securityContext": map[string]interface{}{
						"runAsNonRoot":             true,
						"runAsUser":                1000,
						"fsReadOnlyRootFilesystem": true,
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

	if err := r.Client.Update(ctx, heliosAppCopy); err != nil {
		return fmt.Errorf("failed to update HeliosApp with PreSync Job config: %w", err)
	}

	return nil
}

// HasDatabaseTrait checks if the HeliosApp has any component with database trait.
func HasDatabaseTrait(heliosApp *appv1alpha1.HeliosApp) bool {
	for _, comp := range heliosApp.Spec.Components {
		for _, trait := range comp.Traits {
			if trait.Type == "database" {
				return true
			}
		}
	}
	return false
}
