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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// OscMachineTemplateSpec define oscMachine template
type OscMachineTemplateSpec struct {
	Template OscMachineTemplateResource `json:"template"`
}

type OscMachineTemplateStatus struct {
	Capacity   corev1.ResourceList  `json:"capacity,omitempty"`
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:subresource:status
//+kubebuilder:object:root=true
//+kubebuilder:resource:path=oscmachinetemplates,scope=Namespaced,categories=cluster-api
//+kubebuilder:storageversion

// OscMachineTemplate is the Schema for the OscMachineTemplate API
type OscMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscMachineTemplateSpec   `json:"spec,omitempty"`
	Status OscMachineTemplateStatus `json:"status,omitempty"`
}

// OscMachineTemplateList contains a list of OscMachineTemplate
// +kubebuilder:object:root=true
type OscMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitemmpty"`
	Items           []OscMachineTemplate `json:"items"`
}

// OscMachineTemplateResource is the Schema for the OscMachineTemplate api
type OscMachineTemplateResource struct {
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       OscMachineSpec       `json:"spec"`
}

func (m *OscMachineTemplate) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

func (m *OscMachineTemplate) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&OscMachineTemplate{}, &OscMachineTemplateList{})
}
