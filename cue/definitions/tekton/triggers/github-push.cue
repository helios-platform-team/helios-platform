package triggers

import (
    "helios.io/cue/definitions/tekton"
)

// =====================================================
// GITEA PUSH TRIGGER BUNDLE
// =====================================================

#GiteaPushTriggerBundle: tekton.#TriggerBundle & {
    // Alias the parameter field to bundleParams for global access
    bundleParams=parameter: _

    // 1. TRIGGER BINDING
    _binding: tekton.#TektonTriggerBinding & {
        parameter: {
            name:      "\(bundleParams.appName)-gitea-binding"
            namespace: bundleParams.namespace
        }
        config: params: [
            {name: "git-repo-url", value: "$(body.repository.clone_url)"},
            {name: "git-revision", value: "$(body.after)"},
        ]
    }

    // 2. TRIGGER TEMPLATES
    _template: tekton.#TektonTriggerTemplate & {
        // Capture bundleParams locally
        let _bp = bundleParams

        parameter: {
            name:      "\(_bp.appName)-gitea-template"
            namespace: _bp.namespace
        }
        config: {
            params: [
                {name: "git-repo-url", description: "From Webhook"},
                {name: "git-revision", description: "From Webhook"},
            ]

            // Inline PipelineRun to avoid abstraction issues with labels/uid
            resourcetemplates: [{
                apiVersion: "tekton.dev/v1beta1"
                kind:       "PipelineRun"
                metadata: {
                    name:      "\(_bp.appName)-run-$(uid)"
                    namespace: _bp.namespace
                    labels: {
                        "helios.io/managed-by":       "helios-operator"
                        "app.kubernetes.io/part-of":  "helios-platform"
                        "app.kubernetes.io/instance": _bp.pipelineName
                        "app.kubernetes.io/name":     _bp.appName
                        "janus-idp.io/tekton":        _bp.appName
                    }
                }
                spec: {
                    pipelineRef: name: _bp.pipelineName
                    serviceAccountName: _bp.serviceAccount
                    
                    params: [
                        // Use operator-provided repo URL (already rewritten to in-cluster URL if needed).
                        {name: "app-repo-url", value:       _bp.gitRepo},
                        {name: "app-repo-revision", value:  "$(tt.params.git-revision)"},
                        {name: "image-repo", value:         _bp.imageRepo},
                        {name: "GITOPS_REPO_URL", value:    _bp.gitopsRepo},
                        {name: "MANIFEST_PATH", value:      _bp.gitopsPath},
                        {name: "GITOPS_REPO_BRANCH", value: _bp.gitopsBranch},
                        {name: "GITOPS_SECRET_REF", value:  _bp.gitopsSecret},
                        {name: "GITOPS_AUTHOR_NAME", value: "Helios Bot"},
                        {name: "GITOPS_AUTHOR_EMAIL", value: "helios-bot@helios.local"},
                        {name: "CONTEXT_SUBPATH", value:    _bp.contextSubpath},
                        {name: "replicas", value:           "\(_bp.replicas)"},
                        {name: "port", value:               "\(_bp.port)"},
                        {name: "docker-secret", value:      _bp.dockerSecret},
                        {name: "test-command", value:       _bp.testCommand},
                        {name: "test-image", value:         _bp.testImage},
                        {name: "argocd-namespace", value: _bp.argoCDNamespace},
                        {name: "argocd-app-name", value:  _bp.argoCDAppName},
                    ]

                    workspaces: [
                        {
                            name: "source-workspace"
                            volumeClaimTemplate: {
                                spec: {
                                    accessModes: ["ReadWriteOnce"]
                                    resources: requests: storage: "1Gi"
                                }
                            }
                        },
                        {
                            name: "gitops-workspace"
                            volumeClaimTemplate: {
                                spec: {
                                    accessModes: ["ReadWriteOnce"]
                                    resources: requests: storage: "1Gi"
                                }
                            }
                        },
                    ]
                }
            }]
        }
    }

    _dbMigrateTemplate: tekton.#TektonTriggerTemplate & {
        let _bp = bundleParams

        parameter: {
            name:      "\(_bp.appName)-db-migrate-template"
            namespace: _bp.namespace
        }
        config: {
            params: [
                {name: "git-revision", description: "From Webhook"},
            ]

            resourcetemplates: [{
                apiVersion: "tekton.dev/v1beta1"
                kind:       "PipelineRun"
                metadata: {
                    name:      "\(_bp.appName)-migrate-$(uid)"
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
                    pipelineRef: {name: "db-migrate"}
                    serviceAccountName: _bp.serviceAccount

                    params: [
                        {name: "app-repo-url", value: _bp.gitRepo},
                        {name: "app-repo-revision", value: "$(tt.params.git-revision)"},
                        {name: "db-secret-name", value: "api-db-secret"},
                        {name: "migration-source", value: "db/migration"},
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
    _listener: tekton.#TektonEventListener & {
        parameter: {
            name:      "\(bundleParams.appName)-listener"
            namespace: bundleParams.namespace
        }
        config: {
            triggers: [
                {
                    name: "gitea-push"
                    bindings: [{ref: _binding.parameter.name}]
                    template: {ref: _template.parameter.name}

                    // Use the cluster Git webhook interceptor for push event validation.
                    interceptors: [{
                        ref: {name: "github", kind: "ClusterInterceptor"}
                        params: [
                            {name: "secretRef", value: {
                                secretName: bundleParams.webhookSecret
                                secretKey: "secret"
                            }},
                            {name: "eventTypes", value: ["push"]},
                        ]
                    }]
                },
                {
                    name: "db-migrate-on-migrations"
                    bindings: [{ref: _binding.parameter.name}]
                    template: {ref: _dbMigrateTemplate.parameter.name}

                    interceptors: [
                        {
                            ref: {name: "github", kind: "ClusterInterceptor"}
                            params: [
                                {name: "secretRef", value: {
                                    secretName: bundleParams.webhookSecret
                                    secretKey: "secret"
                                }},
                                {name: "eventTypes", value: ["push"]},
                            ]
                        },
                        {
                            ref: {name: "cel", kind: "ClusterInterceptor"}
                            params: [{
                                name: "filter"
                                value: "has(body.commits) && body.commits.exists(c, (has(c.added) && c.added.exists(f, f.startsWith('db/migration/') || f.startsWith('db/migrations/'))) || (has(c.modified) && c.modified.exists(f, f.startsWith('db/migration/') || f.startsWith('db/migrations/'))))"
                            }]
                        },
                    ]
                },
            ]
        }
    }

    // 4. BUNDLE OUTPUTS
    outputs: [
        _binding.output,
        _template.output,
        _dbMigrateTemplate.output,
        _listener.output,
    ]
}