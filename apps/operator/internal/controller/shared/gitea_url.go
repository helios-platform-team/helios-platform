package shared

import (
	"os"
	"strings"
)

// RewriteGiteaURL translates an external Gitea URL (e.g. http://localhost:3030/...)
// to the in-cluster service URL so in-cluster components can reach it.
//
// Falls back to the original URL if GITEA_URL / GITEA_INTERNAL_URL are not set.
func RewriteGiteaURL(repoURL string) string {
	externalURL := os.Getenv("GITEA_URL")
	internalURL := os.Getenv("GITEA_INTERNAL_URL")
	if externalURL != "" && internalURL != "" {
		rewritten := strings.Replace(repoURL, externalURL, internalURL, 1)
		if rewritten != repoURL {
			return rewritten
		}

		const (
			localhost = "localhost"
			loopback  = "127.0.0.1"
		)
		if strings.Contains(externalURL, localhost) {
			altExternal := strings.Replace(externalURL, localhost, loopback, 1)
			rewritten = strings.Replace(repoURL, altExternal, internalURL, 1)
			if rewritten != repoURL {
				return rewritten
			}
		}
		if strings.Contains(externalURL, loopback) {
			altExternal := strings.Replace(externalURL, loopback, localhost, 1)
			rewritten = strings.Replace(repoURL, altExternal, internalURL, 1)
			if rewritten != repoURL {
				return rewritten
			}
		}
	}
	return repoURL
}
