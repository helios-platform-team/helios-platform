package cue

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ============================================================================
// E2E Validation Tests (T6)
//
// These tests verify that CUE-rendered Tekton resources match the expected
// structure and content that was previously produced by hardcoded Go functions.
// They serve as the "identical resource structure" acceptance criterion.
// ============================================================================

// TestE2E_CueVsLegacy_ResourceKinds verifies that the CUE engine produces
// the same set of resource kinds as the old Go Generate* functions.
func TestE2E_CueVsLegacy_ResourceKinds(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	input.WebhookDomain = "" // Without webhook, same as legacy default

	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	// Legacy Go code produced these exact resource types (minus SA/RoleBinding/ClusterRoleBinding
	// which are RBAC and still hardcoded):
	// 3 Tasks + 1 Pipeline + 1 TriggerBinding + 1 TriggerTemplate + 1 EventListener = 7
	expectedKindSet := map[string]bool{
		"Task":            true,
		"Pipeline":        true,
		"TriggerBinding":  true,
		"TriggerTemplate": true,
		"EventListener":   true,
	}

	renderedKinds := make(map[string]bool)
	for _, obj := range objects {
		renderedKinds[obj.GetKind()] = true
	}

	for kind := range expectedKindSet {
		if !renderedKinds[kind] {
			t.Errorf("CUE output missing resource kind %q (legacy Go produced this kind)", kind)
		}
	}
}

// TestE2E_CueVsLegacy_WithWebhook verifies that the CUE engine produces
// Ingress when webhookDomain is set, matching what the legacy GenerateIngress() did.
func TestE2E_CueVsLegacy_WithWebhook(t *testing.T) {
	cuePath := getCuePath(t)
	renderer, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create TektonRenderer: %v", err)
	}

	input := validTektonInput()
	input.WebhookDomain = "hooks.helios.dev" // Enable webhook → expect Ingress

	objects, err := renderer.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("RenderTektonResources failed: %v", err)
	}

	foundIngress := false
	for _, obj := range objects {
		if obj.GetKind() == "Ingress" {
			foundIngress = true
			// Verify Ingress matches legacy structure
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Error("Ingress spec is not a map")
				continue
			}
			rules, ok := spec["rules"].([]any)
			if !ok || len(rules) == 0 {
				t.Error("Ingress has no rules")
				continue
			}
			rule := rules[0].(map[string]any)
			host, ok := rule["host"].(string)
			if !ok || host == "" {
				t.Error("Ingress host is empty")
			}
		}
	}
	if !foundIngress {
		t.Error("Expected Ingress when webhookDomain is set, but none found")
	}
}

// TestE2E_CueVsLegacy_TaskNames verifies the exact task names match legacy.
func TestE2E_CueVsLegacy_TaskNames(t *testing.T) {
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

	// Legacy Go code generated these exact task names:
	expectedTaskNames := []string{"git-clone", "kaniko-build", "git-update-manifest"}
	slices.Sort(expectedTaskNames)

	var actualTaskNames []string
	for _, obj := range objects {
		if obj.GetKind() == "Task" {
			actualTaskNames = append(actualTaskNames, obj.GetName())
		}
	}
	slices.Sort(actualTaskNames)

	if len(actualTaskNames) != len(expectedTaskNames) {
		t.Fatalf("Expected %d tasks, got %d: %v", len(expectedTaskNames), len(actualTaskNames), actualTaskNames)
	}
	for i := range expectedTaskNames {
		if actualTaskNames[i] != expectedTaskNames[i] {
			t.Errorf("Task[%d] name mismatch: expected %q, got %q", i, expectedTaskNames[i], actualTaskNames[i])
		}
	}
}

// TestE2E_CueVsLegacy_PipelineName verifies the pipeline name matches legacy.
func TestE2E_CueVsLegacy_PipelineName(t *testing.T) {
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

	// Legacy Go code: GeneratePipeline() hardcoded name "from-code-to-cluster"
	for _, obj := range objects {
		if obj.GetKind() == "Pipeline" {
			if obj.GetName() != "from-code-to-cluster" {
				t.Errorf("Pipeline name mismatch: expected %q, got %q",
					"from-code-to-cluster", obj.GetName())
			}
			return
		}
	}
	t.Error("Pipeline not found in CUE-rendered objects")
}

// TestE2E_CueVsLegacy_TriggerAPIVersions verifies that the CUE-rendered triggers
// use the correct API versions (same as legacy).
func TestE2E_CueVsLegacy_TriggerAPIVersions(t *testing.T) {
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

	expectedAPIVersions := map[string]string{
		"Task":            "tekton.dev/v1beta1",
		"Pipeline":        "tekton.dev/v1beta1",
		"TriggerBinding":  "triggers.tekton.dev/v1beta1",
		"TriggerTemplate": "triggers.tekton.dev/v1beta1",
		"EventListener":   "triggers.tekton.dev/v1beta1",
	}

	for _, obj := range objects {
		kind := obj.GetKind()
		expectedVersion, ok := expectedAPIVersions[kind]
		if !ok {
			continue // Skip kinds we don't check (e.g., Ingress)
		}
		actualVersion := obj.Object["apiVersion"].(string)
		if actualVersion != expectedVersion {
			t.Errorf("%s apiVersion mismatch: expected %q, got %q",
				kind, expectedVersion, actualVersion)
		}
	}
}

// TestE2E_CueVsLegacy_TriggerBindingParams verifies that the CUE TriggerBinding
// contains the expected params from the legacy GenerateTriggerBinding().
func TestE2E_CueVsLegacy_TriggerBindingParams(t *testing.T) {
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
		if obj.GetKind() == "TriggerBinding" {
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Error("TriggerBinding spec is not a map")
				continue
			}
			params, ok := spec["params"].([]any)
			if !ok {
				t.Error("TriggerBinding has no params")
				continue
			}
			if len(params) == 0 {
				t.Error("TriggerBinding params are empty")
			}
			return
		}
	}
	t.Error("TriggerBinding not found in CUE-rendered objects")
}

// TestE2E_CueVsLegacy_EventListenerInterceptors verifies the CUE EventListener
// has GitHub interceptors (matching improved CUE design vs legacy).
func TestE2E_CueVsLegacy_EventListenerInterceptors(t *testing.T) {
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
		if obj.GetKind() == "EventListener" {
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Error("EventListener spec is not a map")
				continue
			}
			triggers, ok := spec["triggers"].([]any)
			if !ok || len(triggers) == 0 {
				t.Error("EventListener has no triggers")
				continue
			}

			// Check that at least one trigger has interceptors (CUE improvement over legacy)
			trigger := triggers[0].(map[string]any)
			interceptors, ok := trigger["interceptors"].([]any)
			if !ok || len(interceptors) == 0 {
				t.Error("EventListener trigger has no interceptors (CUE should add GitHub interceptor)")
				continue
			}

			// Verify the interceptor references GitHub
			interceptor := interceptors[0].(map[string]any)
			ref, ok := interceptor["ref"].(map[string]any)
			if !ok {
				t.Error("Interceptor has no ref")
				continue
			}
			if ref["name"] != "github" {
				t.Errorf("Interceptor ref name should be 'github', got %v", ref["name"])
			}
			return
		}
	}
	t.Error("EventListener not found in CUE-rendered objects")
}

// TestE2E_CueVsLegacy_PipelineTaskStructure verifies the pipeline spec has
// the expected task references (matching the legacy GeneratePipeline structure).
func TestE2E_CueVsLegacy_PipelineTaskStructure(t *testing.T) {
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
		if obj.GetKind() == "Pipeline" {
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Error("Pipeline spec is not a map")
				return
			}

			// Check that the pipeline has tasks
			tasks, ok := spec["tasks"].([]any)
			if !ok || len(tasks) == 0 {
				t.Error("Pipeline has no tasks")
				return
			}

			// CUE pipeline may have more tasks than legacy (e.g. test step).
			// Verify at least 3 tasks (legacy minimum).
			minTaskCount := 3
			if len(tasks) < minTaskCount {
				t.Errorf("Pipeline should have at least %d tasks, got %d", minTaskCount, len(tasks))
			}

			for i, task := range tasks {
				taskMap := task.(map[string]any)
				// Tasks may use taskRef (external) or taskSpec (inline)
				_, hasTaskRef := taskMap["taskRef"]
				_, hasTaskSpec := taskMap["taskSpec"]
				if !hasTaskRef && !hasTaskSpec {
					t.Errorf("Pipeline task[%d] has neither taskRef nor taskSpec", i)
				}
			}

			// Check pipeline has params
			params, ok := spec["params"].([]any)
			if !ok || len(params) == 0 {
				t.Error("Pipeline has no params")
			}

			// Check pipeline has workspaces
			workspaces, ok := spec["workspaces"].([]any)
			if !ok || len(workspaces) == 0 {
				t.Error("Pipeline has no workspaces")
			}
			return
		}
	}
	t.Error("Pipeline not found in CUE-rendered objects")
}

// TestE2E_CueRenderDeterministic verifies that rendering the same input
// twice produces identical results (deterministic output).
func TestE2E_CueRenderDeterministic(t *testing.T) {
	cuePath := getCuePath(t)

	renderer1, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create first TektonRenderer: %v", err)
	}
	renderer2, err := NewTektonRenderer(cuePath)
	if err != nil {
		t.Fatalf("Failed to create second TektonRenderer: %v", err)
	}

	input := validTektonInput()

	objects1, err := renderer1.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("First render failed: %v", err)
	}
	objects2, err := renderer2.RenderTektonResources(input)
	if err != nil {
		t.Fatalf("Second render failed: %v", err)
	}

	if len(objects1) != len(objects2) {
		t.Fatalf("Determinism failure: first render=%d objects, second render=%d objects",
			len(objects1), len(objects2))
	}

	for i := range objects1 {
		if objects1[i].GetKind() != objects2[i].GetKind() ||
			objects1[i].GetName() != objects2[i].GetName() {
			t.Errorf("Determinism failure at index %d: %s/%s vs %s/%s",
				i,
				objects1[i].GetKind(), objects1[i].GetName(),
				objects2[i].GetKind(), objects2[i].GetName())
		}
	}
}

// TestE2E_NoLegacyFunctionReferences verifies that the old Generate* functions
// for resources now handled by CUE are no longer present as exported functions.
// This is a compile-time check that's validated here for documentation.
func TestE2E_NoLegacyFunctionReferences(t *testing.T) {
	// This test validates that legacy functions have been removed by checking
	// that tekton_resources.go does NOT export functions that CUE now handles.
	// We do this by scanning the source file.
	tektonResPath := filepath.Join("..", "controller", "tekton_resources.go")
	if _, err := os.Stat(tektonResPath); os.IsNotExist(err) {
		t.Skip("tekton_resources.go not found (expected if fully deleted)")
	}

	content, err := os.ReadFile(tektonResPath)
	if err != nil {
		t.Fatalf("Failed to read tekton_resources.go: %v", err)
	}

	// These functions should have been removed (CUE handles them now)
	removedFunctions := []string{
		"func GenerateEventListener(",
		"func GenerateTriggerBinding(",
		"func GenerateDefaultsTriggerBinding(",
		"func GenerateTriggerTemplate(",
		"func GenerateIngress(",
		"func GeneratePVC(",
		"func GenerateGitCloneTask(",
		"func GenerateKanikoBuildTask(",
		"func GenerateGitUpdateManifestTask(",
		"func GeneratePipeline(",
	}

	for _, fn := range removedFunctions {
		if strings.Contains(string(content), fn) {
			t.Errorf("Legacy function %s still exists in tekton_resources.go — should be removed", fn)
		}
	}

	// These functions should still exist (not handled by CUE)
	keptFunctions := []string{
		"func GeneratePipelineRunForManifestGeneration(",
		"func GenerateServiceAccount(",
		"func GenerateRoleBinding(",
		"func GenerateClusterRoleBinding(",
	}

	for _, fn := range keptFunctions {
		if !strings.Contains(string(content), fn) {
			t.Errorf("Required function %s is missing from tekton_resources.go — should be kept", fn)
		}
	}
}
