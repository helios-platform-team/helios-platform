// build-only pipeline definition.
package pipelines

import "helios.io/cue/definitions/tekton"

// =====================================================
// BUILD-ONLY PIPELINE
// Composed from reusable patterns - proves patterns work
// =====================================================

// Simplified params for build-only pipeline
#BuildOnlyParams: [
	// #PipelineParams is local (patterns.cue), but references tekton.#CommonParams inside itself
	tekton.#CommonParams.app.repoUrl,
	tekton.#CommonParams.app.repoRevision,
	tekton.#CommonParams.app.imageRepo,
	tekton.#CommonParams.image.contextSubpath,
	tekton.#CommonParams.image.dockerSecret,
	tekton.#CommonParams.test.command,
	tekton.#CommonParams.test.image,
]

// Only needs source workspace
#BuildOnlyWorkspaces: [
	// #PipelineWorkspaces is local (patterns.cue)
	#PipelineWorkspaces.source,
]

// Define the pipeline configuration using patterns
_buildOnlyConfig: {
	description: "CI pipeline: fetch source, run tests, build image"

	params:     #BuildOnlyParams
	workspaces: #BuildOnlyWorkspaces

	// Compose tasks from EXISTING patterns - proves reusability
	tasks: [
		// Reuse #FetchSourcePattern (Local)
		(#FetchSourcePattern & {}).task,

		// Reuse #RunTestsPattern (Local)
		(#RunTestsPattern & {
			_runAfter: ["fetch-source-code"]
		}).task,

		// Reuse #BuildImagePattern (Local)
		(#BuildImagePattern & {
			_runAfter: ["run-tests"]
		}).task,
	]
}

// Register pipeline in the registry
#PipelineRegistry: "build-only": {
	name:        "build-only"
	description: "CI pipeline that builds without GitOps deployment"
	config:      _buildOnlyConfig
}

// =====================================================
// DIRECT EXPORT
// =====================================================

BuildOnly: (#RenderPipeline & {
	pipelineType: "build-only"
	namespace:    "default"
}).output