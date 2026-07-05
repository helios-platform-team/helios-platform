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

// validTektonInput returns a fully populated TektonInput for testing.
func validTektonInput() TektonInput {
	return TektonInput{
		AppName:          "test-app",
		Namespace:        "default",
		GitRepo:          "http://gitea-http.gitea.svc.cluster.local:3000/myuser/test-app.git",
		GitBranch:        "main",
		ImageRepo:        "docker.io/myuser/test-app",
		GitOpsRepo:       "http://gitea-http.gitea.svc.cluster.local:3000/myuser/gitops-repo.git",
		GitOpsPath:       "./apps/test-app",
		GitOpsBranch:     "main",
		GitOpsSecretRef:  "helios-gitops-bot",
		WebhookDomain:    "hooks.helios.dev",
		WebhookSecret:    "gitea-webhook-secret",
		PipelineName:     "from-code-to-cluster",
		PipelineType:     "from-code-to-cluster",
		TriggerType:      "gitea-push",
		ServiceAccount:   "tekton-sa",
		PVCName:          "shared-workspace-pvc",
		ContextSubpath:   "",
		Replicas:         1,
		Port:             8080,
		DockerSecret:     "docker-creds",
		ArgoCDNamespace:  "argocd",
		StorageDriver:    "overlay",
		BuildahIsolation: "chroot",
		BuildPlatforms:   "linux/amd64,linux/arm64",
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

	// With webhookDomain set, we expect 13 objects:
	// 6 Tasks (git-clone, buildah-build, git-update-manifest, argocd-sync, db-migrate, postgrest-reload)
	// + 2 Pipelines (from-code-to-cluster + db-migrate)
	// + 1 TriggerBinding + 2 TriggerTemplates (gitea + db-migrate) + 1 EventListener + 1 Ingress
	expectedCount := 13
	if len(objects) != expectedCount {
		t.Errorf("Expected %d objects, got %d", expectedCount, len(objects))
		for i, obj := range objects {
			t.Logf("  [%d] %s: %s", i, obj.GetKind(), obj.GetName())
		}
	}

	// Verify each expected kind is present
	expectedKinds := map[string]int{
		"Task":            6,
		"Pipeline":        2,
		"TriggerBinding":  1,
		"TriggerTemplate": 2,
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

	// Without webhookDomain: 12 objects (no Ingress)
	// 6 Tasks + 2 Pipelines + 1 TriggerBinding + 2 TriggerTemplates + 1 EventListener
	expectedCount := 12
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
		"buildah-build":       false,
		"git-update-manifest": false,
		"argocd-sync":         false,
		"db-migrate":          false,
		"postgrest-reload":    false,
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

	// Verify the primary pipeline with the name "build-only" exists among rendered pipelines.
	// Note: db-migrate pipeline is always rendered alongside the primary pipeline.
	foundBuildOnly := false
	for _, obj := range objects {
		if obj.GetKind() == "Pipeline" && obj.GetName() == "build-only" {
			foundBuildOnly = true
		}
	}
	if !foundBuildOnly {
		t.Error("Expected primary pipeline 'build-only' not found in rendered objects")
	}
}

func TestRenderTektonResources_ValidationConstraints(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	// 1. Test imageRepo with space
	inputSpace := validTektonInput()
	inputSpace.ImageRepo = "docker.io/my user/test-app"
	_, err = renderer.RenderTektonResources(inputSpace)
	if err == nil {
		t.Error("Expected error for imageRepo containing spaces, got nil")
	}

	// 2. Test contextSubpath with parent directory escape
	inputEscape := validTektonInput()
	inputEscape.ContextSubpath = "../escape"
	_, err = renderer.RenderTektonResources(inputEscape)
	if err == nil {
		t.Error("Expected error for contextSubpath containing traversal, got nil")
	}
}
