package triggers

import (
	"list"
	"helios.io/cue/definitions/tekton"
)

// =====================================================
// TRIGGER REGISTRY
// =====================================================

#TriggerRegistry: {
	// FIX: Remove 'tekton.' prefix. This is a local definition in the same package.
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
	// Inherit contract from Base Trigger Bundle (Validation + Defaults)
	bundleParams: tekton.#TriggerBundle.parameter

	// Lookup Bundle
	_bundleDef: #TriggerRegistry[triggerType]
	
	// Render Bundle
	_bundle: _bundleDef & {
		parameter: bundleParams
	}

	// Optional: Render Ingress if webhookDomain is provided
	// Use List Comprehension to force value binding for 'd'
	_ingress: [
		for d in [webhookDomain] if d != "" {
			(tekton.#WebhookIngress & {
				parameter: {
					name:          "\(bundleParams.appName)-webhook"
					namespace:     bundleParams.namespace
					webhookDomain: d
					serviceName:   "el-\(bundleParams.appName)-listener"
					servicePort:   8080
				}
			}).output
		}
	]

	// Combine Outputs using list.Concat
	outputs: list.Concat([_bundle.outputs, _ingress])
}