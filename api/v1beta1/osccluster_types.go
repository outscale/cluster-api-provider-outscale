/*
Copyright 2022 The Kubernetes Authors.

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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func OscReplaceName(name string) string {
	replacer := strings.NewReplacer(".", "-", "/", "-", "_", "-")
	return replacer.Replace(name)
}

// OscClusterSpec defines the desired state of OscCluster
type OscClusterSpec struct {
	Network              OscNetwork            `json:"network,omitempty"`
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

// OscClusterStatus defines the observed state of OscCluster
type OscClusterStatus struct {
	Ready          bool                     `json:"ready,omitempty"`
	Network        OscNetworkResource       `json:"network,omitempty"`
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`
	Conditions     clusterv1.Conditions     `json:"conditions,omitempty"`
	VmState        *VmState                 `json:"vmState,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OscCluster is the Schema for the oscclusters API
type OscCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscClusterSpec   `json:"spec,omitempty"`
	Status OscClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OscClusterList contains a list of OscCluster
type OscClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscCluster `json:"items"`
}

// GetConditions returns status of the state of the cluster resource.
func (r *OscCluster) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions set status of the state of the cluster resource from clusterv1.Conditions.
func (r *OscCluster) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&OscCluster{}, &OscClusterList{})
}
