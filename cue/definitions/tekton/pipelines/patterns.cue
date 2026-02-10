package pipelines

import "helios.io/cue/definitions/tekton"

// =====================================================
// TASK NAMES
// =====================================================
#TaskNames: {
	gitClone:          "git-clone"
	kanikoBuild:       "kaniko-build"
	gitUpdateManifest: "git-update-manifest"
}

// =====================================================
// PIPELINE PARAMS
// =====================================================
#PipelineParams: {
	// App source params
	appRepoUrl:     tekton.#CommonParams.app.repoUrl
	appRepoRevision: tekton.#CommonParams.app.repoRevision
	imageRepo:      tekton.#CommonParams.app.imageRepo

	// GitOps params
	gitopsRepoUrl: tekton.#CommonParams.gitops.repoUrl
	manifestPath:  tekton.#CommonParams.gitops.manifestPath
	gitopsBranch:  tekton.#CommonParams.gitops.branch

	// Image params
	contextSubpath: tekton.#CommonParams.image.contextSubpath
	dockerSecret:   tekton.#CommonParams.image.dockerSecret

	// Testing params
	testCommand: tekton.#CommonParams.test.command
	testImage:   tekton.#CommonParams.test.image

	// App config params
	replicas: {
		name:    "replicas"
		default: "2"
	}
	port: {
		name:    "port"
		default: "8080"
	}
}

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
// WORKSPACES
// =====================================================
#PipelineWorkspaces: {
	source: {
		name: tekton.#Defaults.workspaces.source
	}
	gitops: {
		name: tekton.#Defaults.workspaces.gitops
	}
}

#PipelineWorkspacesList: [
	#PipelineWorkspaces.source,
	#PipelineWorkspaces.gitops,
]

// =====================================================
// PIPELINE TASK PATTERNS
// =====================================================

// #FetchSourcePattern
#FetchSourcePattern: {
	_name:     string | *"fetch-source-code"
	_runAfter: [...string] | *[]

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
			{name: tekton.#CommonParams.git.url.name, value:      "$(params.\(#PipelineParams.appRepoUrl.name))"},
			{name: tekton.#CommonParams.git.revision.name, value: "$(params.\(#PipelineParams.appRepoRevision.name))"},
		]
	}
}

// #RunTestsPattern
#RunTestsPattern: {
	_name:     string | *"run-tests"
	_runAfter: [...string]

	task: {
		name:     _name
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			{name: tekton.#CommonParams.test.command.name, value: "$(params.\(#PipelineParams.testCommand.name))"},
			{name: tekton.#CommonParams.test.image.name, value:   "$(params.\(#PipelineParams.testImage.name))"},
		]
		taskSpec: {
			params: [
				{name: tekton.#CommonParams.test.command.name, type: "string", default: tekton.#CommonParams.test.command.default},
				{name: tekton.#CommonParams.test.image.name, type:   "string"},
			]
			workspaces: [{name: "source"}]
			steps: [{
				name:       "run-tests"
				image:      "$(params.\(tekton.#CommonParams.test.image.name))"
				workingDir: "$(workspaces.source.path)"
				script: """
					#!/usr/bin/env sh
					set -e
					if [ -z "$(params.\(tekton.#CommonParams.test.command.name))" ]; then
					  echo "No test command provided; skipping tests."
					  exit 0
					fi
					echo "Running tests with command: $(params.\(tekton.#CommonParams.test.command.name))"
					sh -lc "$(params.\(tekton.#CommonParams.test.command.name))"
					"""
			}]
		}
	}
}

// #BuildImagePattern
#BuildImagePattern: {
	_name:     string | *"build-and-push-image"
	_runAfter: [...string]

	task: {
		name: _name
		// FIX TYPO: tetkon -> tekton
		taskRef: name: #TaskNames.kanikoBuild
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			// FIX TYPO: tetkon -> tekton
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			// FIX TYPOS: tetkon -> tekton
			{name: tekton.#CommonParams.image.name.name, value:        "$(params.\(#PipelineParams.imageRepo.name)):$(params.\(#PipelineParams.appRepoRevision.name))"},
			{name: tekton.#CommonParams.image.contextSubpath.name, value: "$(params.\(#PipelineParams.contextSubpath.name))"},
			{name: tekton.#CommonParams.image.dockerSecret.name, value:  "$(params.\(#PipelineParams.dockerSecret.name))"},
		]
	}
}

// #UpdateGitOpsPattern
#UpdateGitOpsPattern: {
	_name:            string | *"update-gitops-manifest"
	_runAfter:        [...string]
	_imageSourceTask: string | *"build-and-push-image"

	task: {
		name: _name
		taskRef: name: #TaskNames.gitUpdateManifest
		runAfter: _runAfter
		workspaces: [{
			name:      "gitops-repo"
			workspace: #PipelineWorkspaces.gitops.name
		}]
		params: [
			{name: tekton.#CommonParams.gitops.repoUrl.name, value:      "$(params.\(#PipelineParams.gitopsRepoUrl.name))"},
			{name: tekton.#CommonParams.gitops.manifestPath.name, value: "$(params.\(#PipelineParams.manifestPath.name))"},
			{name: tekton.#CommonParams.gitops.newImageUrl.name, value:  "$(tasks.\(_imageSourceTask).results.IMAGE_URL)"},
			{name: tekton.#CommonParams.gitops.branch.name, value:       "$(params.\(#PipelineParams.gitopsBranch.name))"},
			{name: "REPLICAS", value:                             "$(params.\(#PipelineParams.replicas.name))"},
			{name: "PORT", value:                                 "$(params.\(#PipelineParams.port.name))"},
		]
	}
}