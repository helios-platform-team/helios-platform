package tekton

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"time"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/shared"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GeneratePipelineRun creates a PipelineRun to trigger the CI/CD pipeline.
// PipelineRuns are ephemeral (unique timestamp per run), unlike the static
// resources that CUE handles.
func GeneratePipelineRun(heliosApp *appv1alpha1.HeliosApp, pipelineName string, npmCachePVCName ...string) (*unstructured.Unstructured, error) {
	timestamp := time.Now().Format("20060102-150405.000")
	prName := fmt.Sprintf("%s-manifest-%s", heliosApp.Name, timestamp)

	contextSubpath := cmp.Or(heliosApp.Spec.ContextSubpath, "")
	appRepoRevision := cmp.Or(heliosApp.Spec.GitBranch, "main")
	gitOpsBranch := cmp.Or(heliosApp.Spec.GitOpsBranch, "main")
	serviceAccountName := cmp.Or(heliosApp.Spec.ServiceAccount, "default")
	gitOpsSecretRef := cmp.Or(heliosApp.Spec.GitOpsSecretRef, "helios-gitops-bot")
	argoNS := cmp.Or(heliosApp.Spec.ArgoCDNamespace, "argocd")

	replicas := heliosApp.Spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	port := heliosApp.Spec.Port
	if port <= 0 || port > 65535 {
		port = 8080
	}
	testImage := cmp.Or(heliosApp.Spec.TestImage, "node:24")

	params := make([]any, 0, 18)
	params = append(params,
		map[string]any{"name": "app-repo-url", "value": shared.RewriteGiteaURL(heliosApp.Spec.GitRepo)},
		map[string]any{"name": "app-repo-revision", "value": appRepoRevision},
		map[string]any{"name": "image-repo", "value": heliosApp.Spec.ImageRepo},
		map[string]any{"name": "gitops-repo-url", "value": shared.RewriteGiteaURL(heliosApp.Spec.GitOpsRepo)},
		map[string]any{"name": "manifest-path", "value": heliosApp.Spec.GitOpsPath},
		map[string]any{"name": "gitops-repo-branch", "value": gitOpsBranch},
		map[string]any{"name": "gitops-secret-ref", "value": gitOpsSecretRef},
		map[string]any{"name": "gitops-author-name", "value": "Helios Bot"},
		map[string]any{"name": "gitops-author-email", "value": "helios-bot@helios.local"},
		map[string]any{"name": "context-subpath", "value": contextSubpath},
		map[string]any{"name": "replicas", "value": fmt.Sprintf("%d", replicas)},
		map[string]any{"name": "port", "value": fmt.Sprintf("%d", port)},
		map[string]any{"name": "test-command", "value": heliosApp.Spec.TestCommand},
		map[string]any{"name": "test-image", "value": testImage},
		map[string]any{"name": "argocd-namespace", "value": argoNS},
		map[string]any{"name": "argocd-app-name", "value": heliosApp.Name + "-argocd"},
	)

	envJSON, err := json.Marshal(heliosApp.Spec.Env)
	if err != nil {
		log.Printf("Warning: failed to marshal Env: %v", err)
		envJSON = []byte("[]")
	}
	params = append(params, map[string]any{"name": "env-vars", "value": string(envJSON)})

	resourcesJSON, err := json.Marshal(heliosApp.Spec.Resources)
	if err != nil {
		log.Printf("Warning: failed to marshal Resources: %v", err)
		resourcesJSON = []byte("{}")
	}
	params = append(params, map[string]any{"name": "resources", "value": string(resourcesJSON)})

	npmCachePVC := ""
	if len(npmCachePVCName) > 0 {
		npmCachePVC = npmCachePVCName[0]
	}

	workspaceBindings := []any{
		map[string]any{
			"name": "source-workspace",
			"volumeClaimTemplate": map[string]any{
				"spec": map[string]any{
					"accessModes": []any{"ReadWriteOnce"},
					"resources":   map[string]any{"requests": map[string]any{"storage": "1Gi"}},
				},
			},
		},
		map[string]any{
			"name": "gitops-workspace",
			"volumeClaimTemplate": map[string]any{
				"spec": map[string]any{
					"accessModes": []any{"ReadWriteOnce"},
					"resources":   map[string]any{"requests": map[string]any{"storage": "1Gi"}},
				},
			},
		},
		map[string]any{
			"name": "npm-cache",
			"volumeClaimTemplate": map[string]any{
				"spec": map[string]any{
					"accessModes": []any{"ReadWriteOnce"},
					"resources":   map[string]any{"requests": map[string]any{"storage": "5Gi"}},
				},
			},
		},
	}

	if heliosApp.Spec.PVCName != "" {
		workspaceBindings = []any{
			map[string]any{
				"name":                  "source-workspace",
				"persistentVolumeClaim": map[string]any{"claimName": heliosApp.Spec.PVCName},
				"subPath":               "source",
			},
			map[string]any{
				"name":                  "gitops-workspace",
				"persistentVolumeClaim": map[string]any{"claimName": heliosApp.Spec.PVCName},
				"subPath":               "gitops",
			},
		}
		npmCacheBinding := map[string]any{
			"name": "npm-cache",
			"volumeClaimTemplate": map[string]any{
				"spec": map[string]any{
					"accessModes": []any{"ReadWriteOnce"},
					"resources":   map[string]any{"requests": map[string]any{"storage": "5Gi"}},
				},
			},
		}
		if npmCachePVC != "" {
			npmCacheBinding = map[string]any{
				"name":                  "npm-cache",
				"persistentVolumeClaim": map[string]any{"claimName": npmCachePVC},
			}
		}
		workspaceBindings = append(workspaceBindings, npmCacheBinding)
	} else if npmCachePVC != "" {
		// Replace the volumeClaimTemplate with a persistent PVC
		for i, wb := range workspaceBindings {
			if wbMap, ok := wb.(map[string]any); ok && wbMap["name"] == "npm-cache" {
				workspaceBindings[i] = map[string]any{
					"name":                  "npm-cache",
					"persistentVolumeClaim": map[string]any{"claimName": npmCachePVC},
				}
				break
			}
		}
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
				"janus-idp.io/tekton":          heliosApp.Name,
			},
		},
		"spec": map[string]any{
			"pipelineRef":        map[string]any{"name": pipelineName},
			"serviceAccountName": serviceAccountName,
			"params":             params,
			"workspaces":         workspaceBindings,
		},
	}
	return &unstructured.Unstructured{Object: pr}, nil
}
