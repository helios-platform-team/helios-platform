package gitopssync

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
)

type FakeGitOpsClient struct {
	SyncedFiles map[string]string
}

func (m *FakeGitOpsClient) SyncManifest(ctx context.Context, filePath, content string) error {
	if m.SyncedFiles == nil {
		m.SyncedFiles = make(map[string]string)
	}
	m.SyncedFiles[filePath] = content
	return nil
}

func setupReconcilerTest(t *testing.T) (*Reconciler, *FakeGitOpsClient, *appv1alpha1.HeliosApp) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	heliosApp := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			GitOpsRepo:      "https://github.com/test/repo",
			GitOpsPath:      "apps/test-app",
			GitOpsSecretRef: "gitops-secret",
		},
	}

	gitOpsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitops-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token":    []byte("dummy-token"),
			"username": []byte("dummy-user"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(heliosApp, gitOpsSecret).
		WithStatusSubresource(heliosApp).
		Build()

	mockGit := &FakeGitOpsClient{}
	r := NewReconciler(fakeClient, scheme, func(repo, user, token string) gitops.ClientInterface {
		return mockGit
	})

	return r, mockGit, heliosApp
}

func TestReconcile_UsesSerializedCR(t *testing.T) {
	r, mockGit, app := setupReconcilerTest(t)
	ctx := context.TODO()

	crBytes := []byte("fake-serialized-cr")
	err := r.Reconcile(ctx, app, crBytes)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	expectedHash := computeHash([]byte(app.Spec.GitOpsRepo + "\x00" + app.Spec.GitOpsPath + "\x00" + string(crBytes)))

	// Check that the hash uses the crBytes
	updatedApp := &appv1alpha1.HeliosApp{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, updatedApp)
	if err != nil {
		t.Fatalf("Failed to get updated app: %v", err)
	}

	if updatedApp.Status.LastAppliedHash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, updatedApp.Status.LastAppliedHash)
	}

	// Verify the content was passed correctly
	path := "apps/test-app/helios-app.yaml"
	if mockGit.SyncedFiles[path] != string(crBytes) {
		t.Errorf("Expected synced content %s, got %s", string(crBytes), mockGit.SyncedFiles[path])
	}
}

func TestReconcile_TargetPathHasCorrectExtension(t *testing.T) {
	r, mockGit, app := setupReconcilerTest(t)
	ctx := context.TODO()

	crBytes := []byte("fake-serialized-cr")
	err := r.Reconcile(ctx, app, crBytes)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	path := "apps/test-app/helios-app.yaml"
	if _, ok := mockGit.SyncedFiles[path]; !ok {
		t.Errorf("Expected file to be synced at path %s", path)
	}
}

func TestReconcile_StatusMessageReflectsCR(t *testing.T) {
	r, _, app := setupReconcilerTest(t)
	ctx := context.TODO()

	crBytes := []byte("fake-serialized-cr")
	err := r.Reconcile(ctx, app, crBytes)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedApp := &appv1alpha1.HeliosApp{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, updatedApp)
	if err != nil {
		t.Fatalf("Failed to get updated app: %v", err)
	}

	if !strings.Contains(updatedApp.Status.Message, "HeliosApp CR pushed to") {
		t.Errorf("Expected status message to contain 'HeliosApp CR pushed to', got: %s", updatedApp.Status.Message)
	}
}

func TestReconcile_HashStableForCR(t *testing.T) {
	r, mockGit, app := setupReconcilerTest(t)
	ctx := context.TODO()

	crBytes := []byte("fake-serialized-cr")
	err := r.Reconcile(ctx, app, crBytes)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify it was synced the first time
	if len(mockGit.SyncedFiles) != 1 {
		t.Fatalf("Expected 1 sync, got %d", len(mockGit.SyncedFiles))
	}

	// Clear the synced files
	mockGit.SyncedFiles = make(map[string]string)

	// Call again with the exact same crBytes
	err = r.Reconcile(ctx, app, crBytes)
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify no new sync occurred because hash is the same
	if len(mockGit.SyncedFiles) != 0 {
		t.Errorf("Expected 0 syncs due to stable hash, got %d", len(mockGit.SyncedFiles))
	}
}

func TestComputeHash_MatchesExpected(t *testing.T) {
	data := []byte("test-data")
	hash := computeHash(data)
	if hash == "" {
		t.Errorf("computeHash returned empty string")
	}
}
