package tasks

import "helios.io/cue/definitions/tekton"

// Database Migration Task using golang-migrate
// Runs database migrations from the source repository
// Expects DATABASE_URL to be injected from Kubernetes Secret
#DBMigrate: tekton.#TektonTask & {
parameter: {
name: "db-migrate"
}

_config: tekton.#Defaults

output: spec: {
params: [
{
name:        "migration-source"
description: "Path to migrations directory in the cloned repo (e.g., db/migrations)"
type:        "string"
default:     "db/migrations"
},
{
name:        "database-url"
description: "Database connection URL (postgres://user:pass@host:port/dbname)"
type:        "string"
},
]

workspaces: [{
name:        "source"
description: "Workspace containing cloned repository with migrations"
}]

volumes: [{
name: "db-credentials"
secret: {
secretName: "database-secret"
}
}]

steps: [
{
name:  "migrate"
image: "migrate/migrate:v4.17.0"
workingDir: "$(workspaces.source.path)"
env: [
{
name: "DATABASE_URL"
valueFrom: {
secretKeyRef: {
name: "database-secret"
key:  "DATABASE_URL"
}
}
},
]
volumeMounts: [{
name:      "db-credentials"
mountPath: "/etc/db-credentials"
readOnly:  true
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

migrate \\
-path "$MIGRATIONS_DIR" \\
-database "$DATABASE_URL" \\
up

if [ $? -eq 0 ]; then
echo "SUCCESS: Migrations completed successfully"
else
echo "ERROR: Migrations failed"
exit 1
fi
"""
},
]
}
}
