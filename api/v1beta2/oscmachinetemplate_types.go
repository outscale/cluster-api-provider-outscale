/*
Copyright 2022.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

// OscMachineTemplateSpec defines the desired state of OscMachineTemplate
type OscMachineTemplateSpec struct {
	Template OscMachineTemplateResource `json:"template"`
}

// OscMachineTemplateResource is the Schema for the OscMachineTemplate api
type OscMachineTemplateResource struct {
	ObjectMeta clusterv1beta1.ObjectMeta `json:"metadata,omitempty"`
	Spec       OscMachineSpec            `json:"spec"`
}

// OscMachineTemplateStatus defines the observed state of OscMachineTemplate
type OscMachineTemplateStatus struct {
	Capacity   corev1.ResourceList       `json:"capacity,omitempty"`
	Conditions clusterv1beta1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=oscmachinetemplates,scope=Namespaced,categories=cluster-api

// OscMachineTemplate is the Schema for the oscmachinetemplates API
type OscMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscMachineTemplateSpec   `json:"spec,omitempty"`
	Status OscMachineTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OscMachineTemplateList contains a list of OscMachineTemplate
type OscMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OscMachineTemplate{}, &OscMachineTemplateList{})
}
