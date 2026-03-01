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
