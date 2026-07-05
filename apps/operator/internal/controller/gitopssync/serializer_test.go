/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gitopssync_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/gitopssync"
)

// TestSerializeHeliosApp_HappyPath tests basic serialization with a valid, populated CR
func TestSerializeHeliosApp_HappyPath(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	if len(output) == 0 {
		t.Fatal("SerializeHeliosApp returned empty YAML")
	}

	// Verify YAML is valid by unmarshaling
	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("serialized YAML is invalid: %v\nOutput:\n%s", err, string(output))
	}

	// Verify basic fields are preserved
	if unmarshaled.Name != "test-app" {
		t.Errorf("Name not preserved: got %q, want %q", unmarshaled.Name, "test-app")
	}
	if unmarshaled.Namespace != "default" {
		t.Errorf("Namespace not preserved: got %q, want %q", unmarshaled.Namespace, "default")
	}
	if unmarshaled.Spec.GitRepo != "https://git.example.com/repo" {
		t.Errorf("Spec.GitRepo not preserved")
	}
}

// TestSerializeHeliosApp_NilInput tests error handling for nil input
func TestSerializeHeliosApp_NilInput(t *testing.T) {
	output, err := gitopssync.SerializeHeliosApp(nil)

	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	if output != nil {
		t.Errorf("expected nil output for nil input, got %s", string(output))
	}
	if err.Error() != "serialize failed: HeliosApp instance is nil" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestSerializeHeliosApp_StripsStatus verifies Status subresource is removed
func TestSerializeHeliosApp_StripsStatus(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
		Status: appv1alpha1.HeliosAppStatus{
			Phase: "Ready",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Status should be empty (zero value)
	if unmarshaled.Status.Phase != "" {
		t.Errorf("Status.Phase not stripped: got %q, want %q", unmarshaled.Status.Phase, "")
	}

	// Verify all status fields are cleared
	if len(unmarshaled.Status.Conditions) > 0 {
		t.Errorf("Status.Conditions not cleared: got %d conditions, want 0", len(unmarshaled.Status.Conditions))
	}
	if len(unmarshaled.Status.ResourcesCreated) > 0 {
		t.Errorf("Status.ResourcesCreated not cleared: got %d resources, want 0", len(unmarshaled.Status.ResourcesCreated))
	}
	if unmarshaled.Status.LastAppliedHash != "" {
		t.Errorf("Status.LastAppliedHash not cleared: got %q, want empty", unmarshaled.Status.LastAppliedHash)
	}

	t.Logf("Serialized YAML:\n%s", string(output))
}

// TestSerializeHeliosApp_StripsResourceVersion verifies resourceVersion is removed
func TestSerializeHeliosApp_StripsResourceVersion(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-app",
			Namespace:       "default",
			ResourceVersion: "12345",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.ResourceVersion != "" {
		t.Errorf("ResourceVersion not stripped: got %q, want empty", unmarshaled.ResourceVersion)
	}
}

// TestSerializeHeliosApp_StripsUID verifies uid is removed
func TestSerializeHeliosApp_StripsUID(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			UID:       "abc-123-def-456",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.UID != "" {
		t.Errorf("UID not stripped: got %q, want empty", unmarshaled.UID)
	}
}

// TestSerializeHeliosApp_StripsGeneration verifies generation is removed
func TestSerializeHeliosApp_StripsGeneration(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.Generation != 0 {
		t.Errorf("Generation not stripped: got %d, want 0", unmarshaled.Generation)
	}
}

// TestSerializeHeliosApp_StripsManagedFields verifies managedFields is removed
func TestSerializeHeliosApp_StripsManagedFields(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			ManagedFields: []metav1.ManagedFieldsEntry{
				{
					Manager: "kubectl",
					FieldsV1: &metav1.FieldsV1{
						Raw: []byte(`{"f:metadata":{"f:name":{}}}`),
					},
				},
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(unmarshaled.ManagedFields) != 0 {
		t.Errorf("ManagedFields not stripped: got %d fields, want 0", len(unmarshaled.ManagedFields))
	}
}

// TestSerializeHeliosApp_StripsCreationTimestamp verifies creationTimestamp is removed
func TestSerializeHeliosApp_StripsCreationTimestamp(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if !unmarshaled.CreationTimestamp.IsZero() {
		t.Errorf("CreationTimestamp not stripped: got %v, want zero", unmarshaled.CreationTimestamp)
	}
}

// TestSerializeHeliosApp_StripsFinalizers verifies finalizers are removed
func TestSerializeHeliosApp_StripsFinalizers(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "default",
			Finalizers: []string{"heliosapp.operator.com/finalizer", "custom/finalizer"},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(unmarshaled.Finalizers) != 0 {
		t.Errorf("Finalizers not stripped: got %v, want empty", unmarshaled.Finalizers)
	}
}

// TestSerializeHeliosApp_StripsOwnerReferences verifies ownerReferences are removed
func TestSerializeHeliosApp_StripsOwnerReferences(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Namespace",
					Name:       "parent",
					UID:        "some-uid",
				},
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(unmarshaled.OwnerReferences) != 0 {
		t.Errorf("OwnerReferences not stripped: got %v, want empty", unmarshaled.OwnerReferences)
	}
}

// TestSerializeHeliosApp_PreservesName verifies Name is preserved
func TestSerializeHeliosApp_PreservesName(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-special-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.Name != "my-special-app" {
		t.Errorf("Name not preserved: got %q, want %q", unmarshaled.Name, "my-special-app")
	}
}

// TestSerializeHeliosApp_PreservesNamespace verifies Namespace is preserved
func TestSerializeHeliosApp_PreservesNamespace(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "custom-namespace",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.Namespace != "custom-namespace" {
		t.Errorf("Namespace not preserved: got %q, want %q", unmarshaled.Namespace, "custom-namespace")
	}
}

// TestSerializeHeliosApp_PreservesLabels verifies Labels are preserved
func TestSerializeHeliosApp_PreservesLabels(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Labels: map[string]string{
				"app":  "myapp",
				"team": "platform",
				"env":  "prod",
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.Labels["app"] != "myapp" {
		t.Errorf("Label 'app' not preserved: got %q, want %q", unmarshaled.Labels["app"], "myapp")
	}
	if unmarshaled.Labels["team"] != "platform" {
		t.Errorf("Label 'team' not preserved: got %q, want %q", unmarshaled.Labels["team"], "platform")
	}
	if unmarshaled.Labels["env"] != "prod" {
		t.Errorf("Label 'env' not preserved: got %q, want %q", unmarshaled.Labels["env"], "prod")
	}
}

// TestSerializeHeliosApp_PreservesAnnotations verifies Annotations are preserved (except kubectl-injected ones)
func TestSerializeHeliosApp_PreservesAnnotations(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Annotations: map[string]string{
				"custom/annotation":  "value1",
				"another/annotation": "value2",
				"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"v1","kind":"Pod"}`, // Should be removed
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// User annotations should be preserved
	if unmarshaled.Annotations["custom/annotation"] != "value1" {
		t.Errorf("custom annotation not preserved: got %q, want %q", unmarshaled.Annotations["custom/annotation"], "value1")
	}
	if unmarshaled.Annotations["another/annotation"] != "value2" {
		t.Errorf("another annotation not preserved: got %q, want %q", unmarshaled.Annotations["another/annotation"], "value2")
	}

	// kubectl-injected annotation should be removed
	if _, hasKubectl := unmarshaled.Annotations["kubectl.kubernetes.io/last-applied-configuration"]; hasKubectl {
		t.Error("kubectl.kubernetes.io/last-applied-configuration should be removed but is still present")
	}
}

// TestSerializeHeliosApp_DoesNotMutateOriginal verifies deep-copy prevents mutations
func TestSerializeHeliosApp_DoesNotMutateOriginal(t *testing.T) {
	original := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			UID:       "original-uid",
			Annotations: map[string]string{
				"custom": "value",
				"kubectl.kubernetes.io/last-applied-configuration": "should-be-removed",
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
		Status: appv1alpha1.HeliosAppStatus{
			Phase: "Ready",
		},
	}

	// Keep a copy of original state before serialization
	originalUID := original.UID
	originalStatus := original.Status.Phase

	_, err := gitopssync.SerializeHeliosApp(original)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	// Verify original object is unchanged
	if original.UID != originalUID {
		t.Errorf("Original object UID was mutated: got %q, want %q", original.UID, originalUID)
	}
	if original.Status.Phase != originalStatus {
		t.Errorf("Original object Status.Phase was mutated: got %q, want %q", original.Status.Phase, originalStatus)
	}

	// kubectl annotation should still be in original
	if _, hasKubectl := original.Annotations["kubectl.kubernetes.io/last-applied-configuration"]; !hasKubectl {
		t.Error("Original object was mutated: kubectl annotation was removed")
	}
}

// TestSerializeHeliosApp_ComplexSpec tests complex Spec preservation
func TestSerializeHeliosApp_ComplexSpec(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-app",
			Namespace: "production",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			Owner:           "platform-team",
			GitRepo:         "https://git.example.com/myapp",
			GitBranch:       "main",
			GitOpsRepo:      "https://git.example.com/gitops",
			GitOpsPath:      "apps/myapp",
			GitOpsBranch:    "main",
			GitOpsSecretRef: "gitea-credentials",
			ImageRepo:       "registry.example.com/myapp",
			PipelineName:    "from-code-to-cluster",
			TriggerType:     "gitea-push",
			WebhookSecret:   "gitea-webhook-secret",
			Description:     "My test application",
			ArgoCDNamespace: "argocd",
			ArgoCDProject:   "default",
			Replicas:        3,
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Verify all Spec fields are preserved
	if unmarshaled.Spec.Owner != "platform-team" {
		t.Errorf("Owner not preserved")
	}
	if unmarshaled.Spec.GitRepo != "https://git.example.com/myapp" {
		t.Errorf("GitRepo not preserved")
	}
	if unmarshaled.Spec.GitOpsRepo != "https://git.example.com/gitops" {
		t.Errorf("GitOpsRepo not preserved")
	}
	if unmarshaled.Spec.ImageRepo != "registry.example.com/myapp" {
		t.Errorf("ImageRepo not preserved")
	}
	if unmarshaled.Spec.Replicas != 3 {
		t.Errorf("Replicas not preserved: got %d, want 3", unmarshaled.Spec.Replicas)
	}
}

// TestSerializeHeliosApp_TypeMetaSet verifies TypeMeta is properly set in output
func TestSerializeHeliosApp_TypeMetaSet(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if unmarshaled.APIVersion != "app.helios.io/v1alpha1" {
		t.Errorf("APIVersion not set correctly: got %q, want %q", unmarshaled.APIVersion, "app.helios.io/v1alpha1")
	}
	if unmarshaled.Kind != "HeliosApp" {
		t.Errorf("Kind not set correctly: got %q, want %q", unmarshaled.Kind, "HeliosApp")
	}
}

// TestSerializeHeliosApp_EmptyAnnotations verifies empty annotations map is handled correctly
func TestSerializeHeliosApp_EmptyAnnotations(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "default",
			Annotations: map[string]string{}, // Empty map
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Empty annotations should be cleaned to nil
	if len(unmarshaled.Annotations) > 0 {
		t.Errorf("empty annotations not cleaned to nil: got %v", unmarshaled.Annotations)
	}
}

// TestSerializeHeliosApp_OnlyKubectlAnnotation verifies kafka annotation removal results in empty map cleanup
func TestSerializeHeliosApp_OnlyKubectlAnnotation(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": `{"foo":"bar"}`,
			},
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitRepo:    "https://git.example.com/repo",
			GitOpsRepo: "https://git.example.com/gitops",
			GitOpsPath: "apps/test",
			ImageRepo:  "registry.example.com/test",
		},
	}

	output, err := gitopssync.SerializeHeliosApp(app)
	if err != nil {
		t.Fatalf("SerializeHeliosApp failed: %v", err)
	}

	var unmarshaled appv1alpha1.HeliosApp
	if err := yaml.Unmarshal(output, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// All annotations removed, map should be nil or empty
	if len(unmarshaled.Annotations) > 0 {
		t.Errorf("annotations not fully cleaned: got %v", unmarshaled.Annotations)
	}
}
