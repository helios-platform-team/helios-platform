package controller

import (
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

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

	for i := range iterations {
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
	// Create a fake client
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appv1alpha1.AddToScheme(scheme)

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
		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(app).
			Build()

		r := &HeliosAppReconciler{
			Client: client,
			Scheme: scheme,
		}

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets failed: %v", err)
		}

		// Verify secret was created
		secret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{
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
				"DB_HOST": []byte("existing-host"),
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(app, existingSecret).
			Build()

		r := &HeliosAppReconciler{
			Client: client,
			Scheme: scheme,
		}

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, app)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets failed: %v", err)
		}

		// Verify existing secret was not modified
		secret := &corev1.Secret{}
		err = client.Get(ctx, types.NamespacedName{
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

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(appWithoutDB).
			Build()

		r := &HeliosAppReconciler{
			Client: client,
			Scheme: scheme,
		}

		ctx := t.Context()
		err := r.reconcileDatabaseSecrets(ctx, appWithoutDB)
		if err != nil {
			t.Fatalf("reconcileDatabaseSecrets should not fail for app without database traits: %v", err)
		}

		// Verify no secret was created
		secretList := &corev1.SecretList{}
		err = client.List(ctx, secretList)
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
			expectedPGDATA := PostgresDataPath + "/" + PostgresDataSubPath
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
		if !strings.Contains(cmdStr, "-p $(PGPORT)") {
			t.Errorf("LivenessProbe command missing custom port flag. Got: %s", cmdStr)
		}
	}

	// Verify readinessProbe uses custom port
	if container.ReadinessProbe == nil {
		t.Error("ReadinessProbe should be set on Postgres container")
	} else {
		cmdStr := strings.Join(container.ReadinessProbe.Exec.Command, " ")
		if !strings.Contains(cmdStr, "-p $(PGPORT)") {
			t.Errorf("ReadinessProbe command missing custom port flag. Got: %s", cmdStr)
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
		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(app).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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

		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(appWithoutDB).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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

		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(appWithRedis).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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
		// Should have PORT + DB_HOST + DB_USER + DB_PASS = 4
		if len(container.Env) != 4 {
			t.Fatalf("Expected 4 env vars, got %d", len(container.Env))
		}

		expectedEnvs := map[string]string{
			"DB_HOST": "DB_HOST",
			"DB_USER": "DB_USER",
			"DB_PASS": "DB_PASS",
		}
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
		}
		if len(expectedEnvs) > 0 {
			t.Errorf("Missing expected env vars: %v", expectedEnvs)
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
		// Should have PORT + DB_HOST + DB_USER + DB_PASS = 4
		if len(container.Env) != 4 {
			t.Fatalf("Expected 4 env vars, got %d", len(container.Env))
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
		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

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

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(app, existingDeploy).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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

		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(appWithoutDB).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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
		fullScheme := runtime.NewScheme()
		_ = corev1.AddToScheme(fullScheme)
		_ = appv1alpha1.AddToScheme(fullScheme)
		_ = appsv1.AddToScheme(fullScheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(fullScheme).
			WithObjects(app).
			Build()

		r := &HeliosAppReconciler{
			Client: fakeClient,
			Scheme: fullScheme,
		}

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
