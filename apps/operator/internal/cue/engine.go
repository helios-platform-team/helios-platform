/*
CUE Engine - Rendering engine cho Helios Operator

Engine này chỉ có MỘT public method: Render(app Application) ([]byte, error)
Toàn bộ orchestration logic nằm trong CUE (builder.cue), không nằm trong Go.
*/
package cue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"sigs.k8s.io/yaml"
)

// Application Model - maps to CUE schema
// Đây là contract giữa Operator và CUE Engine
type Application struct {
	App AppSpec `json:"app"`
}

type AppSpec struct {
	Name        string      `json:"name"`
	Namespace   string      `json:"namespace"`
	Owner       string      `json:"owner,omitempty"`
	Description string      `json:"description,omitempty"`
	Components  []Component `json:"components"`
}

type Component struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Traits     []Trait                `json:"traits,omitempty"`
}

type Trait struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// Engine wraps CUE context and provides rendering capability
type Engine struct {
	ctx     *cue.Context
	cuePath string
}

// NewEngine creates a new CUE engine
// cuePath is the path to the cue/ directory containing definitions and engine
func NewEngine(cuePath string) (*Engine, error) {
	// Verify the path exists
	if _, err := os.Stat(cuePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("CUE path does not exist: %s", cuePath)
	}

	return &Engine{
		ctx:     cuecontext.New(),
		cuePath: cuePath,
	}, nil
}

// Render takes Application Model and returns Kubernetes YAML manifests
// This is the ONLY public method - all orchestration happens in CUE
func (e *Engine) Render(app Application) ([]byte, error) {
	// 1. Load the CUE engine package
	instances := load.Instances([]string{"./engine"}, &load.Config{
		Dir: e.cuePath,
	})

	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found")
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	// 2. Build the CUE value
	val := e.ctx.BuildInstance(inst)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to build CUE instance: %w", val.Err())
	}

	// 3. Encode Application Model to CUE value
	appJSON, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application: %w", err)
	}

	appVal := e.ctx.CompileBytes(appJSON)
	if appVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile application JSON: %w", appVal.Err())
	}

	// 4. Fill input with Application Model
	val = val.FillPath(cue.ParsePath("input"), appVal)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to fill input: %w", val.Err())
	}

	// 5. Extract kubernetesObjects
	k8sObjects := val.LookupPath(cue.ParsePath("kubernetesObjects"))
	if k8sObjects.Err() != nil {
		return nil, fmt.Errorf("failed to lookup kubernetesObjects: %w", k8sObjects.Err())
	}

	// 6. Export to JSON
	jsonBytes, err := k8sObjects.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to export to JSON: %w", err)
	}

	// 7. Convert to YAML
	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	return yamlBytes, nil
}

// RenderToObjects returns the kubernetes objects as a slice (for direct processing)
func (e *Engine) RenderToObjects(app Application) ([]map[string]interface{}, error) {
	// 1. Load the CUE engine package
	instances := load.Instances([]string{"./engine"}, &load.Config{
		Dir: e.cuePath,
	})

	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found")
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	// 2. Build the CUE value
	val := e.ctx.BuildInstance(inst)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to build CUE instance: %w", val.Err())
	}

	// 3. Encode Application Model to CUE value
	appJSON, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal application: %w", err)
	}

	appVal := e.ctx.CompileBytes(appJSON)
	if appVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile application JSON: %w", appVal.Err())
	}

	// 4. Fill input with Application Model
	val = val.FillPath(cue.ParsePath("input"), appVal)
	if val.Err() != nil {
		return nil, fmt.Errorf("failed to fill input: %w", val.Err())
	}

	// 5. Extract kubernetesObjects
	k8sObjects := val.LookupPath(cue.ParsePath("kubernetesObjects"))
	if k8sObjects.Err() != nil {
		return nil, fmt.Errorf("failed to lookup kubernetesObjects: %w", k8sObjects.Err())
	}

	// 6. Decode to slice
	var objects []map[string]interface{}
	if err := k8sObjects.Decode(&objects); err != nil {
		return nil, fmt.Errorf("failed to decode objects: %w", err)
	}

	return objects, nil
}

// GetCuePath returns the path to CUE definitions
func (e *Engine) GetCuePath() string {
	return e.cuePath
}

// DefaultCuePath returns the default CUE path relative to operator
func DefaultCuePath() string {
	// In production, this would be loaded from ConfigMap mount
	// For development, use relative path to cue/ directory
	return filepath.Join("..", "..", "cue")
}
