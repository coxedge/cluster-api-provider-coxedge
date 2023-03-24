/*
Copyright 2021.

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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// MachineFinalizer allows ReconcileCoxMachine to clean up Cox resources
	// associated with CoxCluster before removing it from the apiserver.
	MachineFinalizer = "coxmachine.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoxMachineSpec defines the desired state of CoxMachine
type CoxMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID string `json:"providerID,omitempty"`

	// AddAnyCastIPAddress enables the AnyCast IP Address feature.
	// +optional
	AddAnyCastIPAddress bool `json:"addAnycastIPAddress,omitempty"`

	// PersistentStorages mount storage volumes to your workload instances.
	// +optional
	PersistentStorages []PersistentStorage `json:"persistentStorages,omitempty"`

	// Expose any ports required by your workload instances
	// +optional
	Ports []Port `json:"ports,omitempty"`

	// SSHAuthorizedKeys contains the public SSH keys that should be added to
	// the machine on first boot. In the CoxEdge API this field is equivalent
	// to `firstBootSSHKey`.
	// +optional
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys,omitempty"`

	// Deployment targets
	// +optional
	Deployments []Deployment `json:"deployments,omitempty"`

	// Specs contains the flavor of the machine. For example, SP-5.
	// +optional
	Specs string `json:"specs,omitempty"`

	// Image is a reference to the OS image that should be used to provision
	// the VM.
	// +optional
	Image string `json:"image,omitempty"`

	// User data compatible with cloud-init
	// +optional
	// UserData string `json:"userData,omitempty"`
}

// Deployment defines instance specifications
type Deployment struct {
	// Name of the deployment instance
	// +optional
	Name string `json:"name,omitempty"`

	// CoxEdge PoPs - geographical location for the instance
	// +optional
	Pops []string `json:"pops,omitempty"`

	// +optional
	EnableAutoScaling bool `json:"enableAutoScaling,omitempty"`

	// number of instances per each PoP defined
	// +optional
	InstancesPerPop string `json:"instancesPerPop,omitempty"`

	// +optional
	CPUUtilization int `json:"cpuUtilization,omitempty"`

	// +optional
	MinInstancesPerPop string `json:"minInstancesPerPop,omitempty"`

	// +optional
	MaxInstancesPerPop string `json:"maxInstancesPerPop,omitempty"`
}

// Port defines instance network policies
type Port struct {
	Protocol       string `json:"protocol"`
	PublicPort     string `json:"publicPort"`
	PublicPortDesc string `json:"publicPortDesc,omitempty"`
}

// PersistentStorage defines instances' mounted persistent storage options
type PersistentStorage struct {
	Path string `json:"path"`
	Size string `json:"size"`
}

// CoxMachineStatus defines the observed state of CoxMachine
type CoxMachineStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	TaskID       string  `json:"taskID,omitempty"`
	TaskStatus   string  `json:"taskStatus,omitempty"`
	Ready        bool    `json:"ready,omitempty"`
	ErrorMessage *string `json:"errormessage,omitempty"`

	// Conditions defines current service state of the Machine.
	// +optional
	Conditions clusterv1beta1.Conditions `json:"conditions,omitempty"`

	// Addresses contains the IP and/or DNS addresses of the CoxEdge instances.
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this CoxMachine belongs"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this CoxMachine"
// +kubebuilder:printcolumn:name="WorkloadID",type="string",JSONPath=".spec.providerID",description="CoxEdge workload ID"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"

// CoxMachine is the Schema for the coxmachines API
type CoxMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoxMachineSpec   `json:"spec,omitempty"`
	Status CoxMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CoxMachineList contains a list of CoxMachine
type CoxMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoxMachine `json:"items"`
}

// GetConditions returns the set of conditions for this object.
func (m *CoxMachine) GetConditions() clusterv1beta1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (m *CoxMachine) SetConditions(conditions clusterv1beta1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&CoxMachine{}, &CoxMachineList{})
}
