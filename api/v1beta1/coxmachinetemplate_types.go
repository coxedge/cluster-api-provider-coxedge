package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CoxMachineTemplateSpec defines the desired state of CoxMachineTemplate.
type CoxMachineTemplateSpec struct {
	Template CoxMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=coxmachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// CoxMachineTemplate is the Schema for the coxmachinetemplates API.
type CoxMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CoxMachineTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// CoxMachineTemplateList contains a list of CoxMachineTemplate.
type CoxMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoxMachineTemplate `json:"items"`
}

// CoxMachineTemplateResource describes the data needed to create am CoxMachine from a template.
type CoxMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec CoxMachineSpec `json:"spec"`
}

func init() {
	SchemeBuilder.Register(&CoxMachineTemplate{}, &CoxMachineTemplateList{})
}
