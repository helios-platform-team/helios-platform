// from-code-to-cluster pipeline definition.
package pipelines

// =====================================================
// PIPELINE DEFINITION
// Composed from reusable patterns
// =====================================================

// Define the pipeline configuration using patterns
_fromCodeToClusterConfig: {
	description: "Complete CI/CD pipeline: fetch source, run tests, build image, update GitOps"

	// Use pipeline params from patterns.cue
	params: #PipelineParamsList

	// Use pipeline workspaces from patterns.cue
	workspaces: #PipelineWorkspacesList

	// Compose tasks from patterns
	tasks: [
		// 1. Fetch source code
		(#FetchSourcePattern & {}).task,

		// 2. Run tests (after fetch)
		(#RunTestsPattern & {
			_runAfter: ["fetch-source-code"]
		}).task,

		// 3. Build and push image (after tests)
		(#BuildImagePattern & {
			_runAfter: ["run-tests"]
		}).task,

		// 4. Update GitOps manifest (after build)
		(#UpdateGitOpsPattern & {
			_runAfter:        ["build-and-push-image"]
			_imageSourceTask: "build-and-push-image"
		}).task,
	]
}

// Register pipeline in the registry
#PipelineRegistry: "from-code-to-cluster": {
	name:        "from-code-to-cluster"
	description: "Complete CI/CD pipeline from source to deployment"
	config:      _fromCodeToClusterConfig
}

// =====================================================
// DIRECT EXPORT
// =====================================================

// Convenience: render pipeline for default namespace
FromCodeToCluster: (#RenderPipeline & {
	pipelineType: "from-code-to-cluster"
	namespace:    "default"
}).output