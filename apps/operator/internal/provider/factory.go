package provider

import (
	"os"
)

// Default is the provider instance for the currently configured git provider.
// It is set at init time based on the GIT_PROVIDER_TYPE environment variable.
var Default GitProvider

func init() {
	Default = NewProviderFromEnv(os.Getenv("GIT_PROVIDER_TYPE"))
}

// NewProviderFromEnv creates a GitProvider based on the given provider type string.
// Falls back to GiteaProvider when the type is empty or unrecognized.
func NewProviderFromEnv(providerType string) GitProvider {
	switch Type(providerType) {
	case TypeGithub:
		return NewGitHubProvider()
	case TypeGitLab:
		return NewGiteaProvider() // fallback until GitLab is implemented
	default:
		return NewGiteaProvider()
	}
}
