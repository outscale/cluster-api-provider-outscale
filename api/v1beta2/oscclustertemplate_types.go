/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// OscClusterTemplateSpec defines the desired state of OscClusterTemplate
type OscClusterTemplateSpec struct {
	Template OscClusterTemplateResource `json:"template"`
}

type OscClusterTemplateResource struct {
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       OscClusterSpec       `json:"spec"`
}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion

// OscClusterTemplate is the Schema for the oscclustertemplates API
type OscClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OscClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// OscClusterTemplateList contains a list of OscClusterTemplate
type OscClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OscClusterTemplate{}, &OscClusterTemplateList{})
}
