package tekton

import (
	"cmp"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
	cueModel "github.com/helios-platform-team/helios-platform/apps/operator/internal/cue"
	"github.com/helios-platform-team/helios-platform/apps/operator/internal/provider"
)

const defaultPipelineName = "from-code-to-cluster"

// MapCRDToTektonInput converts HeliosApp CRD to TektonInput for CUE rendering.
func MapCRDToTektonInput(app *appv1alpha1.HeliosApp) cueModel.TektonInput {
	p := provider.Default

	input := cueModel.TektonInput{
		AppName:         app.Name,
		Namespace:       app.Namespace,
		GitRepo:         p.RewriteURL(app.Spec.GitRepo),
		GitBranch:       app.Spec.GitBranch,
		ImageRepo:       app.Spec.ImageRepo,
		GitOpsRepo:      p.RewriteURL(app.Spec.GitOpsRepo),
		GitOpsPath:      app.Spec.GitOpsPath,
		GitOpsBranch:    app.Spec.GitOpsBranch,
		GitOpsSecretRef: app.Spec.GitOpsSecretRef,
		WebhookDomain:   app.Spec.WebhookDomain,
		WebhookSecret:   app.Spec.WebhookSecret,
		PipelineName:    app.Spec.PipelineName,
		// PipelineType is intentionally derived from PipelineName because
		// the HeliosApp CRD does not have a separate PipelineType field.
		PipelineType:      app.Spec.PipelineName,
		TriggerType:       app.Spec.TriggerType,
		ServiceAccount:    app.Spec.ServiceAccount,
		PVCName:           app.Spec.PVCName,
		ContextSubpath:    app.Spec.ContextSubpath,
		Replicas:          int(app.Spec.Replicas),
		Port:              int(app.Spec.Port),
		TestCommand:       app.Spec.TestCommand,
		TestImage:         app.Spec.TestImage,
		DockerSecret:      "docker-credentials",
		DatabaseSecretRef: app.Spec.DatabaseSecretRef,
		ArgoCDNamespace:   app.Spec.ArgoCDNamespace,
		ArgoCDProject:     app.Spec.ArgoCDProject,
	}

	input.GitBranch = cmp.Or(input.GitBranch, "main")
	input.GitOpsBranch = cmp.Or(input.GitOpsBranch, "main")
	input.GitOpsSecretRef = cmp.Or(input.GitOpsSecretRef, "helios-gitops-bot")
	input.WebhookSecret = cmp.Or(input.WebhookSecret, p.WebhookSecretName())
	// Derive the database secret name dynamically from app name if not explicitly set
	if input.DatabaseSecretRef == "" {
		input.DatabaseSecretRef = app.Name + "-db-secret"
	}
	input.TriggerType = cmp.Or(input.TriggerType, p.DefaultTriggerType())
	if input.PipelineName == "" {
		input.PipelineName = defaultPipelineName
		input.PipelineType = defaultPipelineName
	}
	input.ServiceAccount = cmp.Or(input.ServiceAccount, "default")
	if input.Replicas <= 0 {
		input.Replicas = 1
	}
	if input.Port <= 0 || input.Port > 65535 {
		input.Port = 8080
	}
	input.ArgoCDNamespace = cmp.Or(input.ArgoCDNamespace, "argocd")
	input.ArgoCDProject = cmp.Or(input.ArgoCDProject, "default")

	return input
}
