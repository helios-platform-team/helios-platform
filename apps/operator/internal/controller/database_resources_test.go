package controller

import (
	"context"
	"encoding/json"
	"testing"

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
	dbProps := map[string]interface{}{
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

	dbProps := map[string]interface{}{
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

		ctx := context.Background()
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

		ctx := context.Background()
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

		ctx := context.Background()
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
