package bases

// Base schema for Tekton Task
#TektonTask: {
	apiVersion: "tekton.dev/v1beta1"
	kind:       "Task"
	metadata: {
		name: string
	}
	spec: {
		params?: [...#TaskParam]
		workspaces?: [...#TaskWorkspace]
		results?: [...#TaskResult]
		volumes?: [...#TaskVolume]
		steps: [...#TaskStep]
	}
}

#TaskParam: {
	name:        string
	description?: string
	type?:       "string" | "array"
	default?:    string | [...string]
}

#TaskWorkspace: {
	name:        string
	description?: string
	mountPath?:  string
	readOnly?:   bool
	optional?:   bool
}

#TaskResult: {
	name:        string
	description?: string
}

#TaskVolume: {
	name: string
	// Simplified volume schema, can extend as needed
	secret?: {
		secretName: string
		items?: [...{
			key:  string
			path: string
		}]
	}
	configMap?: {
		name: string
	}
	emptyDir?: {}
}

#TaskStep: {
	name:    string
	image:   string
	command?: [...string]
	args?:    [...string]
	script?:  string
	env?:     [...#EnvVar]
	envFrom?: [...#EnvFromSource]
	volumeMounts?: [...#VolumeMount]
	workingDir?: string
}

#EnvFromSource: {
	prefix?: string
	configMapRef?: {
		name:      string
		optional?: bool
	}
	secretRef?: {
		name:      string
		optional?: bool
	}
}

#EnvVar: {
	name: string
	value?: string
	valueFrom?: {
		secretKeyRef?: {
			name: string
			key:  string
			optional?: bool
		}
		configMapKeyRef?: {
			name: string
			key:  string
			optional?: bool
		}
	}
}

#VolumeMount: {
	name:      string
	mountPath: string
	readOnly?: bool
	subPath?:  string
}

// Common Parameters shared across tasks
#CommonParams: {
	git_url: {
		name:        "url"
		description: "Repository URL to clone."
		type:        "string"
	}
	git_revision: {
		name:        "revision"
		description: "The git revision to clone (e.g., branch, tag, or commit SHA)."
		type:        "string"
		default:     "main"
	}
}
