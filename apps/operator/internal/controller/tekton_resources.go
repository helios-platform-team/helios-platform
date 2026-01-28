package controller

import (
	"encoding/json"
	"fmt"
	"time"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenerateEventListener creates a Tekton EventListener manifest
func GenerateEventListener(name, namespace, triggerName, gitBindingName, defaultsBindingName, templateName, githubSecret string) (*unstructured.Unstructured, error) {
	el := map[string]any{
		"apiVersion": "triggers.tekton.dev/v1beta1",
		"kind":       "EventListener",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"serviceAccountName": "tekton-triggers-sa",
			"triggers": []any{
				map[string]any{
					"name": triggerName,
					"bindings": []any{
						map[string]any{"ref": gitBindingName, "kind": "TriggerBinding"},
						map[string]any{"ref": defaultsBindingName, "kind": "TriggerBinding"},
					},
					"template": map[string]any{
						"ref": templateName,
					},
					"interceptors": []any{
						map[string]any{
							"ref": map[string]any{
								"name": "github",
								"kind": "ClusterInterceptor",
							},
							"params": []any{
								map[string]any{"name": "secretRef", "value": map[string]any{
									"secretName": githubSecret,
									"secretKey":  "secretToken",
								}},
								map[string]any{"name": "eventTypes", "value": []any{"push"}},
							},
						},
					},
				},
			},
		},
	}
	return &unstructured.Unstructured{Object: el}, nil
}

// GenerateTriggerBinding creates a Tekton TriggerBinding manifest
func GenerateTriggerBinding(name, namespace string) (*unstructured.Unstructured, error) {
	tb := map[string]any{
		"apiVersion": "triggers.tekton.dev/v1beta1",
		"kind":       "TriggerBinding",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"params": []any{
				map[string]any{"name": "git-repo-url", "value": "$(body.repository.clone_url)"},
				map[string]any{"name": "git-revision", "value": "$(body.after)"},
				map[string]any{"name": "git-repo-name", "value": "$(body.repository.name)"},
			},
		},
	}
	return &unstructured.Unstructured{Object: tb}, nil
}

// GenerateDefaultsTriggerBinding creates a TriggerBinding with default values from HeliosApp
func GenerateDefaultsTriggerBinding(name, namespace string, app *appv1alpha1.HeliosApp) (*unstructured.Unstructured, error) {
	pvcName := app.Spec.PVCName
	if pvcName == "" {
		pvcName = "shared-workspace-pvc"
	}
	contextSubpath := app.Spec.ContextSubpath
	if contextSubpath == "" {
		contextSubpath = ""
	}
	gitOpsBranch := app.Spec.GitOpsBranch
	if gitOpsBranch == "" {
		gitOpsBranch = "main"
	}
	tb := map[string]any{
		"apiVersion": "triggers.tekton.dev/v1beta1",
		"kind":       "TriggerBinding",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"params": []any{
				map[string]any{"name": "image-repo", "value": app.Spec.ImageRepo},
				map[string]any{"name": "gitops-repo-url", "value": app.Spec.GitOpsRepo},
				map[string]any{"name": "manifest-path-in-gitops-repo", "value": app.Spec.GitOpsPath},
				map[string]any{"name": "gitops-repo-branch", "value": gitOpsBranch},
				map[string]any{"name": "pvc-name", "value": pvcName},
				map[string]any{"name": "context-subpath", "value": contextSubpath},
				map[string]any{"name": "replicas", "value": fmt.Sprintf("%d", app.Spec.Replicas)},
				map[string]any{"name": "port", "value": fmt.Sprintf("%d", app.Spec.Port)},
				map[string]any{"name": "gitops-secret-name", "value": app.Spec.GitOpsSecretRef},
			},
		},
	}
	return &unstructured.Unstructured{Object: tb}, nil
}

// GenerateTriggerTemplate creates a Tekton TriggerTemplate manifest
func GenerateTriggerTemplate(name, namespace, pipelineRunName, pipelineName, serviceAccount string, workspace map[string]any) (*unstructured.Unstructured, error) {
	tt := map[string]any{
		"apiVersion": "triggers.tekton.dev/v1beta1",
		"kind":       "TriggerTemplate",
		"metadata": map[string]any{
			"name":      name,
			"namespace": namespace,
		},
		"spec": map[string]any{
			"params": []any{
				map[string]any{"name": "git-repo-url"},
				map[string]any{"name": "git-revision"},
				map[string]any{"name": "image-repo"},
				map[string]any{"name": "gitops-repo-url"},
				map[string]any{"name": "gitops-repo-branch"},
				map[string]any{"name": "manifest-path-in-gitops-repo"},
				map[string]any{"name": "pvc-name"},
				map[string]any{"name": "context-subpath"},
				map[string]any{"name": "replicas"},
				map[string]any{"name": "port"},
				map[string]any{"name": "gitops-secret-name"},
			},
			"resourcetemplates": []any{
				map[string]any{
					"apiVersion": "tekton.dev/v1beta1",
					"kind":       "PipelineRun",
					"metadata": map[string]any{
						"generateName": pipelineRunName + "-",
					},
					"spec": map[string]any{
						"pipelineRef": map[string]any{
							"name": pipelineName,
						},
						"serviceAccountName": serviceAccount,
						"params": []any{
							map[string]any{"name": "app-repo-url", "value": "$(tt.params.git-repo-url)"},
							map[string]any{"name": "app-repo-revision", "value": "$(tt.params.git-revision)"},
							map[string]any{"name": "image-repo", "value": "$(tt.params.image-repo)"},
							map[string]any{"name": "gitops-repo-url", "value": "$(tt.params.gitops-repo-url)"},
							map[string]any{"name": "manifest-path-in-gitops-repo", "value": "$(tt.params.manifest-path-in-gitops-repo)"},
							map[string]any{"name": "gitops-repo-branch", "value": "$(tt.params.gitops-repo-branch)"},
							map[string]any{"name": "context-subpath", "value": "$(tt.params.context-subpath)"},
							map[string]any{"name": "replicas", "value": "$(tt.params.replicas)"},
							map[string]any{"name": "port", "value": "$(tt.params.port)"},
							map[string]any{"name": "gitops-secret-name", "value": "$(tt.params.gitops-secret-name)"},
						},
						"workspaces": []any{
							map[string]any{"name": "source-workspace", "persistentVolumeClaim": map[string]any{"claimName": "$(tt.params.pvc-name)"}},
							map[string]any{"name": "gitops-workspace", "persistentVolumeClaim": map[string]any{"claimName": "$(tt.params.pvc-name)"}},
							map[string]any{"name": "git-credentials-workspace", "secret": map[string]any{"secretName": "$(tt.params.gitops-secret-name)"}},
						},
						"timeouts": map[string]any{
							"pipeline": "1h",
						},
					},
				},
			},
		},
	}
	return &unstructured.Unstructured{Object: tt}, nil
}

// GeneratePipelineRunForManifestGeneration creates a PipelineRun to generate manifests
func GeneratePipelineRunForManifestGeneration(heliosApp *appv1alpha1.HeliosApp, pipelineName string) (*unstructured.Unstructured, error) {
	timestamp := time.Now().Format("20060102-150405")
	prName := fmt.Sprintf("%s-manifest-%s", heliosApp.Name, timestamp)

	contextSubpath := heliosApp.Spec.ContextSubpath
	if contextSubpath == "" {
		contextSubpath = "" // Default to empty string (Dockerfile at root)
	}

	gitOpsBranch := heliosApp.Spec.GitOpsBranch
	if gitOpsBranch == "" {
		gitOpsBranch = "main"
	}

	params := []any{
		map[string]any{"name": "app-repo-url", "value": heliosApp.Spec.GitRepo},
		map[string]any{"name": "app-repo-revision", "value": heliosApp.Spec.GitBranch},
		map[string]any{"name": "image-repo", "value": heliosApp.Spec.ImageRepo},
		map[string]any{"name": "gitops-repo-url", "value": heliosApp.Spec.GitOpsRepo},
		map[string]any{"name": "manifest-path-in-gitops-repo", "value": heliosApp.Spec.GitOpsPath},
		map[string]any{"name": "gitops-repo-branch", "value": gitOpsBranch},
		map[string]any{"name": "context-subpath", "value": contextSubpath},
		map[string]any{"name": "replicas", "value": fmt.Sprintf("%d", heliosApp.Spec.Replicas)},
		map[string]any{"name": "port", "value": fmt.Sprintf("%d", heliosApp.Spec.Port)},
	}

	// Serialize Env and Resources to JSON
	envJSON, err := json.Marshal(heliosApp.Spec.Env)
	if err != nil {
		envJSON = []byte("[]")
	}
	params = append(params, map[string]any{"name": "env-vars", "value": string(envJSON)})

	resourcesJSON, err := json.Marshal(heliosApp.Spec.Resources)
	if err != nil {
		resourcesJSON = []byte("{}")
	}
	params = append(params, map[string]any{"name": "resources", "value": string(resourcesJSON)})

	// PVC workspace - Pipeline expects two workspaces
	pvcName := heliosApp.Spec.PVCName
	if pvcName == "" {
		pvcName = "pvc-" + heliosApp.Name
	}

	pr := map[string]any{
		"apiVersion": "tekton.dev/v1beta1",
		"kind":       "PipelineRun",
		"metadata": map[string]any{
			"name":      prName,
			"namespace": heliosApp.Namespace,
			"labels": map[string]any{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/managed-by": "helios-operator",
				"helios.io/pipeline-type":      "manifest-generation",
			},
		},
		"spec": map[string]any{
			"pipelineRef": map[string]any{
				"name": pipelineName,
			},
			"serviceAccountName": heliosApp.Spec.ServiceAccount,
			"params":             params,
			"workspaces": []any{
				map[string]any{
					"name": "source-workspace",
					"persistentVolumeClaim": map[string]any{
						"claimName": pvcName,
					},
				},
				map[string]any{
					"name": "gitops-workspace",
					"persistentVolumeClaim": map[string]any{
						"claimName": pvcName,
					},
				},
			},
		},
	}
	return &unstructured.Unstructured{Object: pr}, nil
}

// GenerateIngress creates an Ingress to expose the EventListener
func GenerateIngress(heliosApp *appv1alpha1.HeliosApp, eventListenerName string) (*unstructured.Unstructured, error) {
	if heliosApp.Spec.WebhookDomain == "" {
		return nil, nil // No ingress needed if domain not specified
	}

	ingressName := heliosApp.Name + "-webhook-ingress"
	path := fmt.Sprintf("/hooks/%s", heliosApp.Name)
	pathType := "Prefix"

	// EventListener creates a service named el-<eventListenerName>
	serviceName := "el-" + eventListenerName

	ing := map[string]any{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "Ingress",
		"metadata": map[string]any{
			"name":      ingressName,
			"namespace": heliosApp.Namespace,
			"annotations": map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
			"labels": map[string]any{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/managed-by": "helios-operator",
			},
		},
		"spec": map[string]any{
			"rules": []any{
				map[string]any{
					"host": heliosApp.Spec.WebhookDomain,
					"http": map[string]any{
						"paths": []any{
							map[string]any{
								"path":     path,
								"pathType": pathType,
								"backend": map[string]any{
									"service": map[string]any{
										"name": serviceName,
										"port": map[string]any{
											"number": int64(8080), // Default Tekton EventListener port
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &unstructured.Unstructured{Object: ing}, nil
}

// GenerateServiceAccount creates the tekton-triggers-sa service account
func GenerateServiceAccount(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name":      "tekton-triggers-sa",
				"namespace": namespace,
			},
		},
	}
}

// GenerateRoleBinding creates a RoleBinding for the tekton-triggers-sa to admin role
func GenerateRoleBinding(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]any{
				"name":      "tekton-triggers-sa-admin",
				"namespace": namespace,
			},
			"roleRef": map[string]any{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "admin",
			},
			"subjects": []any{
				map[string]any{
					"kind":      "ServiceAccount",
					"name":      "tekton-triggers-sa",
					"namespace": namespace,
				},
			},
		},
	}
}

func GenerateGitCloneTask(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Task",
			"metadata": map[string]any{
				"name":      "git-clone",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"params": []any{
					map[string]any{"name": "url", "description": "Repository URL to clone."},
					map[string]any{"name": "revision", "description": "The git revision to clone", "default": "main"},
				},
				"workspaces": []any{
					map[string]any{"name": "output", "description": "The workspace where the source code will be cloned."},
				},
				"steps": []any{
					map[string]any{
						"name":  "clone",
						"image": "alpine/git:latest",
						"script": `#!/bin/sh
set -e
# Clean the workspace if it exists
echo "Cleaning workspace: $(workspaces.output.path)"
rm -rf $(workspaces.output.path)/*
rm -rf $(workspaces.output.path)/.[!.]*
# Clone the repository
echo "Cloning $(params.url) to $(workspaces.output.path)"
git clone $(params.url) $(workspaces.output.path)
# Checkout the specified revision
cd $(workspaces.output.path)
echo "Checking out $(params.revision)"
git checkout $(params.revision)
echo "Git clone completed successfully"
`,
					},
				},
			},
		},
	}
}

func GenerateKanikoBuildTask(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Task",
			"metadata": map[string]any{
				"name":      "kaniko-build",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"params": []any{
					map[string]any{"name": "IMAGE"},
					map[string]any{"name": "DOCKERFILE", "default": "Dockerfile"},
					map[string]any{"name": "CONTEXT_SUBPATH", "default": "", "description": "Subdirectory within the workspace where the Dockerfile is located"},
					map[string]any{"name": "docker-secret", "default": "docker-credentials", "description": "Name of the secret containing docker credentials"},
				},
				"workspaces": []any{
					map[string]any{"name": "source"},
				},
				"results": []any{
					map[string]any{"name": "IMAGE_URL"},
				},
				"steps": []any{
					map[string]any{
						"name":  "build-and-push",
						"image": "gcr.io/kaniko-project/executor:latest",
						"env": []any{
							map[string]any{"name": "DOCKER_CONFIG", "value": "/kaniko/.docker"},
						},
						"command": []any{"/kaniko/executor"},
						"args": []any{
							"--dockerfile=$(params.DOCKERFILE)",
							"--context=$(workspaces.source.path)/$(params.CONTEXT_SUBPATH)",
							"--destination=$(params.IMAGE)",
							"--digest-file=/tekton/results/IMAGE_DIGEST",
						},
						"volumeMounts": []any{
							map[string]any{"name": "docker-config", "mountPath": "/kaniko/.docker"},
						},
					},
					map[string]any{
						"name":  "write-image-url",
						"image": "alpine:latest",
						"script": `#!/bin/sh
set -e
echo "$(params.IMAGE)@$(cat /tekton/results/IMAGE_DIGEST)" > $(results.IMAGE_URL.path)
`,
					},
				},
				"volumes": []any{
					map[string]any{
						"name": "docker-config",
						"secret": map[string]any{
							"secretName": "$(params.docker-secret)",
							"items": []any{
								map[string]any{"key": ".dockerconfigjson", "path": "config.json"},
							},
						},
					},
				},
			},
		},
	}
}

func GenerateGitUpdateManifestTask(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Task",
			"metadata": map[string]any{
				"name":      "git-update-manifest",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"params": []any{
					map[string]any{"name": "GITOPS_REPO_URL"},
					map[string]any{"name": "MANIFEST_PATH"},
					map[string]any{"name": "NEW_IMAGE_URL"},
					map[string]any{"name": "GITOPS_REPO_BRANCH"},
					map[string]any{"name": "REPLICAS", "default": "2"},
					map[string]any{"name": "PORT", "default": "8080"},
					map[string]any{"name": "GITOPS_SECRET_NAME", "default": "github-credentials"},
				},
				"workspaces": []any{
					map[string]any{"name": "gitops-repo"},
					map[string]any{"name": "git-credentials", "optional": true},
				},
				"steps": []any{
					map[string]any{
						"name":  "update-manifest",
						"image": "alpine:latest",
						"script": `#!/bin/sh
set -e
apk add --no-cache git yq
cd "$(workspaces.gitops-repo.path)"
echo "Cleaning workspace content..."
rm -rf ./*
rm -rf ./.??*
rm -rf .git
echo "Cloning repo into current directory..."
git clone "$(params.GITOPS_REPO_URL)" .
git checkout -B "$(params.GITOPS_REPO_BRANCH)"
git config user.email "tekton-pipeline@helios.dev"
git config user.name "Tekton Pipeline"
# Read credentials from mounted workspace (secret)
CREDS_PATH="$(workspaces.git-credentials.path)"
if [ -f "${CREDS_PATH}/username" ] && [ -f "${CREDS_PATH}/password" ]; then
  username=$(cat "${CREDS_PATH}/username")
  password=$(cat "${CREDS_PATH}/password")
  RAW_URL=$(echo "$(params.GITOPS_REPO_URL)" | sed 's|https://.*@|https://|')
  REPO_URL_WITH_AUTH="$(echo "$RAW_URL" | sed "s|https://|https://${username}:${password}@|")"
  git remote set-url origin "${REPO_URL_WITH_AUTH}"
  echo "Updated git remote with credentials from mounted secret."
else
    echo "WARNING: Git credentials secret not mounted or missing keys. Push might fail."
fi
export IMAGE_URL="$(params.NEW_IMAGE_URL)"
export REPLICAS="$(params.REPLICAS)"
export PORT="$(params.PORT)"
MANIFEST_PATH="$(params.MANIFEST_PATH)"
if echo "$MANIFEST_PATH" | grep -qvE '\.ya?ml$'; then
    echo "Path '$MANIFEST_PATH' treated as DIRECTORY."
    mkdir -p "$MANIFEST_PATH"
    DEP_FILE="$MANIFEST_PATH/deployment.yaml"
    SVC_FILE="$MANIFEST_PATH/service.yaml"
    MANIFEST_FILES="$DEP_FILE $SVC_FILE"
    APP_NAME=$(basename "$MANIFEST_PATH")
    if [ ! -f "$DEP_FILE" ]; then
        echo "Creating default manifests..."
        printf "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ${APP_NAME}\nspec:\n  replicas: ${REPLICAS}\n  selector:\n    matchLabels:\n      app: ${APP_NAME}\n  template:\n    metadata:\n      labels:\n        app: ${APP_NAME}\n    spec:\n      containers:\n        - name: app\n          image: ${IMAGE_URL}\n          ports:\n            - containerPort: ${PORT}\n" > "$DEP_FILE"
        printf "apiVersion: v1\nkind: Service\nmetadata:\n  name: ${APP_NAME}\nspec:\n  selector:\n    app: ${APP_NAME}\n  ports:\n    - protocol: TCP\n      port: ${PORT}\n      targetPort: ${PORT}\n  type: ClusterIP\n" > "$SVC_FILE"
    fi
else
    echo "Path '$MANIFEST_PATH' treated as FILE."
    mkdir -p "$(dirname "$MANIFEST_PATH")"
    MANIFEST_FILES="$MANIFEST_PATH"
    APP_NAME=$(basename "$MANIFEST_PATH" | sed 's/\.[^.]*$//')
    if [ ! -f "$MANIFEST_PATH" ]; then
        echo "Creating combined manifest file..."
        printf "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: ${APP_NAME}\nspec:\n  replicas: ${REPLICAS}\n  selector:\n    matchLabels:\n      app: ${APP_NAME}\n  template:\n    metadata:\n      labels:\n        app: ${APP_NAME}\n    spec:\n      containers:\n        - name: app\n          image: ${IMAGE_URL}\n          ports:\n            - containerPort: ${PORT}\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: ${APP_NAME}\nspec:\n  selector:\n    app: ${APP_NAME}\n  ports:\n    - protocol: TCP\n      port: ${PORT}\n      targetPort: ${PORT}\n  type: ClusterIP\n" > "$MANIFEST_PATH"
    fi
fi
for FILE in $MANIFEST_FILES; do
  if [ -f "$FILE" ]; then
      echo "Updating $FILE..."
      yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].image = env(IMAGE_URL)' "$FILE"
      yq -i 'select(.kind == "Deployment") .spec.replicas = env(REPLICAS)' "$FILE"
      yq -i 'select(.kind == "Deployment") .spec.template.spec.containers[].ports[0].containerPort = env(PORT)' "$FILE"
      yq -i 'select(.kind == "Service") .spec.ports[0].targetPort = env(PORT)' "$FILE"
  fi
done
git add .
if git diff-index --quiet HEAD --; then
    echo "No changes to commit"
else
    git commit -m "chore: Update image=${IMAGE_URL} [skip-ci]"
    git push origin "$(params.GITOPS_REPO_BRANCH)"
fi
`,
					},
				},
			},
		},
	}
}

func GeneratePipeline(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "tekton.dev/v1beta1",
			"kind":       "Pipeline",
			"metadata": map[string]any{
				"name":      "from-code-to-cluster",
				"namespace": namespace,
			},
			"spec": map[string]any{
				"params": []any{
					map[string]any{"name": "app-repo-url"},
					map[string]any{"name": "app-repo-revision"},
					map[string]any{"name": "image-repo"},
					map[string]any{"name": "gitops-repo-url"},
					map[string]any{"name": "manifest-path-in-gitops-repo"},
					map[string]any{"name": "gitops-repo-branch"},
					map[string]any{"name": "context-subpath", "default": ""},
					map[string]any{"name": "replicas", "default": "2"},
					map[string]any{"name": "port", "default": "8080"},
					map[string]any{"name": "docker-secret", "default": "docker-credentials"},
					map[string]any{"name": "gitops-secret-name", "default": "github-credentials"},
				},
				"workspaces": []any{
					map[string]any{"name": "source-workspace"},
					map[string]any{"name": "gitops-workspace"},
					map[string]any{"name": "git-credentials-workspace", "optional": true},
				},
				"tasks": []any{
					map[string]any{
						"name": "fetch-source-code",
						"taskRef": map[string]any{
							"name": "git-clone",
						},
						"workspaces": []any{
							map[string]any{"name": "output", "workspace": "source-workspace"},
						},
						"params": []any{
							map[string]any{"name": "url", "value": "$(params.app-repo-url)"},
							map[string]any{"name": "revision", "value": "$(params.app-repo-revision)"},
						},
					},
					map[string]any{
						"name":     "build-and-push-image",
						"runAfter": []any{"fetch-source-code"},
						"taskRef": map[string]any{
							"name": "kaniko-build",
						},
						"workspaces": []any{
							map[string]any{"name": "source", "workspace": "source-workspace"},
						},
						"params": []any{
							map[string]any{"name": "IMAGE", "value": "$(params.image-repo):$(params.app-repo-revision)"},
							map[string]any{"name": "CONTEXT_SUBPATH", "value": "$(params.context-subpath)"},
							map[string]any{"name": "docker-secret", "value": "$(params.docker-secret)"},
						},
					},
					map[string]any{
						"name":     "update-gitops-manifest",
						"runAfter": []any{"build-and-push-image"},
						"taskRef": map[string]any{
							"name": "git-update-manifest",
						},
						"workspaces": []any{
							map[string]any{"name": "gitops-repo", "workspace": "gitops-workspace"},
							map[string]any{"name": "git-credentials", "workspace": "git-credentials-workspace"},
						},
						"params": []any{
							map[string]any{"name": "GITOPS_REPO_URL", "value": "$(params.gitops-repo-url)"},
							map[string]any{"name": "MANIFEST_PATH", "value": "$(params.manifest-path-in-gitops-repo)"},
							map[string]any{"name": "NEW_IMAGE_URL", "value": "$(tasks.build-and-push-image.results.IMAGE_URL)"},
							map[string]any{"name": "GITOPS_REPO_BRANCH", "value": "$(params.gitops-repo-branch)"},
							map[string]any{"name": "REPLICAS", "value": "$(params.replicas)"},
							map[string]any{"name": "PORT", "value": "$(params.port)"},
							map[string]any{"name": "GITOPS_SECRET_NAME", "value": "$(params.gitops-secret-name)"},
						},
					},
				},
			},
		},
	}
}

// GenerateRoleBinding creates a RoleBinding for the tekton-triggers-sa to admin role

// NOTE: in a real production operator, these resources might be embedded strings or loaded from files.
// For this plan, we define them here to ensure they exist in the namespace.
// GenerateClusterRoleBinding creates a ClusterRoleBinding for the tekton-triggers-sa to cluster-level permissions
func GenerateClusterRoleBinding(namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]any{
				"name": "tekton-triggers-sa-clusterbinding-" + namespace,
			},
			"roleRef": map[string]any{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     "tekton-triggers-eventlistener-clusterroles",
			},
			"subjects": []any{
				map[string]any{
					"kind":      "ServiceAccount",
					"name":      "tekton-triggers-sa",
					"namespace": namespace,
				},
			},
		},
	}
}
