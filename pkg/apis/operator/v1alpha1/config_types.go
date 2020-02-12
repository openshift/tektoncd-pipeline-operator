package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigSpec defines the desired state of Config
// +k8s:openapi-gen=true
type ConfigSpec struct {
	// namespace where OpenShift pipelines will be installed
	TargetNamespace string `json:"targetNamespace"`
}

// ConfigStatus defines the observed state of Config
// +k8s:openapi-gen=true
type ConfigStatus struct {

	// OperatorUUID is the  uuid (auto-generated) of the operator that
	// installed the pipeline
	OperatorUUID string `json:"operatorUUID,omitempty"`

	// installation status sorted in reverse chronological order
	Conditions []ConfigCondition `json:"conditions,omitempty"`
}

// ConfigCondition defines the observed state of installation at a point in time
// +k8s:openapi-gen=true
type ConfigCondition struct {
	// Code indicates the status of installation of pipeline resources
	// Valid values are:
	//   - "error"
	//   - "installing"
	//   - "installed"
	Code InstallStatus `json:"code"`

	// Additional details about the Code
	Details string `json:"details,omitempty"`

	// The version of OpenShift pipelines
	Version string `json:"version"`
}

// InstallStatus describes the state of installation of pipelines
// +kubebuilder:validation:Enum=Allow;Forbid;Replace
type InstallStatus string

const (

	// EmptyStatus indicates that the core pipeline resources have not even been
	// applied
	EmptyStatus InstallStatus = "empty"

	// InstallingPipeline indicates that the core pipeline resources
	// are being installed
	//InstallingPipeline InstallStatus = "installing-pipeline"

	// AppliedPipeline indicates that the core pipeline resources
	// have been applied on the cluster
	AppliedPipeline InstallStatus = "applied-pipeline"

	// PipelineError indicates that there was an error installing pipeline resources
	// Check details field for additional details
	PipelineApplyError InstallStatus = "error-pipeline-apply"

	// ValidatedPipeline indicates that core pipeline resources have been
	// installed successfully
	ValidatedPipeline InstallStatus = "validated-pipeline"

	// PipelineVali indicates that there was an error installing pipeline resources
	// Check details field for additional details
	PipelineValidateError InstallStatus = "error-pipeline-validate"

	// InvalidResource indicates that the resource specification is invalid
	// Check details field for additional details
	InvalidResource InstallStatus = "invalid-resource"

	// AppliedAddons indicates that the pipeline addons
	// have been applied on the cluster
	AppliedAddons InstallStatus = "applied-addons"

	// AddonsError indicates that there was an error installing addons
	// Check details field for additional details
	AddonsError InstallStatus = "error-addons"

	// NonRedHatResourcesError indicates that there was an error
	// installing non Red Hat Resources
	// Check details field for additional details
	NonRedHatResourcesError InstallStatus = "error-non-redhat-resources"

	// InstalledStatus indicates that all pipeline resources are installed successfully
	InstalledStatus InstallStatus = "installed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config is the Schema for the configs API
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=config
// +kubebuilder:subresource:status
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}

func (c *Config) InstallStatus() InstallStatus {
	con := c.Status.Conditions
	if len(con) == 0 {
		return EmptyStatus
	}
	return con[0].Code
}

func (c *Config) HasInstalledVersion(target string) bool {
	return c.InstallStatus() == InstalledStatus &&
		c.Status.Conditions[0].Version == target
}
