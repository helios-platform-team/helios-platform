// Package tekton defines CUE schemas for Tekton CI/CD resources.
// This is the input schema that Go Operator will fill with HeliosApp data.
package tekton

// #TektonInput: Contract between Go Operator and CUE Engine.
// Maps 1:1 with HeliosAppSpec fields relevant to Tekton resources.
#TektonInput: {
	// === IDENTITY ===
	// appName must be a valid Kubernetes name (lowercase, alphanumeric, hyphens)
	appName:   string & =~"^[a-z][a-z0-9-]*$"
	namespace: string

	// === SOURCE CODE ===
	gitRepo:   string & =~"^https?://.*"
	gitBranch: string | *"main"

	// === CONTAINER IMAGE ===
	imageRepo: string // e.g. "docker.io/myuser/myapp"

	// === GITOPS ===
	gitopsRepo:      string
	gitopsPath:      string
	gitopsBranch:    string | *"main"
	gitopsSecretRef: string | *"github-credentials"

	// === WEBHOOK (optional - triggers only created if set) ===
	webhookDomain?: string
	webhookSecret:  string | *"github-webhook-secret"

	// === PIPELINE CONFIG ===
	pipelineName:   string | *"from-code-to-cluster"
	pipelineType:   string | *"from-code-to-cluster" // For registry lookup
	triggerType:    string | *"github-push"          // For registry lookup
	serviceAccount: string | *"default"
	pvcName:        string | *"shared-workspace-pvc"
	contextSubpath: string | *""

	// === APP CONFIG ===
	replicas: int & >=1 | *1
	port:     int & >=1 & <=65535 | *8080

	// === TESTING ===
	testCommand?: string // e.g. "npm test"

	// === SECRETS ===
	dockerSecret: string | *"docker-credentials"

	// === ARGOCD ===
	argoCDNamespace: string | *"argocd"
	argoCDProject:   string | *"default"
}
