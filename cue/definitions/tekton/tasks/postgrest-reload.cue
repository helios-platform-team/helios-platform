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
params: [
{
name:        "database-url"
description: "Database connection URL (postgres://user:pass@host:port/dbname)"
type:        "string"
},
]

volumes: [{
name: "db-credentials"
secret: {
secretName: "database-secret"
}
}]

steps: [
{
name:  "reload-schema"
image: "postgres:15-alpine"
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

echo "Triggering PostgREST schema reload..."

psql "$DATABASE_URL" -c "NOTIFY pgrst, 'reload schema';"

if [ $? -eq 0 ]; then
echo "SUCCESS: Schema reload triggered successfully"
echo "API will reflect new schema within seconds"
else
echo "ERROR: Failed to trigger schema reload"
exit 1
fi
"""
},
]
}
}
