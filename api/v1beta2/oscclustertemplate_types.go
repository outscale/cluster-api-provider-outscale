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

package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// OscClusterTemplateSpec defines the desired state of OscClusterTemplate
type OscClusterTemplateSpec struct {
	Template OscClusterTemplateResource `json:"template"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=oscclustertemplates,scope=Namespaced,categories=cluster-api
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

type OscClusterTemplateResource struct {
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       OscClusterSpec       `json:"spec"`
}

func init() {
	SchemeBuilder.Register(&OscClusterTemplate{}, &OscClusterTemplateList{})
}
