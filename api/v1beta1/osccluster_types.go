/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
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
	Credentials          OscCredentials        `json:"credentials,omitempty"`
	Network              OscNetwork            `json:"network,omitempty"`
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`
}

// OscClusterStatus defines the observed state of OscCluster
type OscClusterStatus struct {
	Ready bool `json:"ready,omitempty"`
	// deprecated, replaced by resources
	Network              OscNetworkResource       `json:"network,omitempty"`
	Resources            OscClusterResources      `json:"resources,omitempty"`
	ReconcilerGeneration OscReconcilerGeneration  `json:"reconcilerGeneration,omitempty"`
	FailureDomains       clusterv1.FailureDomains `json:"failureDomains,omitempty"`
	Conditions           clusterv1.Conditions     `json:"conditions,omitempty"`
	VmState              *VmState                 `json:"vmState,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=oscclusters,scope=Namespaced,categories=cluster-api

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
