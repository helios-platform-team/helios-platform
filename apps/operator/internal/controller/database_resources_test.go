package controller

import (
	"encoding/base64"
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

func newControllerTestScheme(t *testing.T) *runtime.Scheme {
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

func newControllerTestReconciler(t *testing.T, objs ...client.Object) (*HeliosAppReconciler, client.Client) {
	t.Helper()

	scheme := newControllerTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		Build()

	return &HeliosAppReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}, fakeClient
}

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"short password", 8},
		{"long password", 64},
		{"standard length", DefaultPasswordLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateSecurePassword(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecurePassword failed: %v", err)
			}

			expectedLen := tt.length
			if expectedLen <= 0 {
				expectedLen = DefaultPasswordLength
			}

			if len(password) != expectedLen {
				t.Errorf("Expected password length %d, got %d", expectedLen, len(password))
			}

			// Verify characters are from the charset
			for _, c := range password {
				found := false
				for _, allowed := range PasswordCharset {
					if c == allowed {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Invalid character %c in password", c)
				}
			}
		})
	}
}

func TestGenerateSecureUsername(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"short username", 8},
		{"long username", 32},
		{"standard length", DefaultUsernameLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := GenerateSecureUsername(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureUsername failed: %v", err)
			}

			expectedLen := tt.length
			if expectedLen <= 0 {
				expectedLen = DefaultUsernameLength
			}

			if len(username) != expectedLen {
				t.Errorf("Expected username length %d, got %d", expectedLen, len(username))
			}

			// Verify first character is a letter (database requirement)
			firstChar := username[0]
			if firstChar < 'a' || firstChar > 'z' {
				t.Errorf("First character %c must be a lowercase letter", firstChar)
			}

			// Verify characters are from the charset
			for _, c := range username {
				found := false
				for _, allowed := range UsernameCharset {
					if c == rune(allowed) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Invalid character %c in username", c)
				}
			}
		})
	}
}

func TestGenerateCredentials(t *testing.T) {
	creds, err := GenerateCredentials()
	if err != nil {
		t.Fatalf("GenerateCredentials failed: %v", err)
	}

	if creds.Username == "" {
		t.Error("Username should not be empty")
	}

	if creds.Password == "" {
		t.Error("Password should not be empty")
	}

	if len(creds.Username) != DefaultUsernameLength {
		t.Errorf("Expected username length %d, got %d", DefaultUsernameLength, len(creds.Username))
	}

	if len(creds.Password) != DefaultPasswordLength {
		t.Errorf("Expected password length %d, got %d", DefaultPasswordLength, len(creds.Password))
	}
}

func TestGenerateCredentialsUniqueness(t *testing.T) {
	// Generate multiple credentials and ensure they are unique
	credentials := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		creds, err := GenerateCredentials()
		if err != nil {
			t.Fatalf("GenerateCredentials failed on iteration %d: %v", i, err)
		}

		key := creds.Username + ":" + creds.Password
		if credentials[key] {
			t.Errorf("Duplicate credentials generated on iteration %d", i)
		}
		credentials[key] = true
	}
}

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

	// Verify metadata
	if secret.Name != secretName {
		t.Errorf("Expected secret name %q, got %q", secretName, secret.Name)
	}

	if secret.Namespace != namespace {
		t.Errorf("Expected namespace %q, got %q", namespace, secret.Namespace)
	}

	// Verify labels
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

	// Verify secret data
	if string(secret.Data["DB_USER"]) != creds.Username {
		t.Errorf("Expected DB_USER %q, got %q", creds.Username, string(secret.Data["DB_USER"]))
	}

	if string(secret.Data["DB_PASS"]) != creds.Password {
		t.Errorf("Expected DB_PASS %q, got %q", creds.Password, string(secret.Data["DB_PASS"]))
	}

	if string(secret.Data["DB_HOST"]) != dbHost {
		t.Errorf("Expected DB_HOST %q, got %q", dbHost, string(secret.Data["DB_HOST"]))
	}

	// Verify secret type
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
	// Create a HeliosApp with database traits
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "mydb",
		"version": "16",
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

func TestReconcileDatabaseSecrets(t *testing.T) {
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "mydb",
		"version": "16",
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
		r, fakeClient := newControllerTestReconciler(t, app)

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets failed: %v", err)
		}

		// Verify secret was created
		secret := &corev1.Secret{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "api-server-db-secret",
			Namespace: "default",
		}, secret)
		if err != nil {
			t.Fatalf("Failed to get created secret: %v", err)
		}

		// Verify secret contains required keys
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

		r, fakeClient := newControllerTestReconciler(t, app, existingSecret)

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets failed: %v", err)
		}

		// Verify existing secret was not modified
		secret := &corev1.Secret{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "api-server-db-secret",
			Namespace: "default",
		}, secret)
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		// The existing secret should preserve the original values
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

		r, fakeClient := newControllerTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets should not fail for app without database traits: %v", err)
		}

		// Verify no secret was created
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

func TestGenerateBase64Token(t *testing.T) {
	tests := []struct {
		name       string
		byteLength int
	}{
		{"default", 0},
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateBase64Token(tt.byteLength)
			if err != nil {
				t.Fatalf("GenerateBase64Token failed: %v", err)
			}

			if token == "" {
				t.Error("Token should not be empty")
			}

			// Verify it's valid base64
			// Base64 length = ceil(byteLength * 8 / 6) * 6 / 8 * 4 / 3
			// Simplified: base64 length ≈ byteLength * 4/3, rounded up to multiple of 4
			expectedLen := tt.byteLength
			if expectedLen <= 0 {
				expectedLen = 32
			}
			// Base64 encoding produces 4 characters for every 3 bytes
			expectedBase64Len := ((expectedLen + 2) / 3) * 4
			if len(token) != expectedBase64Len {
				t.Errorf("Expected base64 length %d, got %d", expectedBase64Len, len(token))
			}

			decoded, decodeErr := base64.StdEncoding.DecodeString(token)
			if decodeErr != nil {
				t.Fatalf("Token is not valid base64: %v", decodeErr)
			}
			if len(decoded) != expectedLen {
				t.Errorf("Expected decoded token length %d, got %d", expectedLen, len(decoded))
			}
		})
	}
}

func TestGenerateDatabaseStatefulSet(t *testing.T) {
	sts, err := GenerateDatabaseStatefulSet(
		"test-ns", "my-app-db", "my-app-db-secret",
		"my_custom_db", "16", "2Gi", 5432,
	)

	if err != nil {
		t.Fatalf("GenerateDatabaseStatefulSet failed: %v", err)
	}

	// Verify metadata
	if sts.Name != "my-app-db" {
		t.Errorf("Expected name %q, got %q", "my-app-db", sts.Name)
	}
	if sts.Namespace != "test-ns" {
		t.Errorf("Expected namespace %q, got %q", "test-ns", sts.Namespace)
	}

	// Verify labels
	if sts.Labels["helios.io/db-type"] != "postgres" {
		t.Errorf("Expected db-type label %q, got %q", "postgres", sts.Labels["helios.io/db-type"])
	}
	if sts.Labels["helios.io/trait"] != "database" {
		t.Errorf("Expected trait label %q, got %q", "database", sts.Labels["helios.io/trait"])
	}

	// Verify replicas
	if *sts.Spec.Replicas != 1 {
		t.Errorf("Expected 1 replica, got %d", *sts.Spec.Replicas)
	}

	// Verify serviceName
	if sts.Spec.ServiceName != "my-app-db" {
		t.Errorf("Expected serviceName %q, got %q", "my-app-db", sts.Spec.ServiceName)
	}

	// Verify container
	containers := sts.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}

	container := containers[0]
	if container.Image != "postgres:16" {
		t.Errorf("Expected image %q, got %q", "postgres:16", container.Image)
	}

	// Verify ports
	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 5432 {
		t.Errorf("Expected container port 5432, got %v", container.Ports)
	}
	// Verify POSTGRES_DB env var (the core acceptance criteria)
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

	// Verify PGDATA env var
	foundPGDATA := false
	for _, env := range container.Env {
		if env.Name == "PGDATA" {
			foundPGDATA = true
			expectedPGDATA := PostgresDataPath
			if env.Value != expectedPGDATA {
				t.Errorf("Expected PGDATA value %q, got %q", expectedPGDATA, env.Value)
			}
		}
	}
	if !foundPGDATA {
		t.Error("PGDATA env var not found in container")
	}

	// Verify POSTGRES_INITDB_ARGS env var
	foundInitDB := false
	for _, env := range container.Env {
		if env.Name == "POSTGRES_INITDB_ARGS" {
			foundInitDB = true
		}
	}
	if !foundInitDB {
		t.Error("POSTGRES_INITDB_ARGS env var not found in container")
	}

	// Verify PGPORT env var
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

	// Verify livenessProbe exists and uses custom port
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

	// Verify readinessProbe uses custom port
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

	// Verify volume claim template
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

	// Verify metadata
	if svc.Name != "api-server-db" {
		t.Errorf("Expected name %q, got %q", "api-server-db", svc.Name)
	}
	if svc.Namespace != "test-ns" {
		t.Errorf("Expected namespace %q, got %q", "test-ns", svc.Namespace)
	}

	// Verify headless (clusterIP: None)
	if svc.Spec.ClusterIP != "None" {
		t.Errorf("Expected clusterIP %q, got %q", "None", svc.Spec.ClusterIP)
	}

	// Verify selector
	if svc.Spec.Selector["app"] != "api-server-db" {
		t.Errorf("Expected selector app=%q, got %q", "api-server-db", svc.Spec.Selector["app"])
	}

	// Verify port
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

func TestReconcileDatabaseInstance(t *testing.T) {

	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "my_custom_db",
		"version": "16",
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
		r, fakeClient := newControllerTestReconciler(t, app)

		ctx := t.Context()
		err := r.reconcileDatabaseInstance(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseInstance failed: %v", err)
		}

		// Verify StatefulSet was created
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

		// Verify POSTGRES_DB env var
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

		// Verify headless Service was created
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
					{
						Name: "frontend",
						Type: "web-service",
					},
				},
			},
		}

		r, fakeClient := newControllerTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		err := r.reconcileDatabaseInstance(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("reconcileDatabaseInstance should not fail for app without database traits: %v", err)
		}

		// Verify no StatefulSet or Service was created
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
		redisProps := map[string]any{
			"dbType":  "redis",
			"version": "7",
		}
		redisPropsJSON, _ := json.Marshal(redisProps)

		appWithRedis := &appv1alpha1.HeliosApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-app",
				Namespace: "default",
				UID:       "test-uid-redis",
			},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{
					{
						Name: "cache",
						Type: "web-service",
						Traits: []appv1alpha1.Trait{
							{
								Type: "database",
								Properties: &runtime.RawExtension{
									Raw: redisPropsJSON,
								},
							},
						},
					},
				},
			},
		}

		r, fakeClient := newControllerTestReconciler(t, appWithRedis)

		ctx := t.Context()
		err := r.reconcileDatabaseInstance(ctx, appWithRedis)
		if err != nil {
			t.Fatalf("reconcileDatabaseInstance should not fail for redis type: %v", err)
		}

		// Verify no StatefulSet was created (only postgres is provisioned)
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
				Replicas: func() *int32 { r := int32(1); return &r }(),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "postgres",
							Image: "postgres:15",
						}},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
					},
				}},
			},
		}

		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{Name: "db", Port: 15432}},
			},
		}

		r, fakeClient := newControllerTestReconciler(t, app, existingSts, existingSvc)

		ctx := t.Context()
		err := r.reconcileDatabaseInstance(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseInstance failed: %v", err)
		}

		updatedSts := &appsv1.StatefulSet{}
		err = fakeClient.Get(ctx, types.NamespacedName{Name: "api-server-db", Namespace: app.Namespace}, updatedSts)
		if err != nil {
			t.Fatalf("failed to get updated StatefulSet: %v", err)
		}
		if got := updatedSts.Spec.Template.Spec.Containers[0].Image; got != "postgres:16" {
			t.Fatalf("expected image postgres:16, got %s", got)
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
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				}},
			},
		}

		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-server-db", Namespace: app.Namespace},
			Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "db", Port: 5432}}},
		}

		r, _ := newControllerTestReconciler(t, app, existingSts, existingSvc)

		err := r.reconcileDatabaseInstance(t.Context(), app)
		if err == nil {
			t.Fatal("expected storage drift error, got nil")
		}
		if !strings.Contains(err.Error(), "storage drift detected") {
			t.Fatalf("expected storage drift error, got %v", err)
		}
	})
}

func TestInjectDatabaseEnvVars(t *testing.T) {
	t.Run("InjectsAllEnvVars", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
								},
							},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret")
		if !changed {
			t.Fatal("Expected InjectDatabaseEnvVars to return true (changed)")
		}

		container := deploy.Spec.Template.Spec.Containers[0]
		// Should have PORT + DB_HOST + DB_USER + DB_PASS + DB_PORT = 5
		if len(container.Env) != 5 {
			t.Fatalf("Expected 5 env vars, got %d", len(container.Env))
		}

		expectedEnvs := map[string]string{
			"DB_HOST": "DB_HOST",
			"DB_USER": "DB_USER",
			"DB_PASS": "DB_PASS",
		}
		foundDBPort := false
		for _, env := range container.Env {
			if expectedKey, ok := expectedEnvs[env.Name]; ok {
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Errorf("Env %s should reference a secret", env.Name)
					continue
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("Env %s: expected secret name %q, got %q",
						env.Name, "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != expectedKey {
					t.Errorf("Env %s: expected secret key %q, got %q",
						env.Name, expectedKey, env.ValueFrom.SecretKeyRef.Key)
				}
				delete(expectedEnvs, env.Name)
			}
			if env.Name == "DB_PORT" {
				if env.Value != "5432" {
					t.Errorf("Expected DB_PORT=5432, got %q", env.Value)
				}
				if env.ValueFrom != nil {
					t.Error("DB_PORT should not use ValueFrom")
				}
				foundDBPort = true
			}
		}
		if len(expectedEnvs) > 0 {
			t.Errorf("Missing expected env vars: %v", expectedEnvs)
		}
		if !foundDBPort {
			t.Error("Missing DB_PORT env var")
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "api-server", Image: "myregistry/api:v1"},
						},
					},
				},
			},
		}

		// First injection
		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret")
		if !changed {
			t.Fatal("Expected first injection to report changes")
		}
		firstCount := len(deploy.Spec.Template.Spec.Containers[0].Env)

		// Second injection — should be idempotent
		changed = InjectDatabaseEnvVars(deploy, "api-server-db-secret")
		if changed {
			t.Error("Expected second injection to report no changes (idempotent)")
		}
		secondCount := len(deploy.Spec.Template.Spec.Containers[0].Env)
		if firstCount != secondCount {
			t.Errorf("Env var count changed after idempotent call: %d → %d", firstCount, secondCount)
		}
	})

	t.Run("UpdatesExistingWrongSource", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
									{Name: "DB_HOST", Value: "hardcoded-host"},
									{Name: "DB_USER", ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "wrong-secret"},
											Key:                  "DB_USER",
										},
									}},
								},
							},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret")
		if !changed {
			t.Fatal("Expected InjectDatabaseEnvVars to return true when existing env vars have wrong source")
		}

		container := deploy.Spec.Template.Spec.Containers[0]
		// Should have PORT + DB_HOST + DB_USER + DB_PASS + DB_PORT = 5
		if len(container.Env) != 5 {
			t.Fatalf("Expected 5 env vars, got %d", len(container.Env))
		}

		// DB_HOST should now reference the secret, not a plain value
		for _, env := range container.Env {
			if env.Name == "DB_HOST" {
				if env.Value != "" {
					t.Error("DB_HOST should have Value cleared")
				}
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Fatal("DB_HOST should reference a secret")
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("DB_HOST: expected secret name %q, got %q", "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
			}
			if env.Name == "DB_USER" {
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Fatal("DB_USER should reference a secret")
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("DB_USER: expected secret name %q, got %q", "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
			}
		}
	})

	t.Run("NoContainers", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "test-secret")
		if changed {
			t.Error("Expected no changes for Deployment with no containers")
		}
	})

	t.Run("TargetsNamedContainerWhenPresent", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "istio-proxy", Image: "proxy:v1"},
							{Name: "api-server", Image: "myregistry/api:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "api-server-db-secret", "api-server", 5433)
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if !exactMatch {
			t.Fatal("Expected exact container match")
		}

		proxyEnvCount := len(deploy.Spec.Template.Spec.Containers[0].Env)
		if proxyEnvCount != 0 {
			t.Fatalf("Expected sidecar env to stay unchanged, got %d", proxyEnvCount)
		}

		appContainer := deploy.Spec.Template.Spec.Containers[1]
		expected := map[string]bool{"DB_HOST": false, "DB_USER": false, "DB_PASS": false, "DB_PORT": false}
		for _, env := range appContainer.Env {
			if _, ok := expected[env.Name]; ok {
				expected[env.Name] = true
				if env.Name == "DB_PORT" && env.Value != "5433" {
					t.Errorf("Expected DB_PORT=5433, got %q", env.Value)
				}
			}
		}
		for name, found := range expected {
			if !found {
				t.Errorf("Expected env %s on target container", name)
			}
		}
	})

	t.Run("FallsBackToFirstContainerWhenPreferredMissing", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "only-container", Image: "myregistry/app:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "app-db-secret", "missing-app", 5432)
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if exactMatch {
			t.Fatal("Expected fallback because preferred container does not exist")
		}
		if len(deploy.Spec.Template.Spec.Containers[0].Env) != 4 {
			t.Fatalf("Expected 4 injected DB env vars, got %d", len(deploy.Spec.Template.Spec.Containers[0].Env))
		}
	})

	t.Run("NoPreferredContainerUsesFallback", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "myregistry/app:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "app-db-secret", "", 5432)
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if exactMatch {
			t.Fatal("Expected fallback semantics when preferred container is empty")
		}
	})
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

func TestReconcileDatabaseSecretInjection(t *testing.T) {
	dbProps := map[string]any{
		"dbType":  "postgres",
		"dbName":  "mydb",
		"version": "16",
	}
	dbPropsJSON, _ := json.Marshal(dbProps)

	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			UID:       "test-uid-inject",
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

	t.Run("InjectsIntoExistingDeployment", func(t *testing.T) {
		existingDeploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-server",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "api-server"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "api-server"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
								},
							},
						},
					},
				},
			},
		}

		r, fakeClient := newControllerTestReconciler(t, app, existingDeploy)

		ctx := t.Context()
		pending, err := r.reconcileDatabaseSecretInjection(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecretInjection failed: %v", err)
		}
		if pending {
			t.Error("Expected no pending injection when Deployment exists")
		}

		// Verify the Deployment was updated with DB env vars
		updatedDeploy := &appsv1.Deployment{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "api-server",
			Namespace: "default",
		}, updatedDeploy)
		if err != nil {
			t.Fatalf("Failed to get updated Deployment: %v", err)
		}

		container := updatedDeploy.Spec.Template.Spec.Containers[0]
		expectedEnvNames := map[string]bool{
			"DB_HOST": false,
			"DB_USER": false,
			"DB_PASS": false,
		}
		for _, env := range container.Env {
			if _, ok := expectedEnvNames[env.Name]; ok {
				expectedEnvNames[env.Name] = true
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Errorf("Env %s should reference a secret", env.Name)
				} else if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("Env %s: expected secret name %q, got %q",
						env.Name, "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-db-app",
				Namespace: "default",
				UID:       "test-uid-no-inject",
			},
			Spec: appv1alpha1.HeliosAppSpec{
				Components: []appv1alpha1.Component{
					{
						Name: "frontend",
						Type: "web-service",
					},
				},
			},
		}

		r, _ := newControllerTestReconciler(t, appWithoutDB)

		ctx := t.Context()
		pending, err := r.reconcileDatabaseSecretInjection(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecretInjection should not fail for app without database traits: %v", err)
		}
		if pending {
			t.Error("Expected no pending injection for app without database traits")
		}
	})

	t.Run("DeploymentNotFound_GracefulSkip", func(t *testing.T) {
		// When Deployment doesn't exist yet (ArgoCD hasn't synced),
		// the reconciler should skip without error.
		r, _ := newControllerTestReconciler(t, app)

		ctx := t.Context()
		pending, err := r.reconcileDatabaseSecretInjection(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecretInjection should not fail when Deployment is missing: %v", err)
		}
		if !pending {
			t.Error("Expected pending=true when Deployment is missing")
		}
	})
}
