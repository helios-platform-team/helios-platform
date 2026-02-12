package cue

import (
	"os"
	"path/filepath"
	"testing"
)

// getCuePath returns the path to the cue directory for tests.
func getCuePath(t *testing.T) string {
	t.Helper()
	cuePath := filepath.Join("..", "..", "..", "..", "cue")
	if _, err := os.Stat(cuePath); os.IsNotExist(err) {
		t.Skipf("CUE path does not exist: %s (run from apps/operator directory)", cuePath)
	}
	return cuePath
}

func TestNewTektonRenderer(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}
	if renderer == nil {
		t.Fatal("Expected non-nil TektonRenderer")
	}
}

// validTektonInput returns a fully-populated TektonInput for testing.
func validTektonInput() TektonInput {
	return TektonInput{
		AppName:         "test-app",
		Namespace:       "default",
		GitRepo:         "https://github.com/myuser/test-app",
		GitBranch:       "main",
		ImageRepo:       "docker.io/myuser/test-app",
		GitOpsRepo:      "https://github.com/myuser/gitops-repo",
		GitOpsPath:      "./apps/test-app",
		GitOpsBranch:    "main",
		GitOpsSecretRef: "github-credentials",
		WebhookDomain:   "hooks.helios.dev",
		WebhookSecret:   "github-secret",
		PipelineName:    "from-code-to-cluster",
		PipelineType:    "from-code-to-cluster",
		TriggerType:     "github-push",
		ServiceAccount:  "tekton-sa",
		PVCName:         "shared-workspace-pvc",
		ContextSubpath:  "",
		Replicas:        1,
		Port:            8080,
		DockerSecret:    "docker-creds",
	}
}

func TestRenderTektonResources_AllResources(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// With webhookDomain set, we expect 8 objects:
	// 3 Tasks + 1 Pipeline + 1 TriggerBinding + 1 TriggerTemplate + 1 EventListener + 1 Ingress
	expectedCount := 8
	if len(objects) != expectedCount {
		t.Errorf("Expected %d objects, got %d", expectedCount, len(objects))
		for i, obj := range objects {
			t.Logf("  [%d] %s: %s", i, obj.GetKind(), obj.GetName())
		}
	}

	// Verify each expected kind is present
	expectedKinds := map[string]int{
		"Task":            3,
		"Pipeline":        1,
		"TriggerBinding":  1,
		"TriggerTemplate": 1,
		"EventListener":   1,
		"Ingress":         1,
	}

	kindCounts := make(map[string]int)
	for _, obj := range objects {
		kindCounts[obj.GetKind()]++
	}

	for kind, expected := range expectedKinds {
		actual := kindCounts[kind]
		if actual != expected {
			t.Errorf("Expected %d %s(s), got %d", expected, kind, actual)
		}
	}
}

func TestRenderTektonResources_WithoutWebhook(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	input.WebhookDomain = "" // No webhook → no Ingress

	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// Without webhookDomain: 7 objects (no Ingress)
	expectedCount := 7
	if len(objects) != expectedCount {
		t.Errorf("Expected %d objects (no webhook), got %d", expectedCount, len(objects))
		for i, obj := range objects {
			t.Logf("  [%d] %s: %s", i, obj.GetKind(), obj.GetName())
		}
	}

	// Verify no Ingress
	for _, obj := range objects {
		if obj.GetKind() == "Ingress" {
			t.Error("Expected no Ingress when webhookDomain is empty")
		}
	}
}

func TestRenderTektonResources_CorrectNamespaces(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	input.Namespace = "my-namespace"

	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	for _, obj := range objects {
		ns := obj.GetNamespace()
		if ns != "my-namespace" {
			t.Errorf("%s %q has namespace %q, expected %q",
				obj.GetKind(), obj.GetName(), ns, "my-namespace")
		}
	}
}

func TestRenderTektonResources_CorrectTaskNames(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	expectedTaskNames := map[string]bool{
		"git-clone":           false,
		"kaniko-build":        false,
		"git-update-manifest": false,
	}

	for _, obj := range objects {
		if obj.GetKind() == "Task" {
			name := obj.GetName()
			if _, ok := expectedTaskNames[name]; ok {
				expectedTaskNames[name] = true
			} else {
				t.Errorf("Unexpected task name: %s", name)
			}
		}
	}

	for name, found := range expectedTaskNames {
		if !found {
			t.Errorf("Expected task %q not found in rendered objects", name)
		}
	}
}

func TestRenderTektonResources_GVKSet(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		if gvk.Kind == "" {
			t.Errorf("Object %s has empty GVK Kind", obj.GetName())
		}
		if gvk.Version == "" {
			t.Errorf("Object %s/%s has empty GVK Version", gvk.Kind, obj.GetName())
		}
	}
}

func TestRenderTektonResources_InvalidInput(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	// Invalid: appName with uppercase (violates CUE regex ^[a-z][a-z0-9-]*$)
	input := TektonInput{
		AppName:   "INVALID-NAME",
		Namespace: "default",
		GitRepo:   "not-a-url",
	}

	_, err = renderer.RenderTektonResources(input)
	if err == nil {
		t.Error("Expected error for invalid input, got nil")
	}
}

func TestRenderTektonResources_BuildOnlyPipeline(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	input.PipelineType = "build-only"

	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// Verify pipeline name is "build-only"
	foundPipeline := false
	for _, obj := range objects {
		if obj.GetKind() == "Pipeline" {
			if obj.GetName() != "build-only" {
				t.Errorf("Expected pipeline name 'build-only', got %q", obj.GetName())
			}
			foundPipeline = true
		}
	}
	if !foundPipeline {
		t.Error("Pipeline not found in rendered objects")
	}
}
