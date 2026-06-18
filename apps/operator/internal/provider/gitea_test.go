package provider

import (
	"testing"
)

func TestGiteaProvider_RewriteURL(t *testing.T) {
	p := &GiteaProvider{}
	t.Setenv("GITEA_INTERNAL_URL", "http://gitea-http.gitea.svc.cluster.local:3000")

	t.Run("rewrites exact external match", func(t *testing.T) {
		t.Setenv("GITEA_URL", "http://localhost:3030")
		in := "http://localhost:3030/helios-platform/test-2.git"
		want := "http://gitea-http.gitea.svc.cluster.local:3000/helios-platform/test-2.git"
		if got := p.RewriteURL(in); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("rewrites loopback variant (127.0.0.1)", func(t *testing.T) {
		t.Setenv("GITEA_URL", "http://localhost:3030")
		in := "http://127.0.0.1:3030/helios-platform/test-2.git"
		want := "http://gitea-http.gitea.svc.cluster.local:3000/helios-platform/test-2.git"
		if got := p.RewriteURL(in); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back when GITEA_URL missing", func(t *testing.T) {
		t.Setenv("GITEA_URL", "")
		in := "http://127.0.0.1:3030/helios-platform/test-2.git"
		if got := p.RewriteURL(in); got != in {
			t.Fatalf("got %q, want %q", got, in)
		}
	})

	t.Run("falls back when GITEA_INTERNAL_URL missing", func(t *testing.T) {
		t.Setenv("GITEA_URL", "http://localhost:3030")
		t.Setenv("GITEA_INTERNAL_URL", "")
		in := "http://localhost:3030/helios-platform/test-2.git"
		if got := p.RewriteURL(in); got != in {
			t.Fatalf("got %q, want %q", got, in)
		}
	})
}

func TestGiteaProvider_Defaults(t *testing.T) {
	p := NewGiteaProvider()

	if got := p.Name(); got != TypeGitea {
		t.Errorf("Name() = %q, want %q", got, TypeGitea)
	}
	if got := p.DefaultTriggerType(); got != "gitea-push" {
		t.Errorf("DefaultTriggerType() = %q, want %q", got, "gitea-push")
	}
	if got := p.WebhookSecretName(); got != "gitea-webhook-secret" {
		t.Errorf("WebhookSecretName() = %q, want %q", got, "gitea-webhook-secret")
	}
	if got := p.DefaultUsername(); got != "git" {
		t.Errorf("DefaultUsername() = %q, want %q", got, "git")
	}
	if got := p.TokenEnvVar(); got != "GITEA_TOKEN" {
		t.Errorf("TokenEnvVar() = %q, want %q", got, "GITEA_TOKEN")
	}
	if got := p.UserEnvVar(); got != "GITEA_USER" {
		t.Errorf("UserEnvVar() = %q, want %q", got, "GITEA_USER")
	}
}

func TestGitHubProvider_RewriteURL(t *testing.T) {
	p := &GitHubProvider{}
	t.Setenv("GITHUB_URL", "https://github.mycompany.com")
	t.Setenv("GITHUB_INTERNAL_URL", "http://github-enterprise.default.svc.cluster.local")
	in := "https://github.mycompany.com/myorg/myrepo.git"
	want := "http://github-enterprise.default.svc.cluster.local/myorg/myrepo.git"
	if got := p.RewriteURL(in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGitHubProvider_Defaults(t *testing.T) {
	p := NewGitHubProvider()

	if got := p.Name(); got != TypeGithub {
		t.Errorf("Name() = %q, want %q", got, TypeGithub)
	}
	if got := p.DefaultTriggerType(); got != "git-push" {
		t.Errorf("DefaultTriggerType() = %q, want %q", got, "git-push")
	}
	if got := p.WebhookSecretName(); got != "webhook-secret" {
		t.Errorf("WebhookSecretName() = %q, want %q", got, "webhook-secret")
	}
	if got := p.DefaultUsername(); got != "git" {
		t.Errorf("DefaultUsername() = %q, want %q", got, "git")
	}
	if got := p.TokenEnvVar(); got != "GITHUB_TOKEN" {
		t.Errorf("TokenEnvVar() = %q, want %q", got, "GITHUB_TOKEN")
	}
	if got := p.UserEnvVar(); got != "GITHUB_USER" {
		t.Errorf("UserEnvVar() = %q, want %q", got, "GITHUB_USER")
	}
}

func TestNewProviderFromEnv(t *testing.T) {
	t.Run("empty type returns Gitea", func(t *testing.T) {
		p := NewProviderFromEnv("")
		if p.Name() != TypeGitea {
			t.Errorf("got %q, want %q", p.Name(), TypeGitea)
		}
	})

	t.Run("gitea type returns GiteaProvider", func(t *testing.T) {
		p := NewProviderFromEnv("gitea")
		if p.Name() != TypeGitea {
			t.Errorf("got %q, want %q", p.Name(), TypeGitea)
		}
	})

	t.Run("github type returns GitHubProvider", func(t *testing.T) {
		p := NewProviderFromEnv("github")
		if p.Name() != TypeGithub {
			t.Errorf("got %q, want %q", p.Name(), TypeGithub)
		}
	})

	t.Run("unknown type falls back to Gitea", func(t *testing.T) {
		p := NewProviderFromEnv("unknown")
		if p.Name() != TypeGitea {
			t.Errorf("got %q, want %q", p.Name(), TypeGitea)
		}
	})
}
