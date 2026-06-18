package provider

import (
	"os"
	"strings"
)

// GitHubProvider implements GitProvider for GitHub.com and GitHub Enterprise Server.
type GitHubProvider struct{}

// NewGitHubProvider creates a GitHubProvider.
func NewGitHubProvider() *GitHubProvider { return &GitHubProvider{} }

func (p *GitHubProvider) Name() Type { return TypeGithub }

func (p *GitHubProvider) RewriteURL(repoURL string) string {
	externalURL := strings.TrimSuffix(os.Getenv("GITHUB_URL"), "/")
	internalURL := strings.TrimSuffix(os.Getenv("GITHUB_INTERNAL_URL"), "/")

	if externalURL == "" || internalURL == "" {
		return repoURL
	}
	return strings.Replace(repoURL, externalURL, internalURL, 1)
}

func (p *GitHubProvider) DefaultTriggerType() string { return "git-push" }

func (p *GitHubProvider) WebhookSecretName() string { return "webhook-secret" }

func (p *GitHubProvider) DefaultUsername() string {
	if u := os.Getenv("GITHUB_USER"); u != "" {
		return u
	}
	return "git"
}

func (p *GitHubProvider) TokenEnvVar() string { return "GITHUB_TOKEN" }

func (p *GitHubProvider) UserEnvVar() string { return "GITHUB_USER" }

func (p *GitHubProvider) DefaultCredentialSecretName() string { return "github-credentials" }
