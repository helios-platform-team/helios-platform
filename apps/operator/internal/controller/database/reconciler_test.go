package database

import (
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 to scheme: %v", err)
	}
	if err := appv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add appv1alpha1 to scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add appsv1 to scheme: %v", err)
	}
	return scheme
}

func newTestReconciler(t *testing.T, objs ...client.Object) (*Reconciler, client.Client) {
	t.Helper()
	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()
	return NewReconciler(fakeClient, scheme), fakeClient
}

func TestReconcileSecrets(t *testing.T) {
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "mydb",
		"version": "18.4",
	}
	dbPropsJSON, _ := json.Marshal(dbProps)

	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name: "api-server",
					Type: "web-service",
					Traits: []appv1alpha1.Trait{
						{
							Type: "database",
							Properties: &runtime.RawExtension{
								Raw: dbPropsJSON,
							},
						},
					},
				},
			},
		},
	}

	t.Run("CreateNewSecret", func(t *testing.T) {
		r, fakeClient := newTestReconciler(t, app)

		ctx := t.Context()
		err := r.ReconcileSecrets(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileSecrets failed: %v", err)
		}

		secret := &corev1.Secret{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "api-server-db-secret",
			Namespace: "default",
		}, secret)
		if err != nil {
			t.Fatalf("Failed to get created secret: %v", err)
		}

		if _, ok := secret.Data["DB_USER"]; !ok {
			t.Error("Secret missing DB_USER key")
		}
		if _, ok := secret.Data["DB_PASS"]; !ok {
			t.Error("Secret missing DB_PASS key")
		}
		if _, ok := secret.Data["DB_HOST"]; !ok {
			t.Error("Secret missing DB_HOST key")
		}
	})

	t.Run("ExistingSecretPreserved", func(t *testing.T) {
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-server-db-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"DB_USER": []byte("existing-user"),
				"DB_PASS": []byte("existing-pass"),
				"DB_HOST": []byte("api-server-db"),
			},
		}

		r, fakeClient := newTestReconciler(t, app, existingSecret)

		ctx := t.Context()
		err := r.ReconcileSecrets(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileSecrets failed: %v", err)
		}

		secret := &corev1.Secret{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "api-server-db-secret",
			Namespace: "default",
		}, secret)
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if string(secret.Data["DB_USER"]) != "existing-user" {
			t.Errorf("Expected existing DB_USER to be preserved, got %s", string(secret.Data["DB_USER"]))
		}
		if string(secret.Data["DB_PASS"]) != "existing-pass" {
			t.Errorf("Expected existing DB_PASS to be preserved, got %s", string(secret.Data["DB_PASS"]))
		}
		if string(secret.Data["DB_HOST"]) != "api-server-db" {
			t.Errorf("Expected existing DB_HOST to be preserved, got %s", string(secret.Data["DB_HOST"]))
		}
	})

	t.Run("NoDatabaseTraits", func(t *testing.T) {
		appWithoutDB := &appv1alpha1.HeliosApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-db-app",
				Namespace: "default",
				UID:       "test-uid-456",
			},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{
					{
						Name: "frontend",
						Type: "web-service",
						Traits: []appv1alpha1.Trait{
							{
								Type: "service",
								Properties: &runtime.RawExtension{
									Raw: []byte(`{"port": 3000}`),
								},
							},
						},
					},
				},
			},
		}

		r, fakeClient := newTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		err := r.ReconcileSecrets(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("ReconcileSecrets should not fail for app without database traits: %v", err)
		}

		secretList := &corev1.SecretList{}
		err = fakeClient.List(ctx, secretList)
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if len(secretList.Items) != 0 {
			t.Errorf("Expected no secrets, got %d", len(secretList.Items))
		}
	})
}

func TestReconcileInstances(t *testing.T) {
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "my_custom_db",
		"version": "18.4",
		"storage": "2Gi",
	}
	dbPropsJSON, _ := json.Marshal(dbProps)

	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			UID:       "test-uid-789",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name: "api-server",
					Type: "web-service",
					Traits: []appv1alpha1.Trait{
						{
							Type: "database",
							Properties: &runtime.RawExtension{
								Raw: dbPropsJSON,
							},
						},
					},
				},
			},
		},
	}

	t.Run("CreatesStatefulSetAndService", func(t *testing.T) {
		r, fakeClient := newTestReconciler(t, app)

		ctx := t.Context()
		err := r.ReconcileInstances(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileInstances failed: %v", err)
		}

		stsList := &appsv1.StatefulSetList{}
		err = fakeClient.List(ctx, stsList)
		if err != nil {
			t.Fatalf("Failed to list StatefulSets: %v", err)
		}
		if len(stsList.Items) != 1 {
			t.Fatalf("Expected 1 StatefulSet, got %d", len(stsList.Items))
		}

		sts := stsList.Items[0]
		if sts.Name != "api-server-db" {
			t.Errorf("Expected StatefulSet name %q, got %q", "api-server-db", sts.Name)
		}

		containers := sts.Spec.Template.Spec.Containers
		if len(containers) != 1 {
			t.Fatalf("Expected 1 container, got %d", len(containers))
		}
		foundDB := false
		for _, env := range containers[0].Env {
			if env.Name == "POSTGRES_DB" && env.Value == "my_custom_db" {
				foundDB = true
			}
		}
		if !foundDB {
			t.Error("POSTGRES_DB env var not found with expected value")
		}

		svcList := &corev1.ServiceList{}
		err = fakeClient.List(ctx, svcList)
		if err != nil {
			t.Fatalf("Failed to list Services: %v", err)
		}
		if len(svcList.Items) != 1 {
			t.Fatalf("Expected 1 Service, got %d", len(svcList.Items))
		}
		if svcList.Items[0].Spec.ClusterIP != "None" {
			t.Errorf("Expected headless Service (clusterIP: None), got %q", svcList.Items[0].Spec.ClusterIP)
		}
	})

	t.Run("SkipsWhenNoTraits", func(t *testing.T) {
		appWithoutDB := &appv1alpha1.HeliosApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-db-app",
				Namespace: "default",
				UID:       "test-uid-no-db",
			},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{
					{Name: "frontend", Type: "web-service"},
				},
			},
		}

		r, fakeClient := newTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		err := r.ReconcileInstances(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("ReconcileInstances should not fail for app without database traits: %v", err)
		}

		stsList := &appsv1.StatefulSetList{}
		_ = fakeClient.List(ctx, stsList)
		if len(stsList.Items) != 0 {
			t.Errorf("Expected no StatefulSets, got %d", len(stsList.Items))
		}

		svcList := &corev1.ServiceList{}
		_ = fakeClient.List(ctx, svcList)
		if len(svcList.Items) != 0 {
			t.Errorf("Expected no Services, got %d", len(svcList.Items))
		}
	})

	t.Run("SkipsNonPostgresType", func(t *testing.T) {
		redisProps := map[string]any{"dbType": "redis", "version": "7"}
		redisPropsJSON, _ := json.Marshal(redisProps)

		appWithRedis := &appv1alpha1.HeliosApp{
			ObjectMeta: metav1.ObjectMeta{Name: "redis-app", Namespace: "default", UID: "test-uid-redis"},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{
					{
						Name: "cache",
						Type: "web-service",
						Traits: []appv1alpha1.Trait{
							{Type: "database", Properties: &runtime.RawExtension{Raw: redisPropsJSON}},
						},
					},
				},
			},
		}

		r, fakeClient := newTestReconciler(t, appWithRedis)

		ctx := t.Context()
		err := r.ReconcileInstances(ctx, appWithRedis)
		if err != nil {
			t.Fatalf("ReconcileInstances should not fail for redis type: %v", err)
		}

		stsList := &appsv1.StatefulSetList{}
		_ = fakeClient.List(ctx, stsList)
		if len(stsList.Items) != 0 {
			t.Errorf("Expected no StatefulSets for redis type, got %d", len(stsList.Items))
		}
	})

	t.Run("UpdatesExistingStatefulSetAndService", func(t *testing.T) {
		existingSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec: appsv1.StatefulSetSpec{
				Replicas: func() *int32 { return new(int32(1)) }(),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "postgres", Image: "postgres:18.4"}},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("2Gi")},
						},
					},
				}},
			},
		}

		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "db", Port: 15432}}},
		}

		r, fakeClient := newTestReconciler(t, app, existingSts, existingSvc)

		ctx := t.Context()
		err := r.ReconcileInstances(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileInstances failed: %v", err)
		}

		updatedSts := &appsv1.StatefulSet{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: "api-server-db", Namespace: app.Namespace}, updatedSts)
		if err != nil {
			t.Fatalf("failed to get updated StatefulSet: %v", err)
		}
		if got := updatedSts.Spec.Template.Spec.Containers[0].Image; got != "postgres:18.4" {
			t.Fatalf("expected image postgres:18.4, got %s", got)
		}

		updatedSvc := &corev1.Service{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: "api-server-db", Namespace: app.Namespace}, updatedSvc)
		if err != nil {
			t.Fatalf("failed to get updated Service: %v", err)
		}
		if got := updatedSvc.Spec.Ports[0].Port; got != 5432 {
			t.Fatalf("expected service port 5432, got %d", got)
		}
	})

	t.Run("FailsOnStatefulSetStorageDrift", func(t *testing.T) {
		existingSts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec: appsv1.StatefulSetSpec{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
						},
					},
				}},
			},
		}

		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "db", Port: 5432}}},
		}

		r, _ := newTestReconciler(t, app, existingSts, existingSvc)

		err := r.ReconcileInstances(t.Context(), app)
		if err == nil {
			t.Fatal("expected storage drift error, got nil")
		}
		if !strings.Contains(err.Error(), "storage drift detected") {
			t.Fatalf("expected storage drift error, got %v", err)
		}
	})
}

func TestReconcileInjection(t *testing.T) {
	dbProps := map[string]any{"dbType": "postgres", "dbName": "mydb", "version": "18.4"}
	dbPropsJSON, _ := json.Marshal(dbProps)

	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app", Namespace: "default", UID: "test-uid-inject"},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name: "api-server",
					Type: "web-service",
					Traits: []appv1alpha1.Trait{
						{Type: "database", Properties: &runtime.RawExtension{Raw: dbPropsJSON}},
					},
				},
			},
		},
	}

	t.Run("InjectsIntoExistingDeployment", func(t *testing.T) {
		existingDeploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "api-server"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api-server"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env:   []corev1.EnvVar{{Name: "PORT", Value: "3000"}},
							},
						},
					},
				},
			},
		}

		r, fakeClient := newTestReconciler(t, app, existingDeploy)

		ctx := t.Context()
		pending, err := r.ReconcileInjection(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileInjection failed: %v", err)
		}
		if pending {
			t.Error("Expected no pending injection when Deployment exists")
		}

		updatedDeploy := &appsv1.Deployment{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: "api-server", Namespace: "default"}, updatedDeploy)
		if err != nil {
			t.Fatalf("Failed to get updated Deployment: %v", err)
		}

		container := updatedDeploy.Spec.Template.Spec.Containers[0]
		expectedEnvNames := map[string]bool{
			"DB_HOST": false, "DB_USER": false, "DB_PASS": false,
			"DB_PORT": false, "DB_NAME": false, "DATABASE_URL": false,
		}
		for _, env := range container.Env {
			if _, ok := expectedEnvNames[env.Name]; ok {
				expectedEnvNames[env.Name] = true
				switch env.Name {
				case "DB_HOST", "DB_USER", "DB_PASS":
					if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
						t.Errorf("Env %s should reference a secret", env.Name)
					} else if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
						t.Errorf("Env %s: expected secret name %q, got %q",
							env.Name, "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
					}
				case "DB_PORT":
					if env.Value != "5432" || env.ValueFrom != nil {
						t.Errorf("Env DB_PORT: expected literal 5432, got %+v", env)
					}
				case "DB_NAME":
					if env.Value != "mydb" || env.ValueFrom != nil {
						t.Errorf("Env DB_NAME: expected literal mydb, got %+v", env)
					}
				case "DATABASE_URL":
					if env.Value != postgresDatabaseURLTemplate || env.ValueFrom != nil {
						t.Errorf("Env DATABASE_URL: expected template value, got %+v", env)
					}
				}
			}
		}
		for name, found := range expectedEnvNames {
			if !found {
				t.Errorf("Expected env var %s not found in Deployment", name)
			}
		}
	})

	t.Run("SkipsWhenNoTraits", func(t *testing.T) {
		appWithoutDB := &appv1alpha1.HeliosApp{
			ObjectMeta: metav1.ObjectMeta{Name: "no-db-app", Namespace: "default", UID: "test-uid-no-inject"},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{{Name: "frontend", Type: "web-service"}},
			},
		}

		r, _ := newTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		pending, err := r.ReconcileInjection(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("ReconcileInjection should not fail for app without database traits: %v", err)
		}
		if pending {
			t.Error("Expected no pending injection for app without database traits")
		}
	})

	t.Run("DeploymentNotFound_GracefulSkip", func(t *testing.T) {
		r, _ := newTestReconciler(t, app)

		ctx := t.Context()
		pending, err := r.ReconcileInjection(ctx, app)
		if err != nil {
			t.Fatalf("ReconcileInjection should not fail when Deployment is missing: %v", err)
		}
		if !pending {
			t.Error("Expected pending=true when Deployment is missing")
		}
	})
}
