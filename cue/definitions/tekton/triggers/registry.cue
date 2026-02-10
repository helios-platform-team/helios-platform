package triggers

import (
	"list"
	"helios.io/cue/definitions/tekton"
)

// =====================================================
// TRIGGER REGISTRY
// =====================================================

#TriggerRegistry: {
	"github-push": #GitHubPushTriggerBundle
}

// =====================================================
// HELPER: RENDER TRIGGERS
// Renders the requested TriggerBundle and optional Ingress
// =====================================================

#RenderTriggers: {
	// Input
	triggerType:   string
	webhookDomain: string | *""
	
	// Data passed to the Bundle
	bundleParams: {
		appName:        string
		namespace:      string
		pipelineName:   string
		webhookDomain:  string
		webhookSecret:  string
		gitRepo:        string
		gitBranch:      string
		imageRepo:      string
		gitopsRepo:     string
		gitopsPath:     string
		gitopsBranch:   string
		gitopsSecret:   string
		pvcName:        string
		contextSubpath: string
		replicas:       int
		port:           int
		testCommand:    string
		testImage:      string
		serviceAccount: string
		dockerSecret:   string
	}

	// Lookup Bundle
	_bundleDef: #TriggerRegistry[triggerType]
	
	// Render Bundle
	_bundle: _bundleDef & {
		parameter: bundleParams
	}

	// Optional: Render Ingress if webhookDomain is provided
	// FIX: Use List Comprehension to force value binding.
	// This ensures 'd' is treated as a concrete string inside the struct.
	_ingress: [
		for d in [webhookDomain] if d != "" {
			(tekton.#WebhookIngress & {
				parameter: {
					name:          "\(bundleParams.appName)-webhook"
					namespace:     bundleParams.namespace
					webhookDomain: d // Use the bound variable 'd'
					serviceName:   "el-\(bundleParams.appName)-listener"
					servicePort:   8080
				}
			}).output
		}
	]

	// Combine Outputs using list.Concat
	outputs: list.Concat([_bundle.outputs, _ingress])
}