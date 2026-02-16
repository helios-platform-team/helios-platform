/*
TektonRenderer - Go SDK for CUE Tekton Builder

This renderer bridges the Go Operator with the CUE Tekton Builder.
It takes a TektonInput (mapped from HeliosApp CRD) and produces
Kubernetes unstructured objects via CUE evaluation.

Flow: HeliosApp CR → TektonInput (Go) → CUE Engine → []Unstructured
*/
package cue

import (
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TektonInput maps 1:1 with CUE #TektonInput schema.
// This is the contract between Go Operator and CUE Tekton Builder.
type TektonInput struct {
	// === IDENTITY ===
	AppName   string `json:"appName"`
	Namespace string `json:"namespace"`

	// === SOURCE CODE ===
	GitRepo   string `json:"gitRepo"`
	GitBranch string `json:"gitBranch,omitempty"`

	// === CONTAINER IMAGE ===
	ImageRepo string `json:"imageRepo"`

	// === GITOPS ===
	GitOpsRepo      string `json:"gitopsRepo"`
	GitOpsPath      string `json:"gitopsPath"`
	GitOpsBranch    string `json:"gitopsBranch,omitempty"`
	GitOpsSecretRef string `json:"gitopsSecretRef,omitempty"`

	// === WEBHOOK (optional) ===
	WebhookDomain string `json:"webhookDomain,omitempty"`
	WebhookSecret string `json:"webhookSecret,omitempty"`

	// === PIPELINE CONFIG ===
	PipelineName   string `json:"pipelineName,omitempty"`
	PipelineType   string `json:"pipelineType,omitempty"`
	TriggerType    string `json:"triggerType,omitempty"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	PVCName        string `json:"pvcName,omitempty"`
	ContextSubpath string `json:"contextSubpath,omitempty"`

	// === APP CONFIG ===
	Replicas int `json:"replicas,omitempty"`
	Port     int `json:"port,omitempty"`

	// === TESTING ===
	TestCommand string `json:"testCommand,omitempty"`

	// === SECRETS ===
	DockerSecret string `json:"dockerSecret,omitempty"`

	// === ARGOCD ===
	ArgoCDNamespace string `json:"argoCDNamespace,omitempty"`
	ArgoCDProject   string `json:"argoCDProject,omitempty"`
}

// TektonRendererInterface defines methods for Tekton CUE rendering.
// Used by the controller; enables mocking in tests.
type TektonRendererInterface interface {
	RenderTektonResources(input TektonInput) ([]*unstructured.Unstructured, error)
}

// TektonRenderer wraps the CUE context and provides Tekton resource rendering.
type TektonRenderer struct {
	ctx     *cue.Context
	cuePath string
}

// NewTektonRenderer creates a new TektonRenderer.
// cuePath is the path to the cue/ directory containing definitions and engine.
func NewTektonRenderer(cuePath string) (*TektonRenderer, error) {
	return &TektonRenderer{
		ctx:     cuecontext.New(),
		cuePath: cuePath,
	}, nil
}

// RenderTektonResources takes a TektonInput and returns Kubernetes unstructured objects.
// This is the ONLY public rendering method — all orchestration happens in CUE.
func (r *TektonRenderer) RenderTektonResources(input TektonInput) ([]*unstructured.Unstructured, error) {
	// 1. Load the CUE engine package (which imports tekton builder)
	instances := load.Instances([]string{"./engine"}, &load.Config{
		Dir: r.cuePath,
	})

	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s/engine", r.cuePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("failed to load CUE tekton instance: %w", inst.Err)
	}

	// 2. Build the CUE value
	val := r.ctx.BuildInstance(inst)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to build CUE tekton instance: %w", val.Err())
	}

	// 3. Encode TektonInput to JSON, then to CUE value
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TektonInput: %w", err)
	}

	inputVal := r.ctx.CompileBytes(inputJSON)
	if inputVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile TektonInput JSON: %w", inputVal.Err())
	}

	// 4. Fill tektonInput in the CUE value
	val = val.FillPath(cue.ParsePath("tektonInput"), inputVal)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to fill tektonInput: %w", val.Err())
	}

	// 5. Extract tektonObjects
	tektonObjects := val.LookupPath(cue.ParsePath("tektonObjects"))
	if tektonObjects.Err() != nil {
		return nil, fmt.Errorf("failed to lookup tektonObjects: %w", tektonObjects.Err())
	}

	// 6. Decode to slice of generic maps
	var rawObjects []map[string]interface{}
	if err := tektonObjects.Decode(&rawObjects); err != nil {
		return nil, fmt.Errorf("failed to decode tektonObjects: %w", err)
	}

	// 7. Convert to Kubernetes unstructured objects
	results := make([]*unstructured.Unstructured, 0, len(rawObjects))
	for i, raw := range rawObjects {
		obj := &unstructured.Unstructured{Object: raw}

		// Validate required fields
		if obj.GetKind() == "" {
			return nil, fmt.Errorf("tektonObjects[%d] has no 'kind' field", i)
		}
		if obj.GetAPIVersion() == "" {
			return nil, fmt.Errorf("tektonObjects[%d] has no 'apiVersion' field", i)
		}

		// Set GVK from apiVersion/kind
		gv, err := schema.ParseGroupVersion(obj.GetAPIVersion())
		if err != nil {
			return nil, fmt.Errorf("tektonObjects[%d] invalid apiVersion %q: %w", i, obj.GetAPIVersion(), err)
		}
		obj.SetGroupVersionKind(gv.WithKind(obj.GetKind()))

		results = append(results, obj)
	}

	return results, nil
}
