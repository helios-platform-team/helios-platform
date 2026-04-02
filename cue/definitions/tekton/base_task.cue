package tekton

// Base schema for Tekton Task using the Config/Output pattern
#TektonTask: {
	// Input configuration
	config: #Defaults

	// Task-specific parameters
	parameter: {
		name: string
		...
	}

	// Output Kubernetes Resource
	output: {
		apiVersion: "tekton.dev/v1beta1"
		kind:       "Task"
		metadata: {
			name: parameter.name
			// Allow additional fields like namespace, labels, annotations
			...
		}
		spec: {
			params?: [...#TaskParam]
			workspaces?: [...#TaskWorkspace]
			results?: [...#TaskResult]
			volumes?: [...#TaskVolume]
			steps: [...#TaskStep]
		}
	}
}

// --- Supporting Definitions ---

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
	entrypoint?: [...string]
	script?:  string
	env?:     [...#EnvVar]
	envFrom?: [...#EnvFromSource]
	volumeMounts?: [...#VolumeMount]
	workingDir?: string
	securityContext?: {
		runAsUser?:    int
		runAsGroup?:   int
		fsGroup?:      int
		runAsNonRoot?: bool
		privileged?:   bool
		...
	}
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

