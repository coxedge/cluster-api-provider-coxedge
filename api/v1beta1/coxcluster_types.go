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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CoxClusterSpec defines the desired state of CoxCluster
type CoxClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CoxCluster. Edit coxcluster_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// CoxClusterStatus defines the observed state of CoxCluster
type CoxClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CoxCluster is the Schema for the coxclusters API
type CoxCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CoxClusterSpec   `json:"spec,omitempty"`
	Status CoxClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CoxClusterList contains a list of CoxCluster
type CoxClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CoxCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CoxCluster{}, &CoxClusterList{})
}
