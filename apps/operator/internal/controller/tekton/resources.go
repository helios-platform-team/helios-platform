package tekton

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GenerateServiceAccount creates the tekton-triggers-sa service account.
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

// GenerateClusterRoleBinding creates a ClusterRoleBinding for the tekton-triggers-sa.
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
