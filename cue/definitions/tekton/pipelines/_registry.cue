// Pipeline registry for managing multiple pipeline definitions.
// Provides a central lookup and rendering mechanism.
package tekton

// =====================================================
// PIPELINE REGISTRY
// Central registry for all pipeline definitions
// =====================================================

// #PipelineDefinition: Schema for a registered pipeline
#PipelineDefinition: {
	// Pipeline metadata
	name:         string
	description?: string

	// Pipeline configuration function
	// Takes namespace and returns configured pipeline
	config: {
		description?: string
		params: [...] | *[]
		workspaces: [...] | *[]
		tasks: [...]
		finally?: [...]
	}
}

// #PipelineRegistry: Map of pipeline names to their definitions
#PipelineRegistry: {
	[Name=string]: #PipelineDefinition & {name: Name}
}

// Global registry instance - pipelines register themselves here
Registry: #PipelineRegistry & {
	// Pipelines will be added here via unification
	// e.g., "from-code-to-cluster": FromCodeToClusterPipeline
}

// =====================================================
// RENDER HELPER
// Renders a pipeline from the registry
// =====================================================

// #RenderPipeline: Helper to render a pipeline from registry
#RenderPipeline: {
	// Input parameters
	_pipelineName: string
	_namespace:    string

	// Lookup pipeline from registry  
	_definition: Registry[_pipelineName]

	// Use #TektonPipeline base to generate output
	_pipeline: #TektonPipeline & {
		parameter: {
			name:      _pipelineName
			namespace: _namespace
		}
		config: _definition.config
	}

	// Output the rendered pipeline
	output: _pipeline.output
}

// #RenderAllPipelines: Renders all pipelines in registry for a namespace
#RenderAllPipelines: {
	_namespace: string

	pipelines: {
		for name, _ in Registry {
			(name): (#RenderPipeline & {
				_pipelineName: name
				_namespace:    _namespace
			}).output
		}
	}
}
