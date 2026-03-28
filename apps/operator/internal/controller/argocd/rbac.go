package argocd

import (
	"cmp"
	"fmt"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenerateSyncRole returns a Role in the Argo CD namespace scoped to the Application name.
func GenerateSyncRole(app *appv1alpha1.HeliosApp) *unstructured.Unstructured {
	argoNS := cmp.Or(app.Spec.ArgoCDNamespace, "argocd")
	appName := app.Name + "-argocd"
	roleName := fmt.Sprintf("helios-argocd-sync-%s", app.Name)
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "Role",
			"metadata": map[string]any{
				"name":      roleName,
				"namespace": argoNS,
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "helios-operator",
					"helios.io/heliosapp":          fmt.Sprintf("%s/%s", app.Namespace, app.Name),
				},
			},
			"rules": []any{
				map[string]any{
					"apiGroups":     []any{"argoproj.io"},
					"resources":     []any{"applications"},
					"resourceNames": []any{appName},
					"verbs":         []any{"get", "patch"},
				},
			},
		},
	}
}

// GenerateSyncRoleBinding binds the pipeline ServiceAccount (app namespace) to the sync Role.
func GenerateSyncRoleBinding(app *appv1alpha1.HeliosApp) *unstructured.Unstructured {
	argoNS := cmp.Or(app.Spec.ArgoCDNamespace, "argocd")
	roleName := fmt.Sprintf("helios-argocd-sync-%s", app.Name)
	rbName := roleName
	saName := cmp.Or(app.Spec.ServiceAccount, "default")
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "RoleBinding",
			"metadata": map[string]any{
				"name":      rbName,
				"namespace": argoNS,
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "helios-operator",
					"helios.io/heliosapp":          fmt.Sprintf("%s/%s", app.Namespace, app.Name),
				},
			},
			"roleRef": map[string]any{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "Role",
				"name":     roleName,
			},
			"subjects": []any{
				map[string]any{
					"kind":      "ServiceAccount",
					"name":      saName,
					"namespace": app.Namespace,
				},
			},
		},
	}
}
