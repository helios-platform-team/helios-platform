package cue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEngine_Render(t *testing.T) {
	// Get the path to the cue directory (relative to test)
	cuePath := filepath.Join("..", "..", "..", "..", "cue")

	// Check if cue path exists
	if _, err := os.Stat(cuePath); os.IsNotExist(err) {
		t.Skipf("CUE path does not exist: %s (run from apps/operator directory)", cuePath)
	}

	// Create engine
	engine, err := NewEngine(cuePath)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create test application
	app := Application{
		App: AppSpec{
			Name:        "test-app",
			Namespace:   "default",
			Owner:       "test-team",
			Description: "Test application",
			Components: []Component{
				{
					Name: "api-server",
					Type: "web-service",
					Properties: map[string]any{
						"image":    "myregistry/api:v1.0.0",
						"port":     3000,
						"replicas": 2,
						"env": []map[string]any{
							{"name": "NODE_ENV", "value": "production"},
						},
					},
					Traits: []Trait{
						{
							Type: "service",
							Properties: map[string]any{
								"port": 3000,
							},
						},
						{
							Type: "ingress",
							Properties: map[string]any{
								"host": "api.example.com",
								"port": 3000,
							},
						},
					},
				},
			},
		},
	}

	// Test RenderToObjects
	t.Run("RenderToObjects", func(t *testing.T) {
		objects, err := engine.RenderToObjects(app)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}

		// Should have 3 objects: Deployment, Service, Ingress
		if len(objects) != 3 {
			t.Errorf("Expected 3 objects, got %d", len(objects))
		}

		// Check Deployment
		foundDeployment := false
		foundService := false
		foundIngress := false

		for _, obj := range objects {
			kind := obj["kind"].(string)
			switch kind {
			case "Deployment":
				foundDeployment = true
				metadata := obj["metadata"].(map[string]any)
				if metadata["name"] != "api-server" {
					t.Errorf("Expected deployment name 'api-server', got '%v'", metadata["name"])
				}
			case "Service":
				foundService = true
				metadata := obj["metadata"].(map[string]any)
				if metadata["name"] != "api-server" {
					t.Errorf("Expected service name 'api-server', got '%v'", metadata["name"])
				}
			case "Ingress":
				foundIngress = true
				metadata := obj["metadata"].(map[string]any)
				if metadata["name"] != "api-server-ingress" {
					t.Errorf("Expected ingress name 'api-server-ingress', got '%v'", metadata["name"])
				}
			}
		}

		if !foundDeployment {
			t.Error("Deployment not found in rendered objects")
		}
		if !foundService {
			t.Error("Service not found in rendered objects")
		}
		if !foundIngress {
			t.Error("Ingress not found in rendered objects")
		}
	})

	// Test Render (YAML output)
	t.Run("Render", func(t *testing.T) {
		yamlBytes, err := engine.Render(app)
		if err != nil {
			t.Fatalf("Failed to render to YAML: %v", err)
		}

		if len(yamlBytes) == 0 {
			t.Error("Expected non-empty YAML output")
		}

		t.Logf("Rendered YAML:\n%s", string(yamlBytes))
	})
}

func TestEngine_NewEngine_InvalidPath(t *testing.T) {
	_, err := NewEngine("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent path, got nil")
	}
}

// TestEngine_RenderWithDatabaseTrait verifies the CUE engine renders the correct
// ConfigMap and Secret when a component includes a "database" trait.
// Acceptance criteria:
//   - app.database.type: "postgres"
//   - app.database.name: "my_custom_db"
func TestEngine_RenderWithDatabaseTrait(t *testing.T) {
	cuePath := filepath.Join("..", "..", "..", "..", "cue")
	if _, err := os.Stat(cuePath); os.IsNotExist(err) {
		t.Skipf("CUE path does not exist: %s", cuePath)
	}

	engine, err := NewEngine(cuePath)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	app := Application{
		App: AppSpec{
			Name:      "test-db-app",
			Namespace: "default",
			Components: []Component{
				{
					Name: "api-server",
					Type: "web-service",
					Properties: map[string]any{
						"image": "myregistry/api:v1.0.0",
						"port":  3000,
					},
					Traits: []Trait{
						{
							Type: "service",
							Properties: map[string]any{
								"port": 3000,
							},
						},
						{
							Type: "database",
							Properties: map[string]any{
								"dbType": "postgres",
								"dbName": "my_custom_db",
							},
						},
					},
				},
			},
		},
	}

	t.Run("RenderDatabaseTrait", func(t *testing.T) {
		objects, err := engine.RenderToObjects(app)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}

		// Expect: Deployment + Service + ConfigMap + Secret = 4
		if len(objects) != 4 {
			t.Fatalf("Expected 4 objects, got %d", len(objects))
		}

		var (
			foundConfigMap bool
			foundSecret    bool
		)

		for _, obj := range objects {
			kind, _ := obj["kind"].(string)
			metadata, _ := obj["metadata"].(map[string]any)
			name, _ := metadata["name"].(string)

			switch {
			case kind == "ConfigMap" && name == "api-server-db-config":
				foundConfigMap = true
				data, ok := obj["data"].(map[string]any)
				if !ok {
					t.Error("ConfigMap data is not a map")
					continue
				}
				// Verify acceptance criteria
				if data["DB_TYPE"] != "postgres" {
					t.Errorf("DB_TYPE: expected %q, got %q", "postgres", data["DB_TYPE"])
				}
				if data["DB_NAME"] != "my_custom_db" {
					t.Errorf("DB_NAME: expected %q, got %q", "my_custom_db", data["DB_NAME"])
				}
				if data["DB_HOST"] != "api-server-db" {
					t.Errorf("DB_HOST: expected %q, got %q", "api-server-db", data["DB_HOST"])
				}
				if data["DB_PORT"] != "5432" {
					t.Errorf("DB_PORT: expected %q, got %q", "5432", data["DB_PORT"])
				}

			case kind == "Secret" && name == "api-server-db-secret":
				foundSecret = true
				// Verify secret has credential keys
				stringData, ok := obj["stringData"].(map[string]any)
				if !ok {
					t.Error("Secret stringData is not a map")
					continue
				}
				if _, hasUser := stringData["DB_USER"]; !hasUser {
					t.Error("Secret missing DB_USER")
				}
				if _, hasPassword := stringData["DB_PASSWORD"]; !hasPassword {
					t.Error("Secret missing DB_PASSWORD")
				}
			}
		}

		if !foundConfigMap {
			t.Error("ConfigMap 'api-server-db-config' not found in rendered objects")
		}
		if !foundSecret {
			t.Error("Secret 'api-server-db-secret' not found in rendered objects")
		}
	})

	t.Run("RenderDatabaseTraitYAML", func(t *testing.T) {
		yamlBytes, err := engine.Render(app)
		if err != nil {
			t.Fatalf("Failed to render YAML: %v", err)
		}
		if len(yamlBytes) == 0 {
			t.Error("Expected non-empty YAML output")
		}
		t.Logf("Database trait YAML:\n%s", string(yamlBytes))
	})
}
