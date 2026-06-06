package pipelines

import "helios.io/cue/definitions/tekton"

// =====================================================
// TASK NAMES
// =====================================================
#TaskNames: {
	gitClone:          "git-clone"
	buildahBuild:      "buildah-build"
	gitUpdateManifest: "git-update-manifest"
	argocdSync:        "argocd-sync"
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
	gitopsSecretRef: tekton.#CommonParams.gitops.secretRef
	gitopsAuthorName: tekton.#CommonParams.gitops.authorName
	gitopsAuthorEmail: tekton.#CommonParams.gitops.authorEmail

	// Image params
	contextSubpath: tekton.#CommonParams.image.contextSubpath
	dockerSecret:   tekton.#CommonParams.image.dockerSecret
	storageDriver:  tekton.#CommonParams.image.storageDriver
	buildahIsolation: tekton.#CommonParams.image.buildahIsolation
	buildPlatforms: tekton.#CommonParams.image.buildPlatforms

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

	// Argo CD sync (after GitOps push) — kubectl patch Application
	argocdNamespace: tekton.#CommonParams.argocd.namespace
	argocdAppName:   tekton.#CommonParams.argocd.appName
}

#PipelineParamsList: [
	#PipelineParams.appRepoUrl,
	#PipelineParams.appRepoRevision,
	#PipelineParams.imageRepo,
	#PipelineParams.gitopsRepoUrl,
	#PipelineParams.manifestPath,
	#PipelineParams.gitopsBranch,
	#PipelineParams.gitopsSecretRef,
	#PipelineParams.gitopsAuthorName,
	#PipelineParams.gitopsAuthorEmail,
	#PipelineParams.contextSubpath,
	#PipelineParams.replicas,
	#PipelineParams.port,
	#PipelineParams.dockerSecret,
	#PipelineParams.storageDriver,
	#PipelineParams.buildahIsolation,
	#PipelineParams.buildPlatforms,
	#PipelineParams.testCommand,
	#PipelineParams.testImage,
	#PipelineParams.argocdNamespace,
	#PipelineParams.argocdAppName,
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

// #RunTestsPattern — installs dependencies and runs tests.
#RunTestsPattern: {
	_name:     string | *"run-tests"
	_runAfter: [...string]

	task: {
		name:     _name
		runAfter: _runAfter
		workspaces: [
			{
				name:      "source"
				workspace: #PipelineWorkspaces.source.name
			},
		]
		params: [
			{name: tekton.#CommonParams.test.command.name, value: "$(params.\(#PipelineParams.testCommand.name))"},
			{name: tekton.#CommonParams.test.image.name, value:   "$(params.\(#PipelineParams.testImage.name))"},
		]
		taskSpec: {
			params: [
				{name: tekton.#CommonParams.test.command.name, type: "string", default: tekton.#CommonParams.test.command.default},
				{name: tekton.#CommonParams.test.image.name, type:   "string"},
			]
			workspaces: [
				{name: "source"},
			]
			steps: [
				{
					// Step 1: Install dependencies.
					name:       "install-dependencies"
					image:      "$(params.\(tekton.#CommonParams.test.image.name))"
					workingDir: "$(workspaces.source.path)"
					script: """
						#!/usr/bin/env sh
						set -e

						# Only run install logic if there is a test command.
						if [ -z "$(params.\(tekton.#CommonParams.test.command.name))" ]; then
						  echo "No test command - skipping install."
						  exit 0
						fi

						# Check for package-lock.json or package.json
						if [ ! -f package-lock.json ] && [ ! -f package.json ]; then
						  echo "No package.json or package-lock.json found - skipping npm install."
						  exit 0
						fi

						# Check if npm is available
						if ! command -v npm >/dev/null 2>&1; then
						  echo "npm not found - skipping npm install."
						  exit 0
						fi

						echo "Running npm ci directly."
						npm ci --ignore-scripts --no-audit --no-fund --progress=false --fetch-retries=5 --fetch-retry-mintimeout=15000
						if [ -f prisma/schema.prisma ]; then
						  npm run prisma:generate
						fi
						"""
				},
				{
					// Step 2: Run the user-supplied test command.
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
						echo "Running tests: $(params.\(tekton.#CommonParams.test.command.name))"
						sh -lc "$(params.\(tekton.#CommonParams.test.command.name))"
						"""
				},
			]
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
		taskRef: name: #TaskNames.buildahBuild
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
			{name: tekton.#CommonParams.image.storageDriver.name, value: "$(params.\(#PipelineParams.storageDriver.name))"},
			{name: tekton.#CommonParams.image.buildahIsolation.name, value: "$(params.\(#PipelineParams.buildahIsolation.name))"},
			{name: tekton.#CommonParams.image.buildPlatforms.name, value: "$(params.\(#PipelineParams.buildPlatforms.name))"},
		]
	}
}

// #BuildMigrateImagePattern - Build Docker image with database migration tool and scripts
// Tags image as <registry>/<app-name>-migrate:latest for use by PreSync Jobs
#BuildMigrateImagePattern: {
	_name:     string | *"build-migrate-image"
	_runAfter: [...string]

	task: {
		name: _name
		taskRef: name: #TaskNames.kanikoBuild
		runAfter: _runAfter
		workspaces: [{
			name:      "source"
			workspace: #PipelineWorkspaces.source.name
		}]
		params: [
			// Build migration image with :latest tag (will be pulled by PreSync Job)
			{name: tekton.#CommonParams.image.name.name, value:        "$(params.\(#PipelineParams.imageRepo.name))-migrate:latest"},
			{name: tekton.#CommonParams.image.contextSubpath.name, value: "."},
			{name: tekton.#CommonParams.image.dockerSecret.name, value:  "$(params.\(#PipelineParams.dockerSecret.name))"},
			// Override Dockerfile to use Dockerfile.migrate
			{name: "DOCKERFILE", value: "./Dockerfile.migrate"},
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
			{name: tekton.#CommonParams.gitops.secretRef.name, value:    "$(params.\(#PipelineParams.gitopsSecretRef.name))"},
			{name: tekton.#CommonParams.gitops.authorName.name, value:   "$(params.\(#PipelineParams.gitopsAuthorName.name))"},
			{name: tekton.#CommonParams.gitops.authorEmail.name, value:  "$(params.\(#PipelineParams.gitopsAuthorEmail.name))"},
			{name: "REPLICAS", value:                             "$(params.\(#PipelineParams.replicas.name))"},
			{name: "PORT", value:                                 "$(params.\(#PipelineParams.port.name))"},
		]
	}
}

// #ArgoCDSyncPattern — kubectl patch Application.operation (see Argo CD sync-kubectl docs)
#ArgoCDSyncPattern: {
	_name:     string | *"argocd-sync"
	_runAfter: [...string]

	task: {
		name:     _name
		taskRef: name: #TaskNames.argocdSync
		runAfter: _runAfter
		params: [
			{name: tekton.#CommonParams.argocd.namespace.name, value: "$(params.\(#PipelineParams.argocdNamespace.name))"},
			{name: tekton.#CommonParams.argocd.appName.name, value:  "$(params.\(#PipelineParams.argocdAppName.name))"},
		]
	}
}