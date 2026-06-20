package engine

import (
	"list"
	"helios.io/cue/definitions/tekton"
	"helios.io/cue/definitions/tekton/tasks"
	"helios.io/cue/definitions/tekton/pipelines"
	"helios.io/cue/definitions/tekton/triggers"
)

// =====================================================
// TEKTON BUILDER
// Generates all CI/CD resources based on Input
// =====================================================

// Input from Operator/User
tektonInput: tekton.#TektonInput

// =====================================================
// VALUE RESOLUTION (Top Level)
// =====================================================

// Resolve Webhook Domain safely.
// Logic: If input is provided, use it. Else, default to empty string "".
let _webhookDomain = [
	if tektonInput.webhookDomain != _|_ {tektonInput.webhookDomain},
	"",
][0]

// Resolve Test Command safely.
let _testCommand = [
	if tektonInput.testCommand != _|_ {tektonInput.testCommand},
	"",
][0]

// Resolve Test Image safely.
let _testImage = [
	if tektonInput.testImage != _|_ {tektonInput.testImage},
	tekton.#CommonParams.test.image.default,
][0]

// Argo CD API URL: explicit override or in-cluster default.
let _argoCDNamespace = [
	if tektonInput.argoCDNamespace != _|_ {tektonInput.argoCDNamespace},
	"argocd",
][0]

// =====================================================
// RENDERING
// =====================================================

// 1. RENDER TASKS
_tasks: [
	for name, _ in tasks.#TaskRegistry {
		(tasks.#RenderTask & {
			taskName:  name
			namespace: tektonInput.namespace
		}).output
	},
]

// 2. RENDER PIPELINES
// Render primary pipeline always
_primaryPipeline: [
	(pipelines.#RenderPipeline & {
		pipelineType: tektonInput.pipelineType
		namespace:    tektonInput.namespace
	}).output,
]

// Also render db-migrate pipeline if it's available and needed by triggers
_dbMigratePipeline: [
	if tektonInput.triggerType == "git-push" || tektonInput.triggerType == "gitea-push" || tektonInput.triggerType == "db-migrate" {
		(pipelines.#RenderPipeline & {
			pipelineType: "db-migrate"
			namespace:    tektonInput.namespace
		}).output
	},
]

_pipeline: list.Concat([_primaryPipeline, _dbMigratePipeline])

// 3. RENDER TRIGGERS
_triggers: (triggers.#RenderTriggers & {
	triggerType: tektonInput.triggerType

	// Use the pre-calculated concrete string
	webhookDomain: _webhookDomain

	bundleParams: {
		appName:      tektonInput.appName
		namespace:    tektonInput.namespace
		pipelineName: tektonInput.pipelineName

		// Use the pre-calculated concrete string
		webhookDomain: _webhookDomain

		webhookSecret:  tektonInput.webhookSecret
		gitRepo:        tektonInput.gitRepo
		gitBranch:      tektonInput.gitBranch
		imageRepo:      tektonInput.imageRepo
		gitopsRepo:     tektonInput.gitopsRepo
		gitopsPath:     tektonInput.gitopsPath
		gitopsBranch:   tektonInput.gitopsBranch
		gitopsSecret:   tektonInput.gitopsSecretRef
		pvcName:        tektonInput.pvcName
		contextSubpath: tektonInput.contextSubpath
		replicas:       tektonInput.replicas
		port:           tektonInput.port

		// Use the pre-calculated concrete string
		testCommand: _testCommand

		// Use explicit test image when provided, else common default.
		testImage:         _testImage
		serviceAccount:    tektonInput.serviceAccount
		dockerSecret:      tektonInput.dockerSecret
		databaseSecretRef: tektonInput.databaseSecretRef

		storageDriver:    tektonInput.storageDriver
		buildahIsolation: tektonInput.buildahIsolation
		buildPlatforms:   tektonInput.buildPlatforms

		argoCDNamespace: _argoCDNamespace
		argoCDAppName:   "\(tektonInput.appName)-argocd"
	}
}).outputs

// 4. FINAL OUTPUT
tektonObjects: list.Concat([_tasks, _pipeline, _triggers])
