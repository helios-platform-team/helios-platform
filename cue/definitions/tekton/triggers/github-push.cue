package triggers

import (
    "helios.io/cue/definitions/tekton"
)

// =====================================================
// GITHUB PUSH TRIGGER BUNDLE
// =====================================================

#GitHubPushTriggerBundle: tekton.#TriggerBundle & {
    // Alias the parameter field to bundleParams for global access
    bundleParams=parameter: _

    // 1. TRIGGER BINDING
    _binding: tekton.#TektonTriggerBinding & {
        parameter: {
            name:      "\(bundleParams.appName)-github-binding"
            namespace: bundleParams.namespace
        }
        config: params: [
            {name: "git-repo-url", value: "$(body.repository.clone_url)"},
            {name: "git-revision", value: "$(body.head_commit.id)"},
        ]
    }

    // 2. TRIGGER TEMPLATE
    _template: tekton.#TektonTriggerTemplate & {
        // Capture bundleParams locally
        let _bp = bundleParams

        parameter: {
            name:      "\(_bp.appName)-github-template"
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
                        {name: "app-repo-url", value:       "$(tt.params.git-repo-url)"},
                        {name: "app-repo-revision", value:  "$(tt.params.git-revision)"},
                        {name: "image-repo", value:         _bp.imageRepo},
                        {name: "GITOPS_REPO_URL", value:    _bp.gitopsRepo},
                        {name: "MANIFEST_PATH", value:      _bp.gitopsPath},
                        {name: "GITOPS_REPO_BRANCH", value: _bp.gitopsBranch},
                        {name: "CONTEXT_SUBPATH", value:    _bp.contextSubpath},
                        {name: "replicas", value:           "\(_bp.replicas)"},
                        {name: "port", value:               "\(_bp.port)"},
                        {name: "docker-secret", value:      _bp.dockerSecret},
                        {name: "test-command", value:       _bp.testCommand},
                        {name: "test-image", value:         _bp.testImage},
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

    // 3. EVENT LISTENER
    _listener: tekton.#TektonEventListener & {
        parameter: {
            name:      "\(bundleParams.appName)-listener"
            namespace: bundleParams.namespace
        }
        config: {
            triggers: [{
                name: "github-push"
                bindings: [{ref: _binding.parameter.name}]
                template: {ref: _template.parameter.name}
                
                // Add GitHub Interceptor for security (Validates webhook secret)
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