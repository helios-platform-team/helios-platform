package argocd

import (
	"strings"
	"testing"

	appv1alpha1 "github.com/helios-platform-team/helios-platform/apps/operator/api/v1alpha1"
)

func TestGenerateArgoApplication_IgnoresOperatorInjectedDatabaseEnv(t *testing.T) {
	app := &appv1alpha1.HeliosApp{
		Spec: appv1alpha1.HeliosAppSpec{
			GitOpsRepo: "http://localhost:3030/helios-platform/nestjs-gitops",
		},
	}
	app.Name = "nestjs"
	app.Namespace = "default"

	generated, err := GenerateArgoApplication(app)
	if err != nil {
		t.Fatalf("GenerateArgoApplication() error = %v", err)
	}

	spec, ok := generated.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("generated spec has unexpected type: %T", generated.Object["spec"])
	}

	ignoreDifferences, ok := spec["ignoreDifferences"].([]any)
	if !ok {
		t.Fatalf("ignoreDifferences has unexpected type: %T", spec["ignoreDifferences"])
	}
	if len(ignoreDifferences) != 1 {
		t.Fatalf("ignoreDifferences length = %d, want 1", len(ignoreDifferences))
	}

	rule, ok := ignoreDifferences[0].(map[string]any)
	if !ok {
		t.Fatalf("ignoreDifferences[0] has unexpected type: %T", ignoreDifferences[0])
	}

	expressions, ok := rule["jqPathExpressions"].([]any)
	if !ok {
		t.Fatalf("jqPathExpressions has unexpected type: %T", rule["jqPathExpressions"])
	}
	if len(expressions) != 1 {
		t.Fatalf("jqPathExpressions length = %d, want 1", len(expressions))
	}
	expression, ok := expressions[0].(string)
	if !ok {
		t.Fatalf("jqPathExpressions[0] has unexpected type: %T", expressions[0])
	}
	if !strings.Contains(expression, "DATABASE_URL") {
		t.Fatalf("jqPathExpressions[0] = %q, want DATABASE_URL to be ignored", expression)
	}
}
