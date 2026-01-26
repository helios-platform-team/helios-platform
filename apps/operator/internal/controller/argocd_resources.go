package controller

import (
	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenerateArgoApplication creates an ArgoCD Application manifest
func GenerateArgoApplication(heliosApp *appv1alpha1.HeliosApp) (*unstructured.Unstructured, error) {
	appName := heliosApp.Name + "-argocd"
	targetNamespace := heliosApp.Spec.ArgoCDNamespace
	if targetNamespace == "" {
		targetNamespace = "argocd"
	}
	project := heliosApp.Spec.ArgoCDProject
	if project == "" {
		project = "default"
	}
	gitOpsBranch := heliosApp.Spec.GitOpsBranch
	if gitOpsBranch == "" {
		gitOpsBranch = "main"
	}

	app := map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]any{
			"name":      appName,
			"namespace": targetNamespace,
			"labels": map[string]any{
				"app.kubernetes.io/name":       heliosApp.Name,
				"app.kubernetes.io/managed-by": "helios-operator",
			},
		},
		"spec": map[string]any{
			"project": project,
			"source": map[string]any{
				"repoURL":        heliosApp.Spec.GitOpsRepo,
				"targetRevision": gitOpsBranch,
				"path":           heliosApp.Spec.GitOpsPath,
			},
			"destination": map[string]any{
				"server":    "https://kubernetes.default.svc",
				"namespace": heliosApp.Namespace,
			},
			"syncPolicy": map[string]any{
				"automated": map[string]any{
					"prune":    true,
					"selfHeal": true,
				},
				"syncOptions": []string{
					"CreateNamespace=true",
				},
			},
		},
	}

	return &unstructured.Unstructured{Object: app}, nil
}
