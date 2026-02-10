package engine


// Provide concrete values for the input
tektonInput: {
	appName:        "my-cool-app"
	namespace:      "default"
	
	// Source
	gitRepo:        "https://github.com/myuser/my-cool-app"
	gitBranch:      "main"
	
	// Image
	imageRepo:      "docker.io/myuser/my-cool-app"
	
	// GitOps
	gitopsRepo:     "https://github.com/myuser/gitops-repo"
	gitopsPath:     "./apps/my-cool-app"
	gitopsBranch:   "main"
	gitopsSecretRef: "github-credentials"
	
	// Pipeline Config
	pipelineType:   "from-code-to-cluster" // Try changing to "build-only" later!
	triggerType:    "github-push"
	
	// Webhook (Optional - uncomment to test ingress)
	webhookDomain: "hooks.helios.dev"
	
	// Other defaults
	serviceAccount: "tekton-sa"
	dockerSecret:   "docker-creds"
	webhookSecret:  "github-secret"
}