// Pipeline registry for managing multiple pipeline definitions.
// Provides a central lookup and rendering mechanism.
// Follows the same pattern as #TaskRegistry from tasks/_registry.cue
package pipelines

import "helios.io/cue/definitions/tekton"

// =====================================================
// PIPELINE REGISTRY
// Central registry for all pipeline definitions
// =====================================================

// #PipelineDefinition: Schema for a registered pipeline
#PipelineDefinition: {
	// Pipeline metadata
	name:         string
	description?: string

	// Pipeline configuration
	config: {
		description?: string
		params: [...] | *[]
		workspaces: [...] | *[]
		tasks: [...]
		finally?: [...]
	}
}

// #PipelineRegistry: Map of pipeline names to their definitions
// Pipelines register themselves here via unification
// Usage: #PipelineRegistry: "my-pipeline": { ... }
#PipelineRegistry: {
	[Name=string]: #PipelineDefinition & {name: Name}
}

// =====================================================
// RENDER HELPER
// Renders a pipeline from the registry
// Interface matches design doc section 5.9
// =====================================================

// #RenderPipeline: Helper to render a pipeline from registry
#RenderPipeline: {
	// Input parameters (matches #RenderTask interface)
	pipelineType: string
	namespace:    string

	// Capture inputs for inner scope
	let _type = pipelineType
	let _ns = namespace

	// Lookup pipeline from registry
	_definition: #PipelineRegistry[_type]

	// Use #TektonPipeline base to generate output
	_pipeline: tekton.#TektonPipeline & {
		parameter: {
			name:      _type
			namespace: _ns
		}
		config: _definition.config
	}

	// Output the rendered pipeline
	output: _pipeline.output
}

// #RenderAllPipelines: Renders all pipelines in registry for a namespace
#RenderAllPipelines: {
	namespace: string

	let _ns = namespace

	pipelines: {
		for name, _ in #PipelineRegistry {
			(name): (#RenderPipeline & {
				pipelineType: name
				namespace:    _ns
			}).output
		}
	}
}
