package provider

import (
	"os"
	"strings"
)

// GiteaProvider implements GitProvider for Gitea instances.
type GiteaProvider struct{}

// NewGiteaProvider creates a GiteaProvider.
func NewGiteaProvider() *GiteaProvider { return &GiteaProvider{} }

func (p *GiteaProvider) Name() Type { return TypeGitea }

func (p *GiteaProvider) RewriteURL(repoURL string) string {
	externalURL := strings.TrimSuffix(os.Getenv("GITEA_URL"), "/")
	internalURL := strings.TrimSuffix(os.Getenv("GITEA_INTERNAL_URL"), "/")

	if externalURL == "" || internalURL == "" {
		return repoURL
	}

	rewritten := strings.Replace(repoURL, externalURL, internalURL, 1)
	if rewritten != repoURL {
		return rewritten
	}

	if strings.Contains(externalURL, "localhost") {
		altExternal := strings.Replace(externalURL, "localhost", "127.0.0.1", 1)
		rewritten = strings.Replace(repoURL, altExternal, internalURL, 1)
		if rewritten != repoURL {
			return rewritten
		}
	}
	if strings.Contains(externalURL, "127.0.0.1") {
		altExternal := strings.Replace(externalURL, "127.0.0.1", "localhost", 1)
		rewritten = strings.Replace(repoURL, altExternal, internalURL, 1)
		if rewritten != repoURL {
			return rewritten
		}
	}

	return repoURL
}

func (p *GiteaProvider) DefaultTriggerType() string { return "gitea-push" }

func (p *GiteaProvider) WebhookSecretName() string { return "gitea-webhook-secret" }

func (p *GiteaProvider) DefaultUsername() string {
	if u := os.Getenv("GITEA_USER"); u != "" {
		return u
	}
	return "git"
}

func (p *GiteaProvider) TokenEnvVar() string { return "GITEA_TOKEN" }

func (p *GiteaProvider) UserEnvVar() string { return "GITEA_USER" }

func (p *GiteaProvider) DefaultCredentialSecretName() string { return "helios-gitops-bot" }
