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
						map[string]any{"ref": gitBindingName},
						map[string]any{"ref": defaultsBindingName},
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
						},
						"workspaces": []any{
							map[string]any{"name": "source-workspace", "persistentVolumeClaim": map[string]any{"claimName": "$(tt.params.pvc-name)"}},
							map[string]any{"name": "gitops-workspace", "persistentVolumeClaim": map[string]any{"claimName": "$(tt.params.pvc-name)"}},
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
