package argocd

import (
	"cmp"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/controller/shared"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenerateArgoApplication creates an ArgoCD Application manifest with PreSync hooks
// if the HeliosApp has a database trait for automatic database migrations.
func GenerateArgoApplication(heliosApp *appv1alpha1.HeliosApp) (*unstructured.Unstructured, error) {
	appName := heliosApp.Name + "-argocd"
	targetNamespace := cmp.Or(heliosApp.Spec.ArgoCDNamespace, "argocd")
	project := cmp.Or(heliosApp.Spec.ArgoCDProject, "default")
	gitOpsBranch := cmp.Or(heliosApp.Spec.GitOpsBranch, "main")

	spec := map[string]any{
		"project": project,
		"source": map[string]any{
			"repoURL":        shared.RewriteGiteaURL(heliosApp.Spec.GitOpsRepo),
			"targetRevision": gitOpsBranch,
			"path":           heliosApp.Spec.GitOpsPath,
			"directory": map[string]any{
				"include": "helios-app.yaml",
			},
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
			"syncOptions": []any{
				"CreateNamespace=true",
			},
		},
		"ignoreDifferences": []any{
			map[string]any{
				"group": "apps",
				"kind":  "Deployment",
				"jqPathExpressions": []any{
					`.spec.template.spec.containers[].env[]? | select(.name | test("^(DB_|DATABASE_URL$)"))`,
				},
			},
		},
	}

	// Add PreSync hook if database trait exists
	if HasDatabaseTrait(heliosApp) {
		spec["syncPolicy"] = map[string]any{
			"automated": map[string]any{
				"prune":    true,
				"selfHeal": true,
			},
			"syncOptions": []any{
				"CreateNamespace=true",
			},
		}

		// Add PreSync hook to application
		// Note: PreSync Job is created and managed by PreSyncReconciler
		// This is referenced via Job annotations, not stored in Application spec
		syncPolicy := spec["syncPolicy"].(map[string]any)
		syncPolicy["syncOptions"] = append(
			syncPolicy["syncOptions"].([]any),
			"SkipDryRunOnMissingResource=true",
		)
		spec["syncPolicy"] = syncPolicy
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
		"spec": spec,
	}

	return &unstructured.Unstructured{Object: app}, nil
}
