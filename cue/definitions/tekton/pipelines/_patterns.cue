// Reusable pipeline patterns for composing Tekton pipelines.
// These patterns encapsulate common task configurations.
package tekton

// =====================================================
// STANDARD PIPELINE PARAMS
// Common parameters used across CI/CD pipelines
// =====================================================
#StandardPipelineParams: {
	// Source code params
	appRepoUrl: {
		name: "app-repo-url"
	}
	appRepoRevision: {
		name: "app-repo-revision"
	}
	imageRepo: {
		name: "image-repo"
	}

	// GitOps params
	gitopsRepoUrl: {
		name: "gitops-repo-url"
	}
	manifestPath: {
		name: "manifest-path-in-gitops-repo"
	}
	gitopsBranch: {
		name: "gitops-repo-branch"
	}

	// Build context param
	contextSubpath: {
		name:    "context-subpath"
		default: ""
	}

	// App config params
	replicas: {
		name:    "replicas"
		default: "2"
	}
	port: {
		name:    "port"
		default: "8080"
	}

	// Secret params
	dockerSecret: {
		name:    "docker-secret"
		default: "docker-credentials"
	}

	// Testing params
	testCommand: {
		name:    "test-command"
		default: ""
	}
	testImage: {
		name:    "test-image"
		default: "node:20"
	}
}

// Convert params definition to list format for pipeline spec
#StandardPipelineParamsList: [
	#StandardPipelineParams.appRepoUrl,
	#StandardPipelineParams.appRepoRevision,
	#StandardPipelineParams.imageRepo,
	#StandardPipelineParams.gitopsRepoUrl,
	#StandardPipelineParams.manifestPath,
	#StandardPipelineParams.gitopsBranch,
	#StandardPipelineParams.contextSubpath,
	#StandardPipelineParams.replicas,
	#StandardPipelineParams.port,
	#StandardPipelineParams.dockerSecret,
	#StandardPipelineParams.testCommand,
	#StandardPipelineParams.testImage,
]

// =====================================================
// STANDARD WORKSPACES
// Common workspace definitions
// =====================================================
#StandardWorkspaces: {
	source: {
		name: "source-workspace"
	}
	gitops: {
		name: "gitops-workspace"
	}
}

#StandardWorkspacesList: [
	#StandardWorkspaces.source,
	#StandardWorkspaces.gitops,
]

// =====================================================
// PIPELINE TASK PATTERNS
// Reusable task configurations for pipelines
// =====================================================

// #FetchSourcePattern: Clones source code repository
#FetchSourcePattern: {
	// Input: task name and optional runAfter
	_name:     string | *"fetch-source-code"
	_runAfter: [...string] | *[]

	// Output: pipeline task configuration
	task: {
		name: _name
		taskRef: name: "git-clone"
		if len(_runAfter) > 0 {
			runAfter: _runAfter
		}
		workspaces: [{
			name:      "output"
			workspace: #StandardWorkspaces.source.name
		}]
		params: [
			{name: "url", value:      "$(params.\(#StandardPipelineParams.appRepoUrl.name))"},
			{name: "revision", value: "$(params.\(#StandardPipelineParams.appRepoRevision.name))"},
		]
	}
}

// #RunTestsPattern: Runs tests in source workspace (inline taskSpec)
#RunTestsPattern: {
	// Input: task name and runAfter
	_name:     string | *"run-tests"
	_runAfter: [...string]

	// Output: pipeline task with inline taskSpec
	task: {
		name:     _name
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			workspace: #StandardWorkspaces.source.name
		}]
		params: [
			{name: "test-command", value: "$(params.\(#StandardPipelineParams.testCommand.name))"},
			{name: "test-image", value:   "$(params.\(#StandardPipelineParams.testImage.name))"},
		]
		taskSpec: {
			params: [
				{name: "test-command", type: "string", default: ""},
				{name: "test-image", type:   "string"},
			]
			workspaces: [{name: "source"}]
			steps: [{
				name:       "run-tests"
				image:      "$(params.test-image)"
				workingDir: "$(workspaces.source.path)"
				script: """
					#!/usr/bin/env sh
					set -e

					if [ -z "$(params.test-command)" ]; then
					  echo "No test command provided; skipping tests."
					  exit 0
					fi

					echo "Running tests with command: $(params.test-command)"
					# Use 'sh -lc' so that compound commands and PATH work
					sh -lc "$(params.test-command)"
					"""
			}]
		}
	}
}

// #BuildImagePattern: Builds and pushes container image using Kaniko
#BuildImagePattern: {
	// Input: task name and runAfter
	_name:     string | *"build-and-push-image"
	_runAfter: [...string]

	// Output: pipeline task configuration
	task: {
		name: _name
		taskRef: name: "kaniko-build"
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			workspace: #StandardWorkspaces.source.name
		}]
		params: [
			{name: "IMAGE", value:          "$(params.\(#StandardPipelineParams.imageRepo.name)):$(params.\(#StandardPipelineParams.appRepoRevision.name))"},
			{name: "CONTEXT_SUBPATH", value: "$(params.\(#StandardPipelineParams.contextSubpath.name))"},
			{name: "docker-secret", value:  "$(params.\(#StandardPipelineParams.dockerSecret.name))"},
		]
	}
}

// #UpdateGitOpsPattern: Updates GitOps manifest with new image
#UpdateGitOpsPattern: {
	// Input: task name, runAfter, and image source task
	_name:            string | *"update-gitops-manifest"
	_runAfter:        [...string]
	_imageSourceTask: string | *"build-and-push-image"

	// Output: pipeline task configuration
	task: {
		name: _name
		taskRef: name: "git-update-manifest"
		runAfter: _runAfter
		workspaces: [{
			name:      "gitops-repo"
			workspace: #StandardWorkspaces.gitops.name
		}]
		params: [
			{name: "GITOPS_REPO_URL", value:    "$(params.\(#StandardPipelineParams.gitopsRepoUrl.name))"},
			{name: "MANIFEST_PATH", value:      "$(params.\(#StandardPipelineParams.manifestPath.name))"},
			{name: "NEW_IMAGE_URL", value:      "$(tasks.\(_imageSourceTask).results.IMAGE_URL)"},
			{name: "GITOPS_REPO_BRANCH", value: "$(params.\(#StandardPipelineParams.gitopsBranch.name))"},
			{name: "REPLICAS", value:           "$(params.\(#StandardPipelineParams.replicas.name))"},
			{name: "PORT", value:               "$(params.\(#StandardPipelineParams.port.name))"},
		]
	}
}
