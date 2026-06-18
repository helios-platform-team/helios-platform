package provider

// Type represents the type of git provider.
type Type string

const (
	// TypeGitea represents a Gitea instance.
	TypeGitea Type = "gitea"
	// TypeGithub represents GitHub.com or GitHub Enterprise Server.
	TypeGithub Type = "github"
	// TypeGitLab represents GitLab.com or self-hosted GitLab.
	TypeGitLab Type = "gitlab"
)

// GitProvider defines an abstraction over different git hosting providers.
// Implementations handle provider-specific URL rewriting, defaults, and configuration.
type GitProvider interface {
	// Name returns the provider type identifier (e.g., "gitea", "github").
	Name() Type

	// RewriteURL translates an external repository URL to an in-cluster URL
	// that internal components can reach. Returns the original URL if no
	// rewriting is configured.
	RewriteURL(externalURL string) string

	// DefaultTriggerType returns the default trigger type for Tekton webhooks.
	DefaultTriggerType() string

	// WebhookSecretName returns the default Kubernetes Secret name for webhook secrets.
	WebhookSecretName() string

	// DefaultUsername returns the default username for git authentication
	// when none is explicitly provided.
	DefaultUsername() string
}
