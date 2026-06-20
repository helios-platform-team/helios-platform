package provider

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CredentialsResolver resolves git authentication credentials.
type CredentialsResolver interface {
	// ResolveGitCredentials resolves the token and username for git operations.
	//
	// Resolution order:
	//  1. Explicit secret referenced by secretRef (highest priority)
	//  2. Provider default secret (e.g., "helios-gitops-bot" for Gitea)
	//  3. Environment variables (provider-specific with generic fallback)
	ResolveGitCredentials(ctx context.Context, namespace, secretRef string) (token, username string, err error)
}

// credentialResolver is the default implementation of CredentialsResolver.
type credentialResolver struct {
	client   client.Client
	provider GitProvider
}

// NewCredentialResolver creates a CredentialsResolver backed by the given
// Kubernetes client and git provider.
func NewCredentialResolver(c client.Client, p GitProvider) CredentialsResolver {
	return &credentialResolver{client: c, provider: p}
}

// ResolveGitCredentials resolves the token and username for git operations.
//
// Resolution order:
//  1. Explicit secret referenced by secretRef (highest priority)
//  2. Provider default secret (e.g., "helios-gitops-bot" for Gitea)
//  3. Environment variables (provider-specific with generic fallback)
func (r *credentialResolver) ResolveGitCredentials(
	ctx context.Context,
	namespace, secretRef string,
) (token, username string, err error) {
	// 1. App-specific secret reference
	if secretRef != "" {
		t, u, found := r.readSecret(ctx, namespace, secretRef)
		if !found {
			return "", "", fmt.Errorf("secret %q not found or missing 'token'/'password' key in namespace %q", secretRef, namespace)
		}
		return t, u, nil
	}

	// 2. Provider default secret
	defaultSecret := r.provider.DefaultCredentialSecretName()
	if t, u, found := r.readSecret(ctx, namespace, defaultSecret); found {
		return t, u, nil
	}

	// 3. Environment variable fallback
	token = readTokenEnv(r.provider)
	username = readUserEnv(r.provider)
	if username == "" {
		username = r.provider.DefaultUsername()
	}

	if token == "" {
		return "", "", fmt.Errorf(
			"GitOps token is empty. Set %s or %s env var, create a Kubernetes Secret %q, or set GitOpsSecretRef",
			r.provider.TokenEnvVar(), "GIT_TOKEN", defaultSecret,
		)
	}

	return token, username, nil
}

// readSecret looks up a Kubernetes Secret and extracts the token and username.
// Returns zero values and false if the secret is missing or lacks credentials.
func (r *credentialResolver) readSecret(ctx context.Context, namespace, name string) (token, username string, found bool) {
	var secret corev1.Secret
	if err := r.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &secret); err != nil {
		return "", "", false
	}

	if t, ok := secret.Data["token"]; ok {
		token = string(t)
	} else if p, ok := secret.Data["password"]; ok {
		token = string(p)
	} else {
		return "", "", false
	}

	if u, ok := secret.Data["username"]; ok {
		username = string(u)
	}

	return token, username, true
}

// readTokenEnv reads the auth token from the provider-specific env var,
// falling back to the generic GIT_TOKEN.
func readTokenEnv(p GitProvider) string {
	if t := os.Getenv(p.TokenEnvVar()); t != "" {
		return t
	}
	return os.Getenv("GIT_TOKEN")
}

// readUserEnv reads the username from the provider-specific env var,
// falling back to the generic GIT_USER.
func readUserEnv(p GitProvider) string {
	if u := os.Getenv(p.UserEnvVar()); u != "" {
		return u
	}
	return os.Getenv("GIT_USER")
}
