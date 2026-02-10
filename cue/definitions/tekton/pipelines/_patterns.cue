// Reusable pipeline patterns for composing Tekton pipelines.
// These patterns encapsulate common task configurations.
// Uses #CommonParams and #Defaults from _common.cue as single source of truth.
package tekton

// =====================================================
// TASK NAMES: Must match #TaskRegistry entries
// =====================================================
#TaskNames: {
	gitClone:          "git-clone"
	kanikoBuild:       "kaniko-build"
	gitUpdateManifest: "git-update-manifest"
}

// =====================================================
// PIPELINE PARAMS: Reuse #CommonParams from _common.cue
// Single source of truth for parameter definitions
// =====================================================
#PipelineParams: {
	// App source params - from #CommonParams.app
	appRepoUrl:     #CommonParams.app.repoUrl
	appRepoRevision: #CommonParams.app.repoRevision
	imageRepo:      #CommonParams.app.imageRepo

	// GitOps params - from #CommonParams.gitops
	gitopsRepoUrl: #CommonParams.gitops.repoUrl
	manifestPath:  #CommonParams.gitops.manifestPath
	gitopsBranch:  #CommonParams.gitops.branch

	// Image params - from #CommonParams.image
	contextSubpath: #CommonParams.image.contextSubpath
	dockerSecret:   #CommonParams.image.dockerSecret

	// Testing params - from #CommonParams.test
	testCommand: #CommonParams.test.command
	testImage:   #CommonParams.test.image

	// App config params (pipeline-specific, not in #CommonParams)
	replicas: {
		name:    "replicas"
		default: "2"
	}
	port: {
		name:    "port"
		default: "8080"
	}
}

// Convert params definition to list format for pipeline spec
#PipelineParamsList: [
	#PipelineParams.appRepoUrl,
	#PipelineParams.appRepoRevision,
	#PipelineParams.imageRepo,
	#PipelineParams.gitopsRepoUrl,
	#PipelineParams.manifestPath,
	#PipelineParams.gitopsBranch,
	#PipelineParams.contextSubpath,
	#PipelineParams.replicas,
	#PipelineParams.port,
	#PipelineParams.dockerSecret,
	#PipelineParams.testCommand,
	#PipelineParams.testImage,
]

// =====================================================
// WORKSPACES: Reuse #Defaults.workspaces from _common.cue
// =====================================================
#PipelineWorkspaces: {
	source: {
		name: #Defaults.workspaces.source
	}
	gitops: {
		name: #Defaults.workspaces.gitops
	}
}

#PipelineWorkspacesList: [
	#PipelineWorkspaces.source,
	#PipelineWorkspaces.gitops,
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
		taskRef: name: #TaskNames.gitClone
		if len(_runAfter) > 0 {
			runAfter: _runAfter
		}
		workspaces: [{
			name:      "output"
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			{name: #CommonParams.git.url.name, value:      "$(params.\(#PipelineParams.appRepoUrl.name))"},
			{name: #CommonParams.git.revision.name, value: "$(params.\(#PipelineParams.appRepoRevision.name))"},
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
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			{name: #CommonParams.test.command.name, value: "$(params.\(#PipelineParams.testCommand.name))"},
			{name: #CommonParams.test.image.name, value:   "$(params.\(#PipelineParams.testImage.name))"},
		]
		taskSpec: {
			params: [
				{name: #CommonParams.test.command.name, type: "string", default: #CommonParams.test.command.default},
				{name: #CommonParams.test.image.name, type:   "string"},
			]
			workspaces: [{name: "source"}]
			steps: [{
				name:       "run-tests"
				image:      "$(params.\(#CommonParams.test.image.name))"
				workingDir: "$(workspaces.source.path)"
				script: """
					#!/usr/bin/env sh
					set -e

					if [ -z "$(params.\(#CommonParams.test.command.name))" ]; then
					  echo "No test command provided; skipping tests."
					  exit 0
					fi

					echo "Running tests with command: $(params.\(#CommonParams.test.command.name))"
					# Use 'sh -lc' so that compound commands and PATH work
					sh -lc "$(params.\(#CommonParams.test.command.name))"
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
		taskRef: name: #TaskNames.kanikoBuild
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			{name: #CommonParams.image.name.name, value:        "$(params.\(#PipelineParams.imageRepo.name)):$(params.\(#PipelineParams.appRepoRevision.name))"},
			{name: #CommonParams.image.contextSubpath.name, value: "$(params.\(#PipelineParams.contextSubpath.name))"},
			{name: #CommonParams.image.dockerSecret.name, value:  "$(params.\(#PipelineParams.dockerSecret.name))"},
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
		taskRef: name: #TaskNames.gitUpdateManifest
		runAfter: _runAfter
		workspaces: [{
			name:      "gitops-repo"
			workspace: #PipelineWorkspaces.gitops.name
		}]
		params: [
			{name: #CommonParams.gitops.repoUrl.name, value:      "$(params.\(#PipelineParams.gitopsRepoUrl.name))"},
			{name: #CommonParams.gitops.manifestPath.name, value: "$(params.\(#PipelineParams.manifestPath.name))"},
			{name: #CommonParams.gitops.newImageUrl.name, value:  "$(tasks.\(_imageSourceTask).results.IMAGE_URL)"},
			{name: #CommonParams.gitops.branch.name, value:       "$(params.\(#PipelineParams.gitopsBranch.name))"},
			{name: "REPLICAS", value:                             "$(params.\(#PipelineParams.replicas.name))"},
			{name: "PORT", value:                                 "$(params.\(#PipelineParams.port.name))"},
		]
	}
}
