package database

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

// Reconciler handles database-related reconciliation (secrets, instances, injection).
type Reconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// NewReconciler creates a new database Reconciler.
func NewReconciler(c client.Client, scheme *runtime.Scheme) *Reconciler {
	return &Reconciler{Client: c, Scheme: scheme}
}

// ReconcileSystemSecrets copies system-level secrets (docker-credentials, etc.)
// from the default namespace to the app's namespace. This ensures that Tekton
// tasks have access to required secrets for image building and pushing.
func (r *Reconciler) ReconcileSystemSecrets(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	// System secrets to provision to each app namespace
	systemSecrets := []string{"docker-credentials", "helios-gitops-bot"}

	for _, secretName := range systemSecrets {
		// Check if secret already exists in app namespace
		appSecret := &corev1.Secret{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: app.Namespace,
		}, appSecret)

		if err == nil {
			log.V(1).Info("System secret already exists in app namespace, skipping",
				"secret", secretName,
				"namespace", app.Namespace)
			continue
		}

		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check for system secret in app namespace",
				"secret", secretName,
				"namespace", app.Namespace)
			return fmt.Errorf("failed to check for system secret %s in namespace %s: %w", secretName, app.Namespace, err)
		}

		// Read secret from default namespace
		defaultSecret := &corev1.Secret{}
		err = r.Client.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: "default",
		}, defaultSecret)

		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("required system secret %s not found in default namespace: %w", secretName, err)
			}
			log.Error(err, "Failed to read system secret from default namespace",
				"secret", secretName)
			return fmt.Errorf("failed to read system secret %s from default namespace: %w", secretName, err)
		}

		// Copy secret to app namespace
		newSecret := r.copySecret(defaultSecret, app.Namespace)

		// Note: Do NOT set controller reference for system secrets
		// They are not "owned" by the app and should persist if the app is deleted

		if err := r.Client.Create(ctx, newSecret); err != nil {
			if errors.IsAlreadyExists(err) {
				log.Info("System secret was created concurrently, skipping",
					"secret", secretName,
					"namespace", app.Namespace)
				continue
			}
			log.Error(err, "Failed to create system secret in app namespace",
				"secret", secretName,
				"namespace", app.Namespace)
			return fmt.Errorf("failed to create system secret %s in namespace %s: %w", secretName, app.Namespace, err)
		}

		log.Info("Successfully provisioned system secret to app namespace",
			"secret", secretName,
			"namespace", app.Namespace)
	}

	return nil
}

// copySecret creates a copy of a secret with a new namespace.
// It preserves the secret type and data, but resets metadata.
func (r *Reconciler) copySecret(source *corev1.Secret, targetNamespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      source.Name,
			Namespace: targetNamespace,
			Labels: map[string]string{
				"helios.io/managed-by":    "operator",
				"helios.io/system-secret": "true",
			},
		},
		Type: source.Type,
		Data: source.Data,
	}
}

// ReconcileSecrets ensures database credential secrets exist for all
// components with database traits. If a secret already exists, it is not
// modified to preserve existing credentials.
func (r *Reconciler) ReconcileSecrets(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	dbTraits := ExtractDatabaseTraits(app)
	if len(dbTraits) == 0 {
		log.V(1).Info("No database traits found, skipping secret creation")
		return nil
	}

	for _, dbTrait := range dbTraits {
		if strings.ToLower(dbTrait.Properties.DBType) != dbTypePostgres {
			log.V(1).Info("Skipping credential secret creation for non-postgres database type",
				"component", dbTrait.ComponentName,
				"dbType", dbTrait.Properties.DBType)
			continue
		}

		secretName := GetDatabaseSecretName(dbTrait.ComponentName)
		dbHost := GetDatabaseHost(dbTrait.ComponentName)

		existingSecret := &corev1.Secret{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: app.Namespace,
		}, existingSecret)

		if err == nil {
			if validateErr := ValidateDatabaseSecret(existingSecret, dbHost); validateErr != nil {
				log.Error(validateErr, "Existing database secret is missing required keys",
					"component", dbTrait.ComponentName,
					"secret", secretName)
				return fmt.Errorf("existing database secret %s is invalid: %w", secretName, validateErr)
			}

			log.Info("Database secret already exists, skipping",
				"component", dbTrait.ComponentName,
				"secret", secretName)
			continue
		}

		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check for existing database secret",
				"component", dbTrait.ComponentName,
				"secret", secretName)
			return fmt.Errorf("failed to check for database secret %s: %w", secretName, err)
		}

		log.Info("Generating database credentials",
			"component", dbTrait.ComponentName,
			"secret", secretName)

		creds, err := GenerateCredentials()
		if err != nil {
			log.Error(err, "Failed to generate database credentials",
				"component", dbTrait.ComponentName)
			return fmt.Errorf("failed to generate credentials for %s: %w", dbTrait.ComponentName, err)
		}

		// Compute effective database name and port
		effectiveDBName := dbTrait.Properties.DBName
		if effectiveDBName == "" {
			effectiveDBName = fmt.Sprintf("%s-db", dbTrait.ComponentName)
		}

		effectivePort := dbTrait.Properties.Port
		if effectivePort <= 0 {
			effectivePort = DefaultPostgresPort
		}

		secret := GenerateDatabaseSecret(app.Namespace, secretName, dbTrait.ComponentName, creds, dbHost, effectiveDBName, int32(effectivePort))

		if err := ctrl.SetControllerReference(app, secret, r.Scheme); err != nil {
			log.Error(err, "Failed to set owner reference for database secret",
				"component", dbTrait.ComponentName,
				"secret", secretName)
			return fmt.Errorf("failed to set owner reference for secret %s: %w", secretName, err)
		}

		if err := r.Client.Create(ctx, secret); err != nil {
			if errors.IsAlreadyExists(err) {
				log.Info("Database secret was created concurrently, skipping",
					"component", dbTrait.ComponentName,
					"secret", secretName)
				continue
			}
			log.Error(err, "Failed to create database secret",
				"component", dbTrait.ComponentName,
				"secret", secretName)
			return fmt.Errorf("failed to create database secret %s: %w", secretName, err)
		}

		log.Info("Successfully created database secret",
			"component", dbTrait.ComponentName,
			"secret", secretName,
			"dbHost", dbHost,
			"effectiveDBName", effectiveDBName,
			"effectivePort", effectivePort)
	}

	return nil
}

// ReconcileInstances provisions database StatefulSets and headless Services
// for components with database traits.
func (r *Reconciler) ReconcileInstances(ctx context.Context, app *appv1alpha1.HeliosApp) error {
	log := logf.FromContext(ctx)

	dbTraits := ExtractDatabaseTraits(app)
	if len(dbTraits) == 0 {
		log.V(1).Info("No database traits found, skipping instance provisioning")
		return nil
	}

	for _, dbTrait := range dbTraits {
		if strings.ToLower(dbTrait.Properties.DBType) != dbTypePostgres {
			log.V(1).Info("Skipping non-postgres database type",
				"component", dbTrait.ComponentName,
				"dbType", dbTrait.Properties.DBType)
			continue
		}

		dbHost := GetDatabaseHost(dbTrait.ComponentName)
		secretName := GetDatabaseSecretName(dbTrait.ComponentName)

		effectiveDBName := EffectiveDatabaseName(dbTrait)

		version := dbTrait.Properties.Version
		if version == "" {
			version = DefaultPostgresVersion
		}

		port := dbTrait.Properties.Port
		if port <= 0 {
			port = DefaultPostgresPort
		}
		if port > 65535 {
			return fmt.Errorf("invalid port %d for component %s: port must be <= 65535", port, dbTrait.ComponentName)
		}

		storage := dbTrait.Properties.Storage
		if storage == "" {
			storage = DefaultDatabaseStorage
		}

		// --- StatefulSet ---
		sts, err := GenerateDatabaseStatefulSet(
			app.Namespace, dbHost, secretName, effectiveDBName, version, storage, int32(port),
		)
		if err != nil {
			log.Error(err, "Failed to generate database StatefulSet",
				"component", dbTrait.ComponentName, "storage", storage)
			return fmt.Errorf("failed to generate StatefulSet for %s: %w", dbHost, err)
		}

		if err := ctrl.SetControllerReference(app, sts, r.Scheme); err != nil {
			log.Error(err, "Failed to set owner reference for database StatefulSet",
				"component", dbTrait.ComponentName)
			return fmt.Errorf("failed to set owner reference for StatefulSet %s: %w", dbHost, err)
		}

		existingSts := &appsv1.StatefulSet{}
		err = r.Client.Get(ctx, types.NamespacedName{Name: dbHost, Namespace: app.Namespace}, existingSts)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to check for StatefulSet %s: %w", dbHost, err)
			}

			log.Info("Creating database StatefulSet",
				"component", dbTrait.ComponentName,
				"statefulset", dbHost,
				"image", fmt.Sprintf("postgres:%s", version))

			if err := r.Client.Create(ctx, sts); err != nil {
				if errors.IsAlreadyExists(err) {
					log.Info("Database StatefulSet was created concurrently, skipping",
						"component", dbTrait.ComponentName)
				} else {
					return fmt.Errorf("failed to create StatefulSet %s: %w", dbHost, err)
				}
			}
		} else {
			log.Info("Database StatefulSet already exists, updating if necessary",
				"component", dbTrait.ComponentName,
				"statefulset", dbHost)

			currentStorage, storageErr := getCurrentStorageRequest(existingSts)
			if storageErr != nil {
				return fmt.Errorf("failed to read StatefulSet %s storage: %w", dbHost, storageErr)
			}

			desiredStorageQty, parseErr := resource.ParseQuantity(storage)
			if parseErr != nil {
				return fmt.Errorf("invalid desired storage for StatefulSet %s: %w", dbHost, parseErr)
			}

			if currentStorage.Cmp(desiredStorageQty) != 0 {
				return fmt.Errorf("database storage drift detected for %s: current=%s desired=%s; StatefulSet volume expansion is not handled automatically", dbHost, currentStorage.String(), desiredStorageQty.String())
			}

			updatedSts := existingSts.DeepCopy()
			updatedSts.Spec.Replicas = sts.Spec.Replicas
			updatedSts.Spec.Template = sts.Spec.Template
			updatedSts.Spec.VolumeClaimTemplates = existingSts.Spec.VolumeClaimTemplates

			if err := r.Client.Update(ctx, updatedSts); err != nil {
				return fmt.Errorf("failed to update StatefulSet %s: %w", dbHost, err)
			}
		}

		// --- Headless Service ---
		svc := GenerateDatabaseService(app.Namespace, dbHost, int32(port))

		if err := ctrl.SetControllerReference(app, svc, r.Scheme); err != nil {
			log.Error(err, "Failed to set owner reference for database Service",
				"component", dbTrait.ComponentName)
			return fmt.Errorf("failed to set owner reference for Service %s: %w", dbHost, err)
		}

		existingSvc := &corev1.Service{}
		err = r.Client.Get(ctx, types.NamespacedName{Name: dbHost, Namespace: app.Namespace}, existingSvc)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to check for Service %s: %w", dbHost, err)
			}

			log.Info("Creating database headless Service",
				"component", dbTrait.ComponentName,
				"service", dbHost)

			if err := r.Client.Create(ctx, svc); err != nil {
				if errors.IsAlreadyExists(err) {
					log.Info("Database Service was created concurrently, skipping",
						"component", dbTrait.ComponentName)
				} else {
					return fmt.Errorf("failed to create Service %s: %w", dbHost, err)
				}
			}
		} else {
			log.Info("Database Service already exists, updating if necessary",
				"component", dbTrait.ComponentName,
				"service", dbHost)

			updatedSvc := existingSvc.DeepCopy()
			updatedSvc.Spec.Ports = svc.Spec.Ports

			if err := r.Client.Update(ctx, updatedSvc); err != nil {
				return fmt.Errorf("failed to update Service %s: %w", dbHost, err)
			}
		}

		log.Info("Successfully reconciled database instance",
			"component", dbTrait.ComponentName,
			"statefulset", dbHost,
			"dbName", effectiveDBName)
	}

	return nil
}

// ReconcileInjection patches live Deployments (deployed by ArgoCD) to inject
// DB_HOST, DB_USER, DB_PASS env vars from the operator-managed Secret.
//
// Returns (pendingInjection, error). pendingInjection is true when one or more
// target Deployments do not exist yet, signaling the caller to requeue.
func (r *Reconciler) ReconcileInjection(ctx context.Context, app *appv1alpha1.HeliosApp) (bool, error) {
	log := logf.FromContext(ctx)

	dbTraits := ExtractDatabaseTraits(app)
	if len(dbTraits) == 0 {
		log.V(1).Info("No database traits found, skipping secret injection")
		return false, nil
	}

	pendingInjection := false

	for _, dbTrait := range dbTraits {
		if strings.ToLower(dbTrait.Properties.DBType) != dbTypePostgres {
			log.V(1).Info("Skipping env var injection for non-postgres database type",
				"component", dbTrait.ComponentName,
				"dbType", dbTrait.Properties.DBType)
			continue
		}

		secretName := GetDatabaseSecretName(dbTrait.ComponentName)
		deployName := dbTrait.ComponentName

		deploy := &appsv1.Deployment{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      deployName,
			Namespace: app.Namespace,
		}, deploy)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Deployment not found yet, will requeue for injection",
					"component", dbTrait.ComponentName,
					"deployment", deployName)
				pendingInjection = true
				continue
			}
			return false, fmt.Errorf("failed to get Deployment %s: %w", deployName, err)
		}

		port := dbTrait.Properties.Port
		if port <= 0 {
			port = DefaultPostgresPort
		}

		dbName := EffectiveDatabaseName(dbTrait)
		changed, exactContainerMatch := InjectDatabaseEnvVarsForContainer(deploy, secretName, dbTrait.ComponentName, int32(port), dbName, dbTrait.Properties.DBType)
		if !exactContainerMatch {
			fallbackContainer := "<no-containers>"
			if len(deploy.Spec.Template.Spec.Containers) > 0 {
				fallbackContainer = deploy.Spec.Template.Spec.Containers[0].Name
			}
			log.Info("Preferred application container not found, using first container for DB env injection",
				"component", dbTrait.ComponentName,
				"deployment", deployName,
				"preferredContainer", dbTrait.ComponentName,
				"fallbackContainer", fallbackContainer)
		}

		if !changed {
			log.V(1).Info("Database env vars already injected, skipping",
				"component", dbTrait.ComponentName,
				"deployment", deployName)
			continue
		}

		if err := r.Client.Update(ctx, deploy); err != nil {
			return false, fmt.Errorf("failed to update Deployment %s with database env vars: %w", deployName, err)
		}

		log.Info("Successfully injected database env vars into Deployment",
			"component", dbTrait.ComponentName,
			"deployment", deployName,
			"secret", secretName)
	}

	return pendingInjection, nil
}

func getCurrentStorageRequest(sts *appsv1.StatefulSet) (resource.Quantity, error) {
	if sts == nil {
		return resource.Quantity{}, fmt.Errorf("statefulset is nil")
	}

	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		return resource.Quantity{}, fmt.Errorf("no volumeClaimTemplates found")
	}

	qty, ok := sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]
	if !ok {
		return resource.Quantity{}, fmt.Errorf("storage request is missing")
	}

	return qty, nil
}
