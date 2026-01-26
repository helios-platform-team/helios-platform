package gitops

import (
	"context"
	"os"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// NewInMemoryGitOpsClient creates a new client that uses in-memory storage (for testing)
func NewInMemoryGitOpsClient(repoURL, username, token string) *GitOpsClient {
	authorName := os.Getenv("GIT_AUTHOR_NAME")
	if authorName == "" {
		authorName = "Helios Operator"
	}
	authorEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	if authorEmail == "" {
		authorEmail = "operator@helios.io"
	}

	return &GitOpsClient{
		RepoURL: repoURL,
		Auth: &http.BasicAuth{
			Username: username,
			Password: token,
		},
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
		inMemory:    true,
	}
}

func TestGitOpsClient_SyncManifest(t *testing.T) {
	// Use the InMemory client to avoid network calls and filesystem issues
	client := NewInMemoryGitOpsClient("https://github.com/example/repo", "user", "token")

	ctx := context.TODO()
	filePath := "apps/test-app/manifest.yaml"
	content := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test-pod"

	err := client.SyncManifest(ctx, filePath, content)
	if err != nil {
		t.Errorf("SyncManifest() error = %v, wantErr %v", err, nil)
	}
}
