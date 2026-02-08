// Base templates for Tekton Trigger resources.
// Includes TriggerBinding, TriggerTemplate, EventListener.
package tekton

// =====================================================
// TRIGGER BINDING
// =====================================================

#TektonTriggerBinding: {
	// === INPUT ===
	parameter: {
		name:      string
		namespace: string
	}

	// === CONFIG ===
	config: {
		params: [...{
			name:  string
			value: string
		}]
	}

	// === OUTPUT ===
	output: {
		apiVersion: #Defaults.tekton.triggerVersion
		kind:       "TriggerBinding"
		metadata: {
			name:      parameter.name
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.name}).labels
		}
		spec: {
			params: config.params
		}
	}
}

// =====================================================
// TRIGGER TEMPLATE
// =====================================================

#TektonTriggerTemplate: {
	// === INPUT ===
	parameter: {
		name:      string
		namespace: string
	}

	// === CONFIG ===
	config: {
		params: [...{
			name:         string
			description?: string
			default?:     string
		}]
		resourcetemplates: [...{...}] // PipelineRun specs
	}

	// === OUTPUT ===
	output: {
		apiVersion: #Defaults.tekton.triggerVersion
		kind:       "TriggerTemplate"
		metadata: {
			name:      parameter.name
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.name}).labels
		}
		spec: {
			params:            config.params
			resourcetemplates: config.resourcetemplates
		}
	}
}

// =====================================================
// EVENT LISTENER
// =====================================================

#TektonEventListener: {
	// === INPUT ===
	parameter: {
		name:      string
		namespace: string
	}

	// === CONFIG ===
	config: {
		serviceAccountName: string | *#Defaults.tekton.serviceAccount
		triggers: [...#EventListenerTrigger]
	}

	// === OUTPUT ===
	output: {
		apiVersion: #Defaults.tekton.triggerVersion
		kind:       "EventListener"
		metadata: {
			name:      parameter.name
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.name}).labels
		}
		spec: {
			serviceAccountName: config.serviceAccountName
			triggers:           config.triggers
		}
	}
}

#EventListenerTrigger: {
	name: string
	bindings: [...{
		ref:   string
		kind?: "TriggerBinding" | "ClusterTriggerBinding"
	}]
	template: {
		ref: string
	}
	interceptors?: [...#Interceptor]
}

#Interceptor: {
	ref: {
		name: string
		kind: "Interceptor" | "ClusterInterceptor"
	}
	params?: [...{
		name:  string
		value: _ // Can be string, object, or array
	}]
}

// =====================================================
// TRIGGER BUNDLE
// A bundle groups related trigger resources together
// =====================================================

#TriggerBundle: {
	// === INPUT ===
	parameter: {
		appName:        string
		namespace:      string
		pipelineName:   string
		webhookDomain:  string
		webhookSecret:  string
		gitRepo:        string
		gitBranch:      string | *"main"
		imageRepo:      string
		gitopsRepo:     string
		gitopsPath:     string
		gitopsBranch:   string | *"main"
		gitopsSecret:   string | *"github-credentials"
		pvcName:        string | *"shared-workspace-pvc"
		contextSubpath: string | *""
	}

	// === OUTPUTS (multiple resources) ===
	outputs: [...{...}]
}

// =====================================================
// INGRESS FOR WEBHOOK
// =====================================================

#WebhookIngress: {
	// === INPUT ===
	parameter: {
		name:          string
		namespace:     string
		webhookDomain: string
		serviceName:   string
		servicePort:   int | *8080
	}

	// === OUTPUT ===
	output: {
		apiVersion: "networking.k8s.io/v1"
		kind:       "Ingress"
		metadata: {
			name:      parameter.name
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.name}).labels
			annotations: {
				"nginx.ingress.kubernetes.io/ssl-redirect": "false"
			}
		}
		spec: {
			ingressClassName: "nginx"
			rules: [{
				host: parameter.webhookDomain
				http: paths: [{
					path:     "/hooks/\(parameter.name)"
					pathType: "Prefix"
					backend: service: {
						name: parameter.serviceName
						port: number: parameter.servicePort
					}
				}]
			}]
		}
	}
}
