// tekton_resources.go contains the remaining Go-based Tekton resource generators
// that are NOT yet handled by the CUE engine.
//
// After the CUE migration (T1-T5), these functions remain because:
// - PipelineRun: Generated on-the-fly with unique timestamps (not a static resource)
// - ServiceAccount/RoleBinding/ClusterRoleBinding: Cluster infrastructure, not app-lifecycle resources
//
// All other Tekton resources (Tasks, Pipeline, TriggerBinding, TriggerTemplate,
// EventListener, Ingress) are now rendered by the CUE engine via TektonRenderer.
package controller

import (
	"cmp"
	"encoding/json"
	"fmt"
	"time"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GeneratePipelineRunForManifestGeneration creates a PipelineRun to generate manifests.
// This remains in Go because PipelineRuns are ephemeral (unique timestamp per run),
// unlike the static resources that CUE handles.
func GeneratePipelineRunForManifestGeneration(heliosApp *appv1alpha1.HeliosApp, pipelineName string) (*unstructured.Unstructured, error) {
	timestamp := time.Now().Format("20060102-150405")
	prName := fmt.Sprintf("%s-manifest-%s", heliosApp.Name, timestamp)

	contextSubpath := heliosApp.Spec.ContextSubpath

	gitOpsBranch := cmp.Or(heliosApp.Spec.GitOpsBranch, "main")

	params := []any{
		map[string]any{"name": "app-repo-url", "value": heliosApp.Spec.GitRepo},
		map[string]any{"name": "app-repo-revision", "value": heliosApp.Spec.GitBranch},
		map[string]any{"name": "image-repo", "value": heliosApp.Spec.ImageRepo},
		map[string]any{"name": "GITOPS_REPO_URL", "value": heliosApp.Spec.GitOpsRepo},
		map[string]any{"name": "MANIFEST_PATH", "value": heliosApp.Spec.GitOpsPath},
		map[string]any{"name": "GITOPS_REPO_BRANCH", "value": gitOpsBranch},
		map[string]any{"name": "CONTEXT_SUBPATH", "value": contextSubpath},
		map[string]any{"name": "replicas", "value": fmt.Sprintf("%d", heliosApp.Spec.Replicas)},
		map[string]any{"name": "port", "value": fmt.Sprintf("%d", heliosApp.Spec.Port)},
		map[string]any{"name": "test-command", "value": heliosApp.Spec.TestCommand},
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
				"janus-idp.io/tekton":          heliosApp.Name,
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
					"volumeClaimTemplate": map[string]any{
						"spec": map[string]any{
							"accessModes": []any{"ReadWriteOnce"},
							"resources": map[string]any{
								"requests": map[string]any{
									"storage": "1Gi",
								},
							},
						},
					},
				},
				map[string]any{
					"name": "gitops-workspace",
					"volumeClaimTemplate": map[string]any{
						"spec": map[string]any{
							"accessModes": []any{"ReadWriteOnce"},
							"resources": map[string]any{
								"requests": map[string]any{
									"storage": "1Gi",
								},
							},
						},
					},
				},
			},
		},
	}
	return &unstructured.Unstructured{Object: pr}, nil
}

// GenerateServiceAccount creates the tekton-triggers-sa service account.
// This remains in Go as it's cluster infrastructure (RBAC), not app-lifecycle.
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

// GenerateRoleBinding creates a RoleBinding for the tekton-triggers-sa to admin role.
// This remains in Go as it's cluster infrastructure (RBAC), not app-lifecycle.
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

// GenerateClusterRoleBinding creates a ClusterRoleBinding for the tekton-triggers-sa
// to cluster-level permissions.
// This remains in Go as it's cluster infrastructure (RBAC), not app-lifecycle.
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
