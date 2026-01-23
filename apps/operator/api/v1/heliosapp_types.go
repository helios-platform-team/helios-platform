/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HeliosAppSpec defines the desired state of HeliosApp
type HeliosAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// Môi trường hoạt động của ứng dụng (Environment Variables)
	Env []EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// Cấu hình tài nguyên (CPU, Memory)
	Resources *ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Bật tính năng tự động tối ưu hóa tài nguyên (Smart Rightsizer)
	EnableAutoOptimization bool `json:"enableAutoOptimization,omitempty"`

	// +kubebuilder:validation:Required
	GitRepo string `json:"gitRepo"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="main"
	GitBranch string `json:"gitBranch,omitempty"`

	// +kubebuilder:validation:Required
	ImageRepo string `json:"imageRepo"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	// Tên của Tekton Pipeline sẽ được kích hoạt.
	PipelineName string `json:"pipelineName"`

	// +kubebuilder:validation:Required
	// Tên của ServiceAccount mà PipelineRun sẽ sử dụng.
	ServiceAccount string `json:"serviceAccount"`

	// +kubebuilder:validation:Required
	// Tên của Secret chứa mã webhook cho GitHub.
	WebhookSecret string `json:"webhookSecret"`

	PVCName string `json:"pvcName"`

	// +kubebuilder:validation:Required
	// URL của repository chứa Helm chart hoặc Kustomize template
	TemplateRepo string `json:"templateRepo"`

	// +kubebuilder:validation:Required
	// Đường dẫn đến template trong templateRepo (ví dụ: "charts/my-app")
	TemplatePath string `json:"templatePath"`

	// +kubebuilder:validation:Required
	// URL của GitOps repository trung tâm nơi lưu trữ manifest đã render
	GitOpsRepo string `json:"gitopsRepo"`

	// +kubebuilder:validation:Required
	// Đường dẫn đến thư mục của ứng dụng trong GitOps repo (ví dụ: "apps/production/my-api")
	GitOpsPath string `json:"gitopsPath"`

	// +kubebuilder:validation:Optional
	// Subdirectory trong source repo chứa Dockerfile (ví dụ: "my-app", để trống nếu Dockerfile ở root)
	ContextSubpath string `json:"contextSubpath,omitempty"`

	// +kubebuilder:validation:Optional
	// Các giá trị tùy chỉnh để render template (ví dụ: ingress.host, resources, etc.)
	// +kubebuilder:pruning:PreserveUnknownFields
	Values map[string]string `json:"values,omitempty"`
}

// HeliosAppStatus defines the observed state of HeliosApp
type HeliosAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Trạng thái tổng quát của App: "Pending", "Syncing", "Healthy", "Failed"
	// +optional
	Phase string `json:"phase,omitempty"`

	// Chi tiết về trạng thái tối ưu hóa
	// +optional
	OptimizationStatus string `json:"optimizationStatus,omitempty"` // "Idle", "Analyzing", "WasteDetected", "Optimized"

	// Lý do chi tiết cho trạng thái hiện tại (Lỗi gì, chờ gì...)
	// +optional
	Message string `json:"message,omitempty"`

	// Trạng thái hiện tại của các tài nguyên được quản lý: "DeploymentReady", "ServiceReady"
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Tag của image đã được deploy thành công
	// +optional
	DeployedVersion string `json:"deployedVersion,omitempty"`

	// Generation của HeliosApp đã được quan sát và xử lý
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Tên của PipelineRun đang chạy hoặc đã hoàn thành cho manifest generation
	// +optional
	ManifestPipelineRun string `json:"manifestPipelineRun,omitempty"`

	// Tên của ArgoCD Application đã được tạo
	// +optional
	ArgoApplication string `json:"argoApplication,omitempty"`

	// CurrentCPU lưu trữ mức tiêu thụ CPU thực tế (được optimize service đo lường)
	// +optional
	CurrentCPU string `json:"currentCPU,omitempty"`

	// CurrentMemory lưu trữ mức tiêu thụ Memory thực tế
	// +optional
	CurrentMemory string `json:"currentMemory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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

// EnvVar defines an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResourceRequirements defines resource requests and limits
type ResourceRequirements struct {
	// +kubebuilder:validation:Optional
	Requests map[string]string `json:"requests,omitempty"`
	// +kubebuilder:validation:Optional
	Limits map[string]string `json:"limits,omitempty"`
}

func init() {
	SchemeBuilder.Register(&HeliosApp{}, &HeliosAppList{})
}
