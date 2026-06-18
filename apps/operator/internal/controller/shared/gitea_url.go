package shared

import (
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/provider"
)

// RewriteGiteaURL translates an external git URL to the in-cluster service URL
// so in-cluster components can reach it. Delegates to the configured git provider.
//
// Deprecated: Use provider.Default.RewriteURL() directly for provider-agnostic code.
func RewriteGiteaURL(repoURL string) string {
	return provider.Default.RewriteURL(repoURL)
}
