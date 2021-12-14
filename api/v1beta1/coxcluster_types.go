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
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CoxCluster. Edit coxcluster_types.go to remove/update
	Foo                  string                     `json:"foo,omitempty"`
	ControlPlaneEndpoint clusterv1beta1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// CoxClusterStatus defines the observed state of CoxCluster
type CoxClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Ready bool `json:"ready"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this CoxCluster belongs"
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

func init() {
	SchemeBuilder.Register(&CoxCluster{}, &CoxClusterList{})
}
