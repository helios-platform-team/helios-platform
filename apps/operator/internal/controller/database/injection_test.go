package database

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestConnectionURLTemplateForDBType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dbType string
		ok     bool
	}{
		{"postgres", true},
		{"POSTGRES", true},
		{"postgresql", true},
		{" PostgreSQL ", true},
		{"mysql", false},
		{"", false},
	}
	for _, tt := range tests {
		name := tt.dbType
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, ok := connectionURLTemplateForDBType(tt.dbType)
			if ok != tt.ok {
				t.Fatalf("ok: got %v, want %v (template %q)", ok, tt.ok, got)
			}
			if tt.ok && got != postgresDatabaseURLTemplate {
				t.Fatalf("template: got %q, want %q", got, postgresDatabaseURLTemplate)
			}
		})
	}
}

func TestInjectDatabaseEnvVars(t *testing.T) {
	t.Run("InjectsAllEnvVars", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
								},
							},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret", "api-server-db", "postgres")
		if !changed {
			t.Fatal("Expected InjectDatabaseEnvVars to return true (changed)")
		}

		container := deploy.Spec.Template.Spec.Containers[0]
		if len(container.Env) != 8 {
			t.Fatalf("Expected 8 env vars, got %d", len(container.Env))
		}

		expectedEnvs := map[string]string{
			"DB_HOST": "DB_HOST",
			"DB_USER": "DB_USER",
			"DB_PASS": "DB_PASS",
			"PGRST_DB_URI": "PGRST_DB_URI",
		}
		foundDBPort := false
		for _, env := range container.Env {
			if expectedKey, ok := expectedEnvs[env.Name]; ok {
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Errorf("Env %s should reference a secret", env.Name)
					continue
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("Env %s: expected secret name %q, got %q",
						env.Name, "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
				if env.ValueFrom.SecretKeyRef.Key != expectedKey {
					t.Errorf("Env %s: expected secret key %q, got %q",
						env.Name, expectedKey, env.ValueFrom.SecretKeyRef.Key)
				}
				delete(expectedEnvs, env.Name)
			}
			if env.Name == "DB_PORT" {
				if env.Value != "5432" {
					t.Errorf("Expected DB_PORT=5432, got %q", env.Value)
				}
				if env.ValueFrom != nil {
					t.Error("DB_PORT should not use ValueFrom")
				}
				foundDBPort = true
			}
			if env.Name == "DB_NAME" {
				if env.Value != "api-server-db" || env.ValueFrom != nil {
					t.Errorf("Expected DB_NAME literal api-server-db, got %+v", env)
				}
			}
			if env.Name == "DATABASE_URL" {
				if env.Value != postgresDatabaseURLTemplate || env.ValueFrom != nil {
					t.Errorf("Expected DATABASE_URL from template, got %+v", env)
				}
			}
		}
		if len(expectedEnvs) > 0 {
			t.Errorf("Missing expected env vars: %v", expectedEnvs)
		}
		if !foundDBPort {
			t.Error("Missing DB_PORT env var")
		}
	})

	t.Run("OmitsDatabaseURLWhenDBTypeUnsupported", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
								},
							},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret", "api-server-db", "mysql")
		if !changed {
			t.Fatal("Expected InjectDatabaseEnvVars to return true (changed)")
		}

		container := deploy.Spec.Template.Spec.Containers[0]
		if len(container.Env) != 7 {
			t.Fatalf("Expected 7 env vars (no DATABASE_URL), got %d", len(container.Env))
		}
		for _, env := range container.Env {
			if env.Name == "DATABASE_URL" {
				t.Fatalf("unexpected DATABASE_URL for dbType mysql: %+v", env)
			}
			if env.Name == "DB_NAME" && env.Value != "api-server-db" {
				t.Errorf("DB_NAME: expected api-server-db, got %q", env.Value)
			}
		}
	})

	t.Run("Idempotent", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "api-server", Image: "myregistry/api:v1"},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret", "api-server-db", "postgres")
		if !changed {
			t.Fatal("Expected first injection to report changes")
		}
		firstCount := len(deploy.Spec.Template.Spec.Containers[0].Env)

		changed = InjectDatabaseEnvVars(deploy, "api-server-db-secret", "api-server-db", "postgres")
		if changed {
			t.Error("Expected second injection to report no changes (idempotent)")
		}
		secondCount := len(deploy.Spec.Template.Spec.Containers[0].Env)
		if firstCount != secondCount {
			t.Errorf("Env var count changed after idempotent call: %d -> %d", firstCount, secondCount)
		}
	})

	t.Run("UpdatesExistingWrongSource", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "api-server",
								Image: "myregistry/api:v1",
								Env: []corev1.EnvVar{
									{Name: "PORT", Value: "3000"},
									{Name: "DB_HOST", Value: "hardcoded-host"},
									{Name: "DB_USER", ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "wrong-secret"},
											Key:                  "DB_USER",
										},
									}},
								},
							},
						},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "api-server-db-secret", "api-server-db", "postgres")
		if !changed {
			t.Fatal("Expected InjectDatabaseEnvVars to return true when existing env vars have wrong source")
		}

		container := deploy.Spec.Template.Spec.Containers[0]
		if len(container.Env) != 8 {
			t.Fatalf("Expected 8 env vars, got %d", len(container.Env))
		}

		for _, env := range container.Env {
			if env.Name == "DB_HOST" {
				if env.Value != "" {
					t.Error("DB_HOST should have Value cleared")
				}
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Fatal("DB_HOST should reference a secret")
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("DB_HOST: expected secret name %q, got %q", "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
			}
			if env.Name == "DB_USER" {
				if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
					t.Fatal("DB_USER should reference a secret")
				}
				if env.ValueFrom.SecretKeyRef.Name != "api-server-db-secret" {
					t.Errorf("DB_USER: expected secret name %q, got %q", "api-server-db-secret", env.ValueFrom.SecretKeyRef.Name)
				}
			}
		}
	})

	t.Run("NoContainers", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{},
					},
				},
			},
		}

		changed := InjectDatabaseEnvVars(deploy, "test-secret", "", "")
		if changed {
			t.Error("Expected no changes for Deployment with no containers")
		}
	})

	t.Run("TargetsNamedContainerWhenPresent", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "istio-proxy", Image: "proxy:v1"},
							{Name: "api-server", Image: "myregistry/api:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "api-server-db-secret", "api-server", 5433, "api-server-db", "postgres")
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if !exactMatch {
			t.Fatal("Expected exact container match")
		}

		proxyEnvCount := len(deploy.Spec.Template.Spec.Containers[0].Env)
		if proxyEnvCount != 0 {
			t.Fatalf("Expected sidecar env to stay unchanged, got %d", proxyEnvCount)
		}

		appContainer := deploy.Spec.Template.Spec.Containers[1]
		expected := map[string]bool{
			"DB_HOST": false, "DB_USER": false, "DB_PASS": false, "PGRST_DB_URI": false, "DB_PORT": false,
			"DB_NAME": false, "DATABASE_URL": false,
		}
		for _, env := range appContainer.Env {
			if _, ok := expected[env.Name]; ok {
				expected[env.Name] = true
				if env.Name == "DB_PORT" && env.Value != "5433" {
					t.Errorf("Expected DB_PORT=5433, got %q", env.Value)
				}
				if env.Name == "DB_NAME" && env.Value != "api-server-db" {
					t.Errorf("Expected DB_NAME=api-server-db, got %q", env.Value)
				}
				if env.Name == "DATABASE_URL" && env.Value != postgresDatabaseURLTemplate {
					t.Errorf("Expected DATABASE_URL template, got %q", env.Value)
				}
			}
		}
		for name, found := range expected {
			if !found {
				t.Errorf("Expected env %s on target container", name)
			}
		}
	})

	t.Run("FallsBackToFirstContainerWhenPreferredMissing", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "only-container", Image: "myregistry/app:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "app-db-secret", "missing-app", 5432, "only-container-db", "postgres")
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if exactMatch {
			t.Fatal("Expected fallback because preferred container does not exist")
		}
		if len(deploy.Spec.Template.Spec.Containers[0].Env) != 7 {
			t.Fatalf("Expected 7 injected DB env vars, got %d", len(deploy.Spec.Template.Spec.Containers[0].Env))
		}
	})

	t.Run("NoPreferredContainerUsesFallback", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "myregistry/app:v1"},
						},
					},
				},
			},
		}

		changed, exactMatch := InjectDatabaseEnvVarsForContainer(deploy, "app-db-secret", "", 5432, "app-db", "postgres")
		if !changed {
			t.Fatal("Expected env injection changes")
		}
		if exactMatch {
			t.Fatal("Expected fallback semantics when preferred container is empty")
		}
	})
}
