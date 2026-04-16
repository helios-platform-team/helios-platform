package tasks

import "helios.io/cue/definitions/tekton"

// PostgREST Schema Reload Task
// Triggers PostgREST to reload schema cache via NOTIFY command
// This ensures the API immediately reflects database changes
#PostgRESTReload: tekton.#TektonTask & {
	parameter: {
		name: "postgrest-reload"
	}

	_config: tekton.#Defaults

	output: spec: {
		params: [{
			name:        "db-secret-name"
			description: "Kubernetes Secret name containing database credentials (expects key PGRST_DB_URI)"
			type:        "string"
			default:     "api-db-secret"
		}]

		steps: [{
			name:  "reload-schema"
			image: "postgres:15-alpine"
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

				echo "Triggering PostgREST schema reload..."

				psql "$DATABASE_URL" -c "NOTIFY pgrst, 'reload schema';"

				echo "SUCCESS: Schema reload triggered successfully"
				"""
		}]
	}
}
