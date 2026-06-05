// db-migrate-image pipeline definition.
// Builds a Docker image containing golang-migrate and SQL migration scripts.
// This image is used by ArgoCD PreSync hooks to run database migrations before deploying PostgREST.
package pipelines

import "helios.io/cue/definitions/tekton"

// =====================================================
// PIPELINE DEFINITION
// Simple 2-task pipeline: fetch source code, then build migration image
// =====================================================

// Simplified params for db-migrate-image pipeline
#DbMigrateImageParams: [
	// App source and image params
	tekton.#CommonParams.app.repoUrl,
	tekton.#CommonParams.app.repoRevision,
	tekton.#CommonParams.app.imageRepo,
	tekton.#CommonParams.image.contextSubpath,
	tekton.#CommonParams.image.dockerSecret,
]

// Only needs source workspace
#DbMigrateImageWorkspaces: [
	// #PipelineWorkspaces is local (patterns.cue)
	#PipelineWorkspaces.source,
]

// Define the pipeline configuration
_dbMigrateImageConfig: {
	description: "Build database migration Docker image with golang-migrate tool and migration scripts"

	// Use simplified params for migration image build
	params: #DbMigrateImageParams

	// Use source workspace
	workspaces: #DbMigrateImageWorkspaces

	// Compose tasks from patterns
	tasks: [
		// 1. Fetch source code (includes db/migrations/)
		(#FetchSourcePattern & {}).task,

		// 2. Build and push migration image (tagged as :latest)
		(#BuildMigrateImagePattern & {
			_runAfter: ["fetch-source-code"]
		}).task,
	]
}

// Register pipeline in the registry
#PipelineRegistry: "db-migrate-image": {
	name:        "db-migrate-image"
	description: "Build migration image with database migration tool and scripts"
	config:      _dbMigrateImageConfig
}

// =====================================================
// DIRECT EXPORT
// =====================================================

// Convenience: render pipeline for default namespace
DbMigrateImage: (#RenderPipeline & {
	pipelineType: "db-migrate-image"
	namespace:    "default"
}).output
