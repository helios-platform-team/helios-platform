package triggers

import (
	"helios.io/cue/definitions/tekton"
)

// =====================================================
// DATABASE MIGRATION TRIGGER BUNDLE
// Triggers db-migrate pipeline only on changes to db/migration path
// =====================================================

#DatabaseMigrationTriggerBundle: tekton.#TriggerBundle & {
	// Alias the parameter field to bundleParams for global access
	bundleParams=parameter: _

	// 1. TRIGGER BINDING
	// Extracts git information from webhook payload
	_binding: tekton.#TektonTriggerBinding & {
		parameter: {
			name:      "\(bundleParams.appName)-db-migrate-binding"
			namespace: bundleParams.namespace
		}
		config: params: [
			{name: "git-repo-url", value: "$(body.repository.clone_url)"},
			{name: "git-revision", value: "$(body.after)"},
		]
	}

	// 2. TRIGGER TEMPLATE
	// Creates a PipelineRun for db-migrate pipeline
	_template: tekton.#TektonTriggerTemplate & {
		let _bp = bundleParams

		parameter: {
			name:      "\(_bp.appName)-db-migrate-template"
			namespace: _bp.namespace
		}
		config: {
			params: [
				{name: "git-repo-url", description: "Repository URL from webhook"},
				{name: "git-revision", description: "Git commit SHA from webhook"},
			]

			// PipelineRun for db-migrate pipeline
			resourcetemplates: [{
				apiVersion: "tekton.dev/v1beta1"
				kind:       "PipelineRun"
				metadata: {
					// Truncate appName to at most 32 chars to keep total name ≤63 chars.
					// CUE slice [:n] panics if string is shorter than n, so we guard with a conditional.
					let _namePrefix = [if len(_bp.appName) > 32 {_bp.appName[:32]}, _bp.appName][0]
					name:      "\(_namePrefix)-migrate-$(uid)"
					namespace: _bp.namespace
					labels: {
						"helios.io/managed-by":       "helios-operator"
						"app.kubernetes.io/part-of":  "helios-platform"
						"app.kubernetes.io/instance": "db-migrate"
						"app.kubernetes.io/name":     _bp.appName
						"janus-idp.io/tekton":        _bp.appName
						"tekton.dev/pipeline":        "db-migrate"
					}
				}
				spec: {
					pipelineRef: {
						name: "db-migrate"
					}
					serviceAccountName: _bp.serviceAccount

					params: [
						{name: "app-repo-url", value: "$(tt.params.git-repo-url)"},
						{name: "app-repo-revision", value: "$(tt.params.git-revision)"},
						{name: "db-secret-name", value: _bp.databaseSecretRef},
						{name: "migration-source", value: "db/migrations"},
						{name: "namespace", value: _bp.namespace},
					]

					workspaces: [{
						name: "source"
						volumeClaimTemplate: {
							spec: {
								accessModes: ["ReadWriteOnce"]
								resources: requests: storage: "1Gi"
							}
						}
					}]
				}
			}]
		}
	}

	// 3. EVENT LISTENER
	// Listens for push events and filters by db/migration path using CEL
	_listener: tekton.#TektonEventListener & {
		parameter: {
			name:      "\(bundleParams.appName)-db-migrate-listener"
			namespace: bundleParams.namespace
		}
		config: {
			triggers: [{
				name: "db-migrate-push"
				bindings: [{ref: _binding.parameter.name}]
				template: {ref: _template.parameter.name}

				// CEL interceptor to filter only db/migration** changes
				// This ensures migration pipeline only runs when migrations directory is modified
				interceptors: [
					{
						// GitHub/Gitea webhook interceptor for authentication
						ref: {name: "github", kind: "ClusterInterceptor"}
						params: [
							{name: "secretRef", value: {
								secretName: bundleParams.webhookSecret
								secretKey:  "secret"
							}},
							{name: "eventTypes", value: ["push"]},
						]
					},
					{
						// CEL interceptor to transform URLs for in-cluster access
						// Replaces localhost:3030 with in-cluster Gitea address
						ref: {name: "cel", kind: "ClusterInterceptor"}
						params: [{
							name:  "overlays"
							value: [
								{
									key:        "repository.clone_url"
									expression: "body.repository.clone_url.replace('http://localhost:3030/', 'http://gitea-http.gitea.svc.cluster.local:3000/')"
								},
							]
						}]
					},
					{
						// CEL filter to only trigger if migrations path changed
						// Includes both added + modified files, matches db/migration or db/migrations
						ref: {name: "cel", kind: "ClusterInterceptor"}
						params: [{
							name:  "filter"
							value: "has(body.commits) && body.commits.exists(c, (has(c.added) && c.added.exists(f, f.startsWith('db/migration/') || f.startsWith('db/migrations/'))) || (has(c.modified) && c.modified.exists(f, f.startsWith('db/migration/') || f.startsWith('db/migrations/'))))"
						}]
					},
				]
			}]
		}
	}

	// 4. BUNDLE OUTPUTS
	outputs: [
		_binding.output,
		_template.output,
		_listener.output,
	]
}
