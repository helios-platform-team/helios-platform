package provider

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCredentialResolver_AppSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-git-secret", Namespace: "default"},
		Data: map[string][]byte{
			"token":    []byte("secret-token"),
			"username": []byte("bot-user"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, username, err := resolver.ResolveGitCredentials(t.Context(), "default", "my-git-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "secret-token" {
		t.Errorf("token = %q, want %q", token, "secret-token")
	}
	if username != "bot-user" {
		t.Errorf("username = %q, want %q", username, "bot-user")
	}
}

func TestCredentialResolver_AppSecretWithPassword(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "my-git-secret", Namespace: "default"},
		Data: map[string][]byte{
			"password": []byte("pass-token"),
			"username": []byte("bot-user"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, _, err := resolver.ResolveGitCredentials(t.Context(), "default", "my-git-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "pass-token" {
		t.Errorf("token = %q, want %q", token, "pass-token")
	}
}

func TestCredentialResolver_MissingAppSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	_, _, err := resolver.ResolveGitCredentials(t.Context(), "default", "missing-secret")
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestCredentialResolver_ProviderDefaultSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	defaultSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "helios-gitops-bot", Namespace: "default"},
		Data: map[string][]byte{
			"token":    []byte("default-token"),
			"username": []byte("default-user"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(defaultSecret).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, username, err := resolver.ResolveGitCredentials(t.Context(), "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "default-token" {
		t.Errorf("token = %q, want %q", token, "default-token")
	}
	if username != "default-user" {
		t.Errorf("username = %q, want %q", username, "default-user")
	}
}

func TestCredentialResolver_EnvVarFallback(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	t.Setenv("GITEA_TOKEN", "env-token")
	t.Setenv("GITEA_USER", "env-user")

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, username, err := resolver.ResolveGitCredentials(t.Context(), "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("token = %q, want %q", token, "env-token")
	}
	if username != "env-user" {
		t.Errorf("username = %q, want %q", username, "env-user")
	}
}

func TestCredentialResolver_GenericEnvVarFallback(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	t.Setenv("GIT_TOKEN", "generic-token")
	t.Setenv("GIT_USER", "generic-user")

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, username, err := resolver.ResolveGitCredentials(t.Context(), "default", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "generic-token" {
		t.Errorf("token = %q, want %q", token, "generic-token")
	}
	if username != "generic-user" {
		t.Errorf("username = %q, want %q", username, "generic-user")
	}
}

func TestCredentialResolver_NoCredentials(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	_, _, err := resolver.ResolveGitCredentials(t.Context(), "default", "")
	if err == nil {
		t.Fatal("expected error when no credentials are configured")
	}
}

func TestCredentialResolver_AppSecretTakesPriority(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	appSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "default"},
		Data: map[string][]byte{
			"token":    []byte("app-token"),
			"username": []byte("app-user"),
		},
	}
	defaultSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "helios-gitops-bot", Namespace: "default"},
		Data: map[string][]byte{
			"token": []byte("default-token"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(appSecret, defaultSecret).Build()
	resolver := NewCredentialResolver(fakeClient, NewGiteaProvider())

	token, _, err := resolver.ResolveGitCredentials(t.Context(), "default", "app-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "app-token" {
		t.Errorf("token = %q, want %q (app secret should take priority)", token, "app-token")
	}
}

func TestNewCredentialResolver(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	p := NewGiteaProvider()
	resolver := NewCredentialResolver(fakeClient, p)
	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}
}
