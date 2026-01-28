/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// HeliosAppSpec defines the desired state of HeliosApp
// Follows OAM (Open Application Model) with components and traits
type HeliosAppSpec struct {
	// Owner team of the application
	// +optional
	Owner string `json:"owner,omitempty"`

	// WebhookDomain is the external domain (e.g., ngrok) for Git webhooks
	// +optional
	WebhookDomain string `json:"webhookDomain,omitempty"`

	// Description of the application
	// +optional
	Description string `json:"description,omitempty"`

	// ImageRepo is the repository where the container image will be pushed
	// +kubebuilder:validation:Required
	ImageRepo string `json:"imageRepo"`

	// GitRepo is the URL of the source code repository
	// +kubebuilder:validation:Required
	GitRepo string `json:"gitRepo"`

	// GitBranch is the branch of the source code repository
	// +optional
	GitBranch string `json:"gitBranch,omitempty"`

	// GitOpsRepo is the URL of the GitOps repository
	// +kubebuilder:validation:Required
	GitOpsRepo string `json:"gitopsRepo"`

	// GitOpsPath is the path within the GitOps repository
	// +kubebuilder:validation:Required
	GitOpsPath string `json:"gitopsPath"`

	// PipelineName is the name of the Tekton Pipeline to run (default: from-code-to-cluster)
	// +optional
	// +kubebuilder:default="from-code-to-cluster"
	PipelineName string `json:"pipelineName,omitempty"`

	// WebhookSecret is the name of the secret containing the GitHub webhook secret token
	// +optional
	// +kubebuilder:default="github-webhook-secret"
	WebhookSecret string `json:"webhookSecret,omitempty"`

	// GitOpsBranch is the branch of the GitOps repository
	// +optional
	// +kubebuilder:default="main"
	GitOpsBranch string `json:"gitopsBranch,omitempty"`

	// GitOpsSecretRef is the name of the secret containing git credentials (token)
	// +optional
	GitOpsSecretRef string `json:"gitopsSecretRef,omitempty"`

	// ArgoCDNamespace is the namespace where ArgoCD is running
	// +optional
	// +kubebuilder:default="argocd"
	ArgoCDNamespace string `json:"argoCDNamespace,omitempty"`

	// ArgoCDProject is the ArgoCD project to use
	// +optional
	// +kubebuilder:default="default"
	ArgoCDProject string `json:"argoCDProject,omitempty"`

	// Replicas is the number of pods to run
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Port is the container port
	// +kubebuilder:validation:Minimum=1
	// +optional
	Port int32 `json:"port,omitempty"`

	// Env variables for the application
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources requirements for the container
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccount to run the application (and pipeline)
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// PVCName is the name of the PVC for the pipeline workspace
	// +optional
	PVCName string `json:"pvcName,omitempty"`

	// ContextSubpath is the path where the Dockerfile is located
	// +optional
	ContextSubpath string `json:"contextSubpath,omitempty"`

	// Components define the workloads of the application
	Components []Component `json:"components"`
}

// Component represents an OAM component (WHAT to run)
type Component struct {
	// Name of the component
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the component (e.g., "web-service")
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Properties specific to the component type
	// +kubebuilder:validation:Required
	// +kubebuilder:pruning:PreserveUnknownFields
	Properties *runtime.RawExtension `json:"properties"`

	// Traits define operational behaviors (HOW to run)
	// +optional
	Traits []Trait `json:"traits,omitempty"`
}

// Trait represents an OAM trait (HOW to run)
type Trait struct {
	// Type of the trait (e.g., "service", "ingress")
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Properties specific to the trait type
	// +kubebuilder:validation:Required
	// +kubebuilder:pruning:PreserveUnknownFields
	Properties *runtime.RawExtension `json:"properties"`
}

// HeliosAppPhase defines the phase of HeliosApp
type HeliosAppPhase string

const (
	PhasePending  HeliosAppPhase = "Pending"
	PhaseReady    HeliosAppPhase = "Ready"
	PhaseFailed   HeliosAppPhase = "Failed"
	PhaseDeleting HeliosAppPhase = "Deleting"
)

// HeliosAppStatus defines the observed state of HeliosApp
type HeliosAppStatus struct {

	// Phase represents the current phase of the application
	// +optional
	// +kubebuilder:validation:Enum=Pending;Ready;Failed;Deleting
	Phase HeliosAppPhase `json:"phase,omitempty"`

	// Message provides additional information about the status
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResourcesCreated lists the resources created by this application
	// +optional
	ResourcesCreated []ResourceRef `json:"resourcesCreated,omitempty"`

	// LastAppliedHash is the hash of the last successfully synced manifest
	// +optional
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`

	// InitialBuildTriggered indicates if the initial PipelineRun was created
	// +optional
	InitialBuildTriggered bool `json:"initialBuildTriggered,omitempty"`
}

// ResourceRef references a Kubernetes resource
type ResourceRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.spec.owner`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HeliosApp is the Schema for the heliosapps API
type HeliosApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HeliosAppSpec   `json:"spec,omitempty"`
	Status HeliosAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HeliosAppList contains a list of HeliosApp
type HeliosAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HeliosApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HeliosApp{}, &HeliosAppList{})
}
