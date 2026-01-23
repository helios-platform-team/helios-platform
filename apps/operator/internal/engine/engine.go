package engine

import (
	"fmt"

	heliosappv1 "github.com/hoangphuc841/helios-operator/api/v1"
)

// RenderManifest generates the Kubernetes manifest for the application
// forcing the "Infrastructure as Data" pattern (OAM-like)
func RenderManifest(app *heliosappv1.HeliosApp) (string, error) {
	// In a real scenario, this would use CUE or Helm Templating.
	// For this task, we generate a standard Deployment + Service YAML.

	manifest := fmt.Sprintf(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    app.kubernetes.io/managed-by: helios-operator
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: main
        image: %s
        ports:
        - containerPort: %d
        resources:
          requests:
            cpu: "%s"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
        env:
%s
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - port: 80
    targetPort: %d
  type: ClusterIP
`,
		app.Name, app.Namespace, app.Name,
		app.Spec.Replicas,
		app.Name,
		app.Name,
		app.Spec.ImageRepo,
		app.Spec.Port,
		"100m",                  // Default CPU request if not optimized
		renderEnv(app.Spec.Env), // Env vars helper
		app.Name, app.Namespace, app.Name,
		app.Name,
		app.Spec.Port,
	)

	return manifest, nil
}

func renderEnv(vars []heliosappv1.EnvVar) string {
	if len(vars) == 0 {
		return "          []"
	}
	result := ""
	for _, v := range vars {
		result += fmt.Sprintf("        - name: %s\n          value: \"%s\"\n", v.Name, v.Value)
	}
	return result
}
