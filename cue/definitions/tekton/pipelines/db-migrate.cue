// db-migrate pipeline definition.
// Specialized pipeline for database migrations and PostgREST schema reload.
// Use case: Run migrations on database schema changes without rebuilding images.
package pipelines

// =====================================================
// PIPELINE DEFINITION
// Database Migration Pipeline
// =====================================================

// Database-specific parameters
_dbMigrateParams: [
	{
		name:        "app-repo-url"
		description: "URL of the application source repository"
		type:        "string"
	},
	{
		name:        "app-repo-revision"
		description: "Git revision/branch (default: main)"
		type:        "string"
		default:     "main"
	},
	{
		name:        "database-url"
		description: "Database connection URL (postgres://user:pass@host:port/dbname)"
		type:        "string"
	},
	{
		name:        "migration-source"
		description: "Path to migrations directory in repo (default: db/migrations)"
		type:        "string"
		default:     "db/migrations"
	},
	{
		name:        "namespace"
		description: "Kubernetes namespace where the app is running"
		type:        "string"
		default:     "default"
	},
]

_dbMigrateWorkspaces: [
	{
		name:        "source"
		description: "Workspace for cloning the source repository"
	},
]

_dbMigrateConfig: {
	description: "Database migration pipeline: clone repo → run migrations → reload PostgREST schema"

	params: _dbMigrateParams

	workspaces: _dbMigrateWorkspaces

	tasks: [
		// 1. Clone the source repository
		{
			name:     "clone-repo"
			taskRef:  {name: "git-clone"}
			workspaces: [{
				name:      "output"
				workspace: "source"
			}]
			params: [
				{name: "url", value:      "$(params.app-repo-url)"},
				{name: "revision", value: "$(params.app-repo-revision)"},
			]
		},

		// 2. Run database migrations (after clone)
		{
			name:     "run-migrations"
			taskRef:  {name: "db-migrate"}
			runAfter: ["clone-repo"]
			workspaces: [{
				name:      "source"
				workspace: "source"
			}]
			params: [
				{name: "database-url", value:      "$(params.database-url)"},
				{name: "migration-source", value: "$(params.migration-source)"},
			]
		},

		// 3. Reload PostgREST schema cache (after migrations)
		{
			name:     "reload-postgrest"
			taskRef:  {name: "postgrest-reload"}
			runAfter: ["run-migrations"]
			params: [
				{name: "database-url", value: "$(params.database-url)"},
			]
		},
	]
}

// Register pipeline in the registry
#PipelineRegistry: "db-migrate": {
	name:        "db-migrate"
	description: "Database migration pipeline for PostgREST applications"
	config:      _dbMigrateConfig
}
