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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// HeliosAppSpec defines the desired state of HeliosApp
// Follows OAM (Open Application Model) with components and traits
type HeliosAppSpec struct {
	// Owner team of the application
	// +optional
	Owner string `json:"owner,omitempty"`

	// Description of the application
	// +optional
	Description string `json:"description,omitempty"`

	// Components define the workloads of the application
	// +kubebuilder:validation:MinItems=1
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
