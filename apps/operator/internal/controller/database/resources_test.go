package database

import (
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

func TestGenerateDatabaseSecret(t *testing.T) {
	namespace := "test-namespace"
	secretName := "my-app-db-secret"
	componentName := "my-app"
	dbHost := "my-app-db"

	creds := &DatabaseCredentials{
		Username: "testuser",
		Password: "testpassword123",
	}

	secret := GenerateDatabaseSecret(namespace, secretName, componentName, creds, dbHost)

	if secret.Name != secretName {
		t.Errorf("Expected secret name %q, got %q", secretName, secret.Name)
	}

	if secret.Namespace != namespace {
		t.Errorf("Expected namespace %q, got %q", namespace, secret.Namespace)
	}

	expectedLabels := map[string]string{
		"app":                   componentName,
		"helios.io/managed-by":  "operator",
		"helios.io/secret-type": "database-credentials",
	}
	for k, v := range expectedLabels {
		if secret.Labels[k] != v {
			t.Errorf("Expected label %s=%s, got %s", k, v, secret.Labels[k])
		}
	}

	if string(secret.Data["DB_USER"]) != creds.Username {
		t.Errorf("Expected DB_USER %q, got %q", creds.Username, string(secret.Data["DB_USER"]))
	}

	if string(secret.Data["DB_PASS"]) != creds.Password {
		t.Errorf("Expected DB_PASS %q, got %q", creds.Password, string(secret.Data["DB_PASS"]))
	}

	if string(secret.Data["DB_HOST"]) != dbHost {
		t.Errorf("Expected DB_HOST %q, got %q", dbHost, string(secret.Data["DB_HOST"]))
	}

	if secret.Type != corev1.SecretTypeOpaque {
		t.Errorf("Expected secret type %v, got %v", corev1.SecretTypeOpaque, secret.Type)
	}
}

func TestGetDatabaseSecretName(t *testing.T) {
	tests := []struct {
		componentName string
		expected      string
	}{
		{"my-app", "my-app-db-secret"},
		{"api-server", "api-server-db-secret"},
		{"backend", "backend-db-secret"},
	}

	for _, tt := range tests {
		t.Run(tt.componentName, func(t *testing.T) {
			result := GetDatabaseSecretName(tt.componentName)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetDatabaseHost(t *testing.T) {
	tests := []struct {
		componentName string
		expected      string
	}{
		{"my-app", "my-app-db"},
		{"api-server", "api-server-db"},
		{"backend", "backend-db"},
	}

	for _, tt := range tests {
		t.Run(tt.componentName, func(t *testing.T) {
			result := GetDatabaseHost(tt.componentName)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractDatabaseTraits(t *testing.T) {
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "mydb",
		"version": "18.3",
	}
	dbPropsJSON, _ := json.Marshal(dbProps)

	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name: "api-server",
					Type: "web-service",
					Traits: []appv1alpha1.Trait{
						{
							Type: "service",
							Properties: &runtime.RawExtension{
								Raw: []byte(`{"port": 8080}`),
							},
						},
						{
							Type: "database",
							Properties: &runtime.RawExtension{
								Raw: dbPropsJSON,
							},
						},
					},
				},
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

	dbTraits := ExtractDatabaseTraits(app)

	if len(dbTraits) != 1 {
		t.Fatalf("Expected 1 database trait, got %d", len(dbTraits))
	}

	trait := dbTraits[0]
	if trait.ComponentName != "api-server" {
		t.Errorf("Expected component name %q, got %q", "api-server", trait.ComponentName)
	}

	if trait.Properties.DBType != "postgres" {
		t.Errorf("Expected dbType %q, got %q", "postgres", trait.Properties.DBType)
	}

	if trait.Properties.DBName != "mydb" {
		t.Errorf("Expected dbName %q, got %q", "mydb", trait.Properties.DBName)
	}
}

func TestGenerateDatabaseStatefulSet(t *testing.T) {
	sts, err := GenerateDatabaseStatefulSet(
		"test-ns", "my-app-db", "my-app-db-secret",
		"my_custom_db", "18.3", "2Gi", 5432,
	)

	if err != nil {
		t.Fatalf("GenerateDatabaseStatefulSet failed: %v", err)
	}

	if sts.Name != "my-app-db" {
		t.Errorf("Expected name %q, got %q", "my-app-db", sts.Name)
	}
	if sts.Namespace != "test-ns" {
		t.Errorf("Expected namespace %q, got %q", "test-ns", sts.Namespace)
	}

	if sts.Labels["helios.io/db-type"] != "postgres" {
		t.Errorf("Expected db-type label %q, got %q", "postgres", sts.Labels["helios.io/db-type"])
	}
	if sts.Labels["helios.io/trait"] != "database" {
		t.Errorf("Expected trait label %q, got %q", "database", sts.Labels["helios.io/trait"])
	}

	if *sts.Spec.Replicas != 1 {
		t.Errorf("Expected 1 replica, got %d", *sts.Spec.Replicas)
	}

	if sts.Spec.ServiceName != "my-app-db" {
		t.Errorf("Expected serviceName %q, got %q", "my-app-db", sts.Spec.ServiceName)
	}

	containers := sts.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}

	container := containers[0]
	if container.Image != "postgres:18.3" {
		t.Errorf("Expected image %q, got %q", "postgres:18.3", container.Image)
	}

	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 5432 {
		t.Errorf("Expected container port 5432, got %v", container.Ports)
	}

	foundPostgresDB := false
	for _, env := range container.Env {
		if env.Name == "POSTGRES_DB" {
			foundPostgresDB = true
			if env.Value != "my_custom_db" {
				t.Errorf("Expected POSTGRES_DB value %q, got %q", "my_custom_db", env.Value)
			}
		}
		if env.Name == "POSTGRES_USER" {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				t.Error("POSTGRES_USER should reference a secret")
			} else {
				if env.ValueFrom.SecretKeyRef.Name != "my-app-db-secret" {
					t.Errorf("Expected secret name %q, got %q",
						"my-app-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != "DB_USER" {
					t.Errorf("Expected secret key %q, got %q",
						"DB_USER", env.ValueFrom.SecretKeyRef.Key)
				}
			}
		}
		if env.Name == "POSTGRES_PASSWORD" {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				t.Error("POSTGRES_PASSWORD should reference a secret")
			} else {
				if env.ValueFrom.SecretKeyRef.Name != "my-app-db-secret" {
					t.Errorf("Expected secret name %q, got %q",
						"my-app-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != "DB_PASS" {
					t.Errorf("Expected secret key %q, got %q",
						"DB_PASS", env.ValueFrom.SecretKeyRef.Key)
				}
			}
		}
	}
	if !foundPostgresDB {
		t.Error("POSTGRES_DB env var not found in container")
	}

	foundPGDATA := false
	for _, env := range container.Env {
		if env.Name == "PGDATA" {
			foundPGDATA = true
			if env.Value != PostgresDataPath {
				t.Errorf("Expected PGDATA value %q, got %q", PostgresDataPath, env.Value)
			}
		}
	}
	if !foundPGDATA {
		t.Error("PGDATA env var not found in container")
	}

	foundInitDB := false
	for _, env := range container.Env {
		if env.Name == "POSTGRES_INITDB_ARGS" {
			foundInitDB = true
		}
	}
	if !foundInitDB {
		t.Error("POSTGRES_INITDB_ARGS env var not found in container")
	}

	foundPGPORT := false
	for _, env := range container.Env {
		if env.Name == "PGPORT" {
			foundPGPORT = true
			if env.Value != "5432" {
				t.Errorf("Expected PGPORT value %q, got %q", "5432", env.Value)
			}
		}
	}
	if !foundPGPORT {
		t.Error("PGPORT env var not found in container")
	}

	if container.LivenessProbe == nil {
		t.Error("LivenessProbe should be set on Postgres container")
	} else {
		cmdStr := strings.Join(container.LivenessProbe.Exec.Command, " ")
		if len(container.LivenessProbe.Exec.Command) != 3 || container.LivenessProbe.Exec.Command[0] != "sh" || container.LivenessProbe.Exec.Command[1] != "-c" {
			t.Errorf("LivenessProbe command should use shell expansion, got: %v", container.LivenessProbe.Exec.Command)
		}
		if !strings.Contains(cmdStr, `-p "$PGPORT"`) {
			t.Errorf("LivenessProbe command missing custom port flag. Got: %s", cmdStr)
		}
		if !strings.Contains(cmdStr, `-d "$POSTGRES_DB"`) {
			t.Errorf("LivenessProbe command should reference POSTGRES_DB env var. Got: %s", cmdStr)
		}
	}

	if container.ReadinessProbe == nil {
		t.Error("ReadinessProbe should be set on Postgres container")
	} else {
		cmdStr := strings.Join(container.ReadinessProbe.Exec.Command, " ")
		if len(container.ReadinessProbe.Exec.Command) != 3 || container.ReadinessProbe.Exec.Command[0] != "sh" || container.ReadinessProbe.Exec.Command[1] != "-c" {
			t.Errorf("ReadinessProbe command should use shell expansion, got: %v", container.ReadinessProbe.Exec.Command)
		}
		if !strings.Contains(cmdStr, `-p "$PGPORT"`) {
			t.Errorf("ReadinessProbe command missing custom port flag. Got: %s", cmdStr)
		}
		if !strings.Contains(cmdStr, `-d "$POSTGRES_DB"`) {
			t.Errorf("ReadinessProbe command should reference POSTGRES_DB env var. Got: %s", cmdStr)
		}
	}

	if len(sts.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("Expected 1 VolumeClaimTemplate, got %d", len(sts.Spec.VolumeClaimTemplates))
	}
	vct := sts.Spec.VolumeClaimTemplates[0]
	storageQty := vct.Spec.Resources.Requests[corev1.ResourceStorage]
	if storageQty.String() != "2Gi" {
		t.Errorf("Expected storage %q, got %q", "2Gi", storageQty.String())
	}
}

func TestGenerateDatabaseStatefulSet_InvalidStorage(t *testing.T) {
	_, err := GenerateDatabaseStatefulSet("default", "my-app-db", "my-app-db-secret", "my_custom_db", "16", "invalid-size", 5432)

	if err == nil {
		t.Fatal("Expected error for invalid storage size, got nil")
	}
	if !strings.Contains(err.Error(), "invalid storage size format") {
		t.Errorf("Expected error to mention invalid storage format, got %v", err)
	}
}

func TestGenerateDatabaseService(t *testing.T) {
	svc := GenerateDatabaseService("test-ns", "api-server-db", 5432)

	if svc.Name != "api-server-db" {
		t.Errorf("Expected name %q, got %q", "api-server-db", svc.Name)
	}
	if svc.Namespace != "test-ns" {
		t.Errorf("Expected namespace %q, got %q", "test-ns", svc.Namespace)
	}

	if svc.Spec.ClusterIP != "None" {
		t.Errorf("Expected clusterIP %q, got %q", "None", svc.Spec.ClusterIP)
	}

	if svc.Spec.Selector["app"] != "api-server-db" {
		t.Errorf("Expected selector app=%q, got %q", "api-server-db", svc.Spec.Selector["app"])
	}

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != 5432 {
		t.Errorf("Expected port 5432, got %d", svc.Spec.Ports[0].Port)
	}
	if svc.Spec.Ports[0].Name != "db" {
		t.Errorf("Expected port name %q, got %q", "db", svc.Spec.Ports[0].Name)
	}
}

func TestValidateDatabaseSecret(t *testing.T) {
	t.Run("ValidSecret", func(t *testing.T) {
		secret := &corev1.Secret{
			Data: map[string][]byte{
				"DB_USER": []byte("user"),
				"DB_PASS": []byte("pass"),
				"DB_HOST": []byte("api-server-db"),
			},
		}
		if err := ValidateDatabaseSecret(secret, "api-server-db"); err != nil {
			t.Fatalf("Expected secret to be valid, got %v", err)
		}
	})

	t.Run("MissingRequiredKeys", func(t *testing.T) {
		secret := &corev1.Secret{Data: map[string][]byte{"DB_USER": []byte("user")}}
		err := ValidateDatabaseSecret(secret, "api-server-db")
		if err == nil {
			t.Fatal("Expected validation error for missing keys")
		}
		if !strings.Contains(err.Error(), "missing required keys") {
			t.Fatalf("Expected missing keys error, got %v", err)
		}
	})

	t.Run("MismatchedHost", func(t *testing.T) {
		secret := &corev1.Secret{
			Data: map[string][]byte{
				"DB_USER": []byte("user"),
				"DB_PASS": []byte("pass"),
				"DB_HOST": []byte("wrong-host"),
			},
		}
		err := ValidateDatabaseSecret(secret, "api-server-db")
		if err == nil {
			t.Fatal("Expected validation error for host mismatch")
		}
		if !strings.Contains(err.Error(), "DB_HOST mismatch") {
			t.Fatalf("Expected host mismatch error, got %v", err)
		}
	})
}
