package tasks

import "helios.io/cue/definitions/tekton"

// Database Migration Task using golang-migrate
// Runs database migrations from the source repository
#DBMigrate: tekton.#TektonTask & {
	parameter: {
		name: "db-migrate"
	}

	_config: tekton.#Defaults

	output: spec: {
		params: [
			{
				name:        "migration-source"
				description: "Path to migrations directory in the cloned repo (e.g., db/migration or db/migrations)"
				type:        "string"
				default:     "db/migrations"
			},
			{
				name:        "db-secret-name"
				description: "Kubernetes Secret name containing database credentials (expects key PGRST_DB_URI)"
				type:        "string"
				default:     "api-db-secret"
			},
		]

		workspaces: [{
			name:        "source"
			description: "Workspace containing cloned repository with migrations"
		}]

		steps: [{
			name:       "migrate"
			image:      "migrate/migrate:v4.17.0"
			workingDir: "$(workspaces.source.path)"
			env: [{
				name: "DATABASE_URL"
				valueFrom: {
					secretKeyRef: {
						name: "$(params.db-secret-name)"
						key:  "PGRST_DB_URI"
					}
				}
			}]
			script: """
				#!/bin/sh
				set -e

				MIGRATIONS_DIR="$(workspaces.source.path)/$(params.migration-source)"

				if [ ! -d "$MIGRATIONS_DIR" ]; then
				  echo "WARNING: No migrations directory found at $MIGRATIONS_DIR"
				  echo "Skipping migrations..."
				  exit 0
				fi

				echo "Running database migrations from $MIGRATIONS_DIR"

				migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up

				echo "SUCCESS: Migrations completed successfully"
				"""
		}]
	}
}
