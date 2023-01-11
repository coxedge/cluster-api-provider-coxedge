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
	// ClusterFinalizer allows ReconcileCoxCluster to clean up Cox resources
	// associated with CoxCluster before removing it from the apiserver.
	ClusterFinalizer = "coxcluster.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoxClusterSpec defines the desired state of CoxCluster
type CoxClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1beta1.APIEndpoint `json:"controlPlaneEndpoint"`

	// Credentials is a reference to an identity to be used when reconciling this cluster.
	// +optional
	Credentials *corev1.LocalObjectReference `json:"credentials,omitempty"`

	// ControlPlaneLoadBalancer is optional configuration for customizing control plane behavior.
	// +optional
	ControlPlaneLoadBalancer CoxLoadBalancerSpec `json:"controlPlaneLoadBalancer,omitempty"`

	WorkersLoadBalancer CoxLoadBalancerSpec `json:"workersLoadBalancer,omitempty"`
}

// CoxClusterStatus defines the observed state of CoxCluster
type CoxClusterStatus struct {
	// Ready denotes that the cluster is ready.
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the Machine.
	// +optional
	Conditions clusterv1beta1.Conditions `json:"conditions,omitempty"`

	// +optional
	ControlPlaneLoadBalancer CoxLoadBalancerStatus `json:"controlPlaneLoadBalancer,omitempty"`

	WorkersLoadBalancer CoxLoadBalancerStatus `json:"workersLoadBalancer,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this CoxCluster belongs"
// +kubebuilder:printcolumn:name="Credentials",type="string",JSONPath=".spec.credentials.name",description="Cluster Credentials"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint",description="API Endpoint"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for Cox instances"

// CoxCluster is the Schema for the coxclusters API
type CoxCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoxClusterSpec   `json:"spec,omitempty"`
	Status CoxClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CoxClusterList contains a list of CoxCluster
type CoxClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoxCluster `json:"items"`
}

type CoxLoadBalancerSpec struct {
	// +optional
	Name string `json:"name"`

	// +optional
	Image string `json:"image,omitempty"`

	// +optional
	Ports []int32 `json:"port,omitempty"`

	// POP for instance
	POP []string `json:"pop,omitempty"`
}

type CoxLoadBalancerStatus struct {
	// +optional
	PublicIP string `json:"publicIP"`
}

// GetConditions returns the set of conditions for this object.
func (m *CoxCluster) GetConditions() clusterv1beta1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (m *CoxCluster) SetConditions(conditions clusterv1beta1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&CoxCluster{}, &CoxClusterList{})
}
