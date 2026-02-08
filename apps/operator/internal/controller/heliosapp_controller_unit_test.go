package controller

import (
	"context"
	"strings"
	"testing"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	heliosCue "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/gitops"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// FakeGitOpsClient is a mock implementation of GitOpsClientInterface for unit tests
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

// FakeCueEngine is a mock implementation of CueEngineInterface
type FakeCueEngine struct{}

func (f *FakeCueEngine) Render(app heliosCue.Application) ([]byte, error) {
	return []byte("rendered: true"), nil
}

func (f *FakeCueEngine) RenderToObjects(app heliosCue.Application) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func TestHeliosAppReconciler_Reconcile_Success(t *testing.T) {
	// 1. Setup Scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	// 2. Setup Mock Objects
	heliosApp := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name:       "frontend",
					Type:       "webservice",
					Properties: &runtime.RawExtension{Raw: []byte(`{"image": "nginx:latest"}`)},
				},
			},
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

	// 3. Setup Fake Client
	// We init with the object existing
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(heliosApp, gitOpsSecret).
		WithStatusSubresource(heliosApp).
		Build()

	// 4. Setup Mock GitOps
	mockGit := &FakeGitOpsClient{}

	// 5. Setup Reconciler
	r := &HeliosAppReconciler{
		Client:    client,
		Scheme:    scheme,
		CueEngine: &FakeCueEngine{},
		GitFactory: func(repo, user, token string) gitops.GitOpsClientInterface {
			return mockGit
		},
	}

	// 6. Run Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	ctx := context.Background()
	res, err := r.Reconcile(ctx, req)

	// 7. Assertions
	if err != nil {
		t.Errorf("Reconcile() error = %v, wantErr %v", err, nil)
	}
	if res.Requeue {
		t.Errorf("Reconcile() Requeue = %v, want %v", res.Requeue, false)
	}

	// Verify GitOps was called
	// SyncManifest(ctx, targetPath, content)
	expectedPath := "apps/test-app/manifest.yaml"
	found := false
	for path := range mockGit.SyncedFiles {
		if strings.Contains(path, expectedPath) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("GitOps SyncManifest was not called for path %s. Synced: %v", expectedPath, mockGit.SyncedFiles)
	}

	// Verify Status Update (Optional, requires fetching object again)
	updatedApp := &appv1alpha1.HeliosApp{}
	_ = client.Get(ctx, req.NamespacedName, updatedApp)
	if updatedApp.Status.Phase != appv1alpha1.PhaseReady {
		t.Errorf("Expected Phase to be %s, got %s", appv1alpha1.PhaseReady, updatedApp.Status.Phase)
	}
}

// Add a test for missing image (Pending phase)
func TestHeliosAppReconciler_Reconcile_PendingImage(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appv1alpha1.AddToScheme(scheme))

	heliosApp := &appv1alpha1.HeliosApp{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-app", Namespace: "default"},
		Spec: appv1alpha1.HeliosAppSpec{
			Components: []appv1alpha1.Component{
				{
					Name:       "backend",
					Type:       "worker",
					Properties: &runtime.RawExtension{Raw: []byte(`{"cmd": "run"}`)}, // Missing image
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(heliosApp).WithStatusSubresource(heliosApp).Build()

	r := &HeliosAppReconciler{
		Client:    client,
		Scheme:    scheme,
		CueEngine: &FakeCueEngine{},
	}

	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "pending-app", Namespace: "default"}})

	if err != nil {
		t.Errorf("Reconcile() error = %v, wantErr %v", err, nil)
	}
	if (res != ctrl.Result{}) {
		t.Errorf("Reconcile() result = %v, want empty", res)
	}

	updatedApp := &appv1alpha1.HeliosApp{}
	_ = client.Get(context.Background(), types.NamespacedName{Name: "pending-app", Namespace: "default"}, updatedApp)
	if updatedApp.Status.Phase != appv1alpha1.PhasePending {
		t.Errorf("Expected Phase to be %s, got %s", appv1alpha1.PhasePending, updatedApp.Status.Phase)
	}
}
