package controller

import (
	"os"
	"testing"
)

func TestRewriteGiteaURL(t *testing.T) {
	t.Setenv("GITEA_INTERNAL_URL", "http://gitea-http.gitea.svc.cluster.local:3000")

	t.Run("rewrites exact external match", func(t *testing.T) {
		t.Setenv("GITEA_URL", "http://localhost:3030")
		in := "http://localhost:3030/helios-platform/test-2.git"
		want := "http://gitea-http.gitea.svc.cluster.local:3000/helios-platform/test-2.git"
		if got := rewriteGiteaURL(in); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("rewrites loopback variant (127.0.0.1)", func(t *testing.T) {
		t.Setenv("GITEA_URL", "http://localhost:3030")
		in := "http://127.0.0.1:3030/helios-platform/test-2.git"
		want := "http://gitea-http.gitea.svc.cluster.local:3000/helios-platform/test-2.git"
		if got := rewriteGiteaURL(in); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back when env missing", func(t *testing.T) {
		_ = os.Unsetenv("GITEA_URL")
		_ = os.Unsetenv("GITEA_INTERNAL_URL")
		in := "http://127.0.0.1:3030/helios-platform/test-2.git"
		if got := rewriteGiteaURL(in); got != in {
			t.Fatalf("got %q, want %q", got, in)
		}
	})
}
