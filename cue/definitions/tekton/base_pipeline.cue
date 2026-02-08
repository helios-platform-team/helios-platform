// Base template for Tekton Pipeline resources.
// Concrete pipelines extend this template.
package tekton

// #TektonPipeline: Base template for all Tekton Pipelines
#TektonPipeline: {
	// === INPUT (required) ===
	parameter: {
		name:      string
		namespace: string
	}

	// === CONFIG (pipeline-specific) ===
	config: {
		description?: string
		params: [...#PipelineParam] | *[]
		workspaces: [...#PipelineWorkspace] | *[]
		tasks: [...#PipelineTask] // At least one task required
		finally?: [...#PipelineTask]
	}

	// === OUTPUT (auto-generated) ===
	output: {
		apiVersion: #Defaults.tekton.apiVersion
		kind:       "Pipeline"
		metadata: {
			name:      parameter.name
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.name}).labels
		}
		spec: {
			if config.description != _|_ {
				description: config.description
			}
			if len(config.params) > 0 {
				params: config.params
			}
			if len(config.workspaces) > 0 {
				workspaces: config.workspaces
			}
			tasks: config.tasks
			if config.finally != _|_ {
				if len(config.finally) > 0 {
					finally: config.finally
				}
			}
		}
	}
}

// =====================================================
// SUPPORTING TYPES
// =====================================================

#PipelineParam: {
	name:         string
	description?: string
	type?:        "string" | "array"
	default?:     string
}

#PipelineWorkspace: {
	name:         string
	description?: string
	optional?:    bool
}

#PipelineTask: {
	name: string
	taskRef: {name: string}

	// Execution order
	runAfter?: [...string]

	// Parameter bindings
	params?: [...{
		name:  string
		value: string
	}]

	// Workspace bindings
	workspaces?: [...{
		name:      string
		workspace: string
	}]

	// Conditions
	when?: [...{
		input:    string
		operator: "in" | "notin"
		values: [...string]
	}]

	// Timeout
	timeout?: string

	// Retry
	retries?: int
}

// =====================================================
// PIPELINERUN TEMPLATE
// =====================================================

#TektonPipelineRun: {
	// === INPUT (required) ===
	parameter: {
		name:          string
		namespace:     string
		pipelineRef:   string
		generateName?: string
	}

	// === CONFIG ===
	config: {
		params?: [...{name: string, value: string}]
		workspaces?: [...#PipelineRunWorkspace]
		serviceAccountName?: string
		timeout?:            string
		podTemplate?: {
			securityContext?: {...}
			nodeSelector?: {[string]: string}
		}
	}

	// === OUTPUT ===
	output: {
		apiVersion: #Defaults.tekton.apiVersion
		kind:       "PipelineRun"
		metadata: {
			if parameter.generateName != _|_ {
				generateName: parameter.generateName
			}
			if parameter.generateName == _|_ {
				name: parameter.name
			}
			namespace: parameter.namespace
			labels: (#AppLabels & {_appName: parameter.pipelineRef}).labels
		}
		spec: {
			pipelineRef: name: parameter.pipelineRef
			if config.params != _|_ && len(config.params) > 0 {
				params: config.params
			}
			if config.workspaces != _|_ && len(config.workspaces) > 0 {
				workspaces: config.workspaces
			}
			if config.serviceAccountName != _|_ {
				serviceAccountName: config.serviceAccountName
			}
			if config.timeout != _|_ {
				timeout: config.timeout
			}
			if config.podTemplate != _|_ {
				podTemplate: config.podTemplate
			}
		}
	}
}

#PipelineRunWorkspace: {
	name: string
	// One of the following
	persistentVolumeClaim?: {claimName: string}
	volumeClaimTemplate?: {...}
	emptyDir?: {}
	secret?: {secretName: string}
	configMap?: {name: string}
}
