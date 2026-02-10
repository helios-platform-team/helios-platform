// build-only pipeline definition.
// A simple CI pipeline that only clones and builds (no GitOps).
// Demonstrates pattern reusability.
package tekton

// =====================================================
// BUILD-ONLY PIPELINE
// Composed from reusable patterns - proves patterns work
// =====================================================

// Simplified params for build-only pipeline (references #CommonParams via #PipelineParams)
#BuildOnlyParams: [
	#PipelineParams.appRepoUrl,
	#PipelineParams.appRepoRevision,
	#PipelineParams.imageRepo,
	#PipelineParams.contextSubpath,
	#PipelineParams.dockerSecret,
	#PipelineParams.testCommand,
	#PipelineParams.testImage,
]

// Only needs source workspace (references #Defaults.workspaces via #PipelineWorkspaces)
#BuildOnlyWorkspaces: [
	#PipelineWorkspaces.source,
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
