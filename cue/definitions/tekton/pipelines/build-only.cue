// build-only pipeline definition.
// A simple CI pipeline that only clones and builds (no GitOps).
// Demonstrates pattern reusability.
package tekton

// =====================================================
// BUILD-ONLY PIPELINE
// Composed from reusable patterns - proves patterns work
// =====================================================

// Simplified params for build-only pipeline
#BuildOnlyParams: [
	#StandardPipelineParams.appRepoUrl,
	#StandardPipelineParams.appRepoRevision,
	#StandardPipelineParams.imageRepo,
	#StandardPipelineParams.contextSubpath,
	#StandardPipelineParams.dockerSecret,
	#StandardPipelineParams.testCommand,
	#StandardPipelineParams.testImage,
]

// Only needs source workspace
#BuildOnlyWorkspaces: [
	#StandardWorkspaces.source,
]

// Define the pipeline configuration using patterns
_buildOnlyConfig: {
	description: "CI pipeline: fetch source, run tests, build image"

	params:     #BuildOnlyParams
	workspaces: #BuildOnlyWorkspaces

	// Compose tasks from EXISTING patterns - proves reusability
	tasks: [
		// Reuse #FetchSourcePattern
		(#FetchSourcePattern & {}).task,

		// Reuse #RunTestsPattern
		(#RunTestsPattern & {
			_runAfter: ["fetch-source-code"]
		}).task,

		// Reuse #BuildImagePattern
		(#BuildImagePattern & {
			_runAfter: ["run-tests"]
		}).task,
	]
}

// Register pipeline in the registry
Registry: "build-only": {
	name:        "build-only"
	description: "CI pipeline that builds without GitOps deployment"
	config:      _buildOnlyConfig
}

// =====================================================
// DIRECT EXPORT
// =====================================================

BuildOnly: (#RenderPipeline & {
	_pipelineName: "build-only"
	_namespace:    "default"
}).output
