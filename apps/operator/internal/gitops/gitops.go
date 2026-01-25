package gitops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitOpsClient handles interactions with the GitOps repository
type GitOpsClient struct {
	RepoURL     string
	Auth        *http.BasicAuth
	AuthorName  string
	AuthorEmail string
}

// NewGitOpsClient creates a new client
// token should be a Personal Access Token (PAT)
func NewGitOpsClient(repoURL, username, token string) *GitOpsClient {
	return &GitOpsClient{
		RepoURL: repoURL,
		Auth: &http.BasicAuth{
			Username: username,
			Password: token,
		},
		AuthorName:  "Helios Operator",
		AuthorEmail: "operator@helios.io",
	}
}

// SyncManifest clones the repo, updates the manifest file, commits, and pushes the changes.
// filePath: relative path to the manifest file in the repo (e.g. "apps/my-app/manifest.yaml")
func (c *GitOpsClient) SyncManifest(ctx context.Context, filePath, content string) error {
	// 1. Create temporary directory
	tempDir, err := os.MkdirTemp("", "helios-gitops-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Ensure cleanup

	// 2. Clone Repository
	fmt.Printf("Cloning %s to %s\n", c.RepoURL, tempDir)
	cloneOpts := &git.CloneOptions{
		URL:      c.RepoURL,
		Progress: os.Stdout,
		Auth:     c.Auth,
		Depth:    1, // Shallow clone for speed
	}

	r, err := git.PlainCloneContext(ctx, tempDir, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// 3. Write Manifest File
	fullPath := filepath.Join(tempDir, filePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// Overwrite mode
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	// 4. Git Add
	// Note: We add the specific file. If it's new, it gets added. If modified, it gets staged.
	if _, err := w.Add(filePath); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	// Check if there are changes to commit
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get git status: %w", err)
	}

	// Check only the specific file status or general status
	if status.IsClean() {
		fmt.Println("No changes to commit.")
		return nil
	}

	// 5. Git Commit
	msg := fmt.Sprintf("Update manifest: %s", filePath)
	commitHash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  c.AuthorName,
			Email: c.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	fmt.Printf("Committed changes: %s\n", commitHash)

	// 6. Git Push
	fmt.Println("Pushing changes...")
	pushOpts := &git.PushOptions{
		Auth: c.Auth,
	}
	if err := r.PushContext(ctx, pushOpts); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	fmt.Println("Successfully pushed to GitOps repository.")
	return nil
}
