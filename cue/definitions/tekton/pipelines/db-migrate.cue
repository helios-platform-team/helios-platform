// db-migrate pipeline definition.
// Specialized pipeline for database migrations and PostgREST schema reload.
// Use case: Run migrations on database schema changes without rebuilding images.
package pipelines

// =====================================================
// PIPELINE DEFINITION
// Database Migration Pipeline
// =====================================================

// =====================================================
// PIPELINE DEFINITION
// Database Migration Pipeline
// =====================================================

_dbMigrateConfig: {
	description: "Database migration pipeline: fetch source → build migration image → update GitOps manifest"

	// Use pipeline params from patterns.cue
	params: #PipelineParamsList

	// Use pipeline workspaces from patterns.cue
	workspaces: #PipelineWorkspacesList

	// Compose tasks from patterns
	tasks: [
		// 1. Fetch source code (includes db/migrations/)
		(#FetchSourcePattern & {}).task,

		// 2. Build and push migration image
		(#BuildMigrateImagePattern & {
			_runAfter: ["fetch-source-code"]
		}).task,

		// 3. Update GitOps manifest (update migrateImage property)
		(#UpdateGitOpsPattern & {
			_runAfter:        ["build-migrate-image"]
			_imageSourceTask: "build-migrate-image"
			_imageType:       "migrate"
		}).task,
	]
}

// Register pipeline in the registry
#PipelineRegistry: "db-migrate": {
	name:        "db-migrate"
	description: "Database migration pipeline for PostgREST applications"
	config:      _dbMigrateConfig
}
