// Base template for Tekton Task resources.
// Concrete tasks extend this template.
package tekton

// #TektonTask: Base template for all Tekton Tasks
#TektonTask: {
	// === INPUT (required) ===
	parameter: {
		name:      string
		namespace: string
	}

	// === CONFIG (task-specific) ===
	config: {
		description?: string
		params: [...#TaskParam] | *[]
		workspaces: [...#TaskWorkspace] | *[]
		results: [...#TaskResult] | *[]
		steps: [...#TaskStep] // At least one step required
		volumes: [...#TaskVolume] | *[]
		stepTemplate?: #StepTemplate
	}

	// === OUTPUT (auto-generated) ===
	output: {
		apiVersion: #Defaults.tekton.apiVersion
		kind:       "Task"
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
			if len(config.results) > 0 {
				results: config.results
			}
			steps: config.steps
			if len(config.volumes) > 0 {
				volumes: config.volumes
			}
			if config.stepTemplate != _|_ {
				stepTemplate: config.stepTemplate
			}
		}
	}
}

// =====================================================
// SUPPORTING TYPES
// =====================================================

#TaskParam: {
	name:         string
	description?: string
	type?:        "string" | "array"
	default?:     string
}

#TaskWorkspace: {
	name:         string
	description?: string
	optional?:    bool
	readOnly?:    bool
	mountPath?:   string
}

#TaskResult: {
	name:         string
	description?: string
}

#TaskStep: {
	name:  string
	image: string

	// Either script OR command (not both)
	script?: string
	command?: [...string]
	args?: [...string]

	workingDir?: string
	env?: [...#EnvVar]
	volumeMounts?: [...#VolumeMount]
	resources?: #ResourceRequirements
	securityContext?: {
		runAsUser?:                int
		runAsGroup?:               int
		allowPrivilegeEscalation?: bool
	}
}

#StepTemplate: {
	env?: [...#EnvVar]
	resources?: #ResourceRequirements
	securityContext?: {...}
}

#TaskVolume: {
	name: string
	// One of the following volume sources
	secret?: {
		secretName: string
		items?: [...{key: string, path: string}]
	}
	emptyDir?: {}
	configMap?: {name: string}
	persistentVolumeClaim?: {claimName: string}
}

#EnvVar: {
	name:   string
	value?: string
	valueFrom?: {
		secretKeyRef?: {
			name: string
			key:  string
		}
		configMapKeyRef?: {
			name: string
			key:  string
		}
	}
}

#VolumeMount: {
	name:      string
	mountPath: string
	readOnly?: bool
	subPath?:  string
}

#ResourceRequirements: {
	limits?: {
		cpu?:    string
		memory?: string
	}
	requests?: {
		cpu?:    string
		memory?: string
	}
}
