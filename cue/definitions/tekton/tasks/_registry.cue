package tekton

// Registry mapping task names to their definitions
#TaskRegistry: {
	"git-clone":           #GitClone
	"kaniko-build":        #KanikoBuild
	"git-update-manifest": #GitUpdateManifest
}

// Helper to render a specific task with optional context injection
#RenderTask: {
	taskName: string
	namespace?: string
	
	// Lookup the task definition
	_task: #TaskRegistry[taskName]
	
	// Return the Kubernetes Resource (output field)
	output: _task.output & {
		let _ns = namespace
		metadata: {
			if _ns != _|_ {
				namespace: _ns
			}
		}
	}
}
