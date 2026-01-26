package gitops

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitOpsClientInterface defines the methods for GitOps operations
type GitOpsClientInterface interface {
	SyncManifest(ctx context.Context, filePath, content string) error
}

// GitOpsClient handles interactions with the GitOps repository
type GitOpsClient struct {
	RepoURL     string
	Auth        *http.BasicAuth
	AuthorName  string
	AuthorEmail string
	// Allow checking out to memory for testing
	inMemory bool
}

// NewGitOpsClient creates a new client
// token should be a Personal Access Token (PAT)
func NewGitOpsClient(repoURL, username, token string) *GitOpsClient {
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
		inMemory:    false,
	}
}

// NewInMemoryGitOpsClient creates a new client that uses in-memory storage (for testing)
func NewInMemoryGitOpsClient(repoURL, username, token string) *GitOpsClient {
	return &GitOpsClient{
		RepoURL: repoURL,
		Auth: &http.BasicAuth{
			Username: username,
			Password: token,
		},
		AuthorName:  "Helios Operator",
		AuthorEmail: "operator@helios.io",
		inMemory:    true,
	}
}

// SyncManifest clones the repo, updates the manifest file, commits, and pushes the changes.
// filePath: relative path to the manifest file in the repo (e.g. "apps/my-app/manifest.yaml")
func (c *GitOpsClient) SyncManifest(ctx context.Context, filePath, content string) error {
	var r *git.Repository
	var err error
	var w *git.Worktree

	if c.inMemory {
		// For testing: Init a new repo in memory directly with a memory filesystem
		fs := memfs.New()
		r, err = git.Init(memory.NewStorage(), fs)
		if err != nil {
			return fmt.Errorf("failed to init in-memory repo: %w", err)
		}
		// Create a dummy commit so we have a HEAD
		w, _ = r.Worktree()
		_, _ = w.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  c.AuthorName,
				Email: c.AuthorEmail,
				When:  time.Now(),
			},
		})
	} else {
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

		r, err = git.PlainCloneContext(ctx, tempDir, false, cloneOpts)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
		w, err = r.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
	}

	// 3. Write Manifest File
	// Note: For in-memory, we can't use os.WriteFile naturally unless we use billy.Filesystem
	// But go-git Worktree uses billy.Filesystem abstractly.
	// So we should use w.Filesystem.

	fs := w.Filesystem
	if err := fs.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	// OPTIMIZATION: Check if content has changed before writing
	// This prevents unnecessary IO and git timestamp updates
	existingFile, err := fs.Open(filePath)
	if err == nil {
		// File exists, check content
		existingContent, readErr := io.ReadAll(existingFile)
		existingFile.Close()

		if readErr == nil && string(existingContent) == content {
			fmt.Println("Manifest content unchanged. Skipping commit.")
			return nil
		}
	}

	f, err := fs.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		return fmt.Errorf("failed to write manifest content: %w", err)
	}
	f.Close()

	// 4. Git Add
	if _, err := w.Add(filePath); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	// Check if there are changes to commit
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get git status: %w", err)
	}

	if status.IsClean() {
		fmt.Println("No changes to commit (git status clean).")
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
	if !c.inMemory {
		fmt.Println("Pushing changes...")
		pushOpts := &git.PushOptions{
			Auth: c.Auth,
		}
		if err := r.PushContext(ctx, pushOpts); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
		fmt.Println("Successfully pushed to GitOps repository.")
	} else {
		fmt.Println("In-memory mode: Skipping push.")
	}

	return nil
}
