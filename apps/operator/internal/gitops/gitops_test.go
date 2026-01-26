package gitops

import (
	"context"
	"testing"
)

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
