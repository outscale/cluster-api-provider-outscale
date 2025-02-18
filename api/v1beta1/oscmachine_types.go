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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// OscMachineSpec defines the desired state of OscMachine
type OscMachineSpec struct {
	ProviderID *string `json:"providerID,omitempty"`
	Node       OscNode `json:"node,omitempty"`
}

// OscMachineStatus defines the observed state of OscMachine
type OscMachineStatus struct {
	Ready          bool                       `json:"ready,omitempty"`
	Addresses      []corev1.NodeAddress       `json:"addresses,omitempty"`
	FailureReason  *errors.MachineStatusError `json:"failureReason,omitempty"`
	VmState        *VmState                   `json:"vmState,omitempty"`
	Node           OscNodeResource            `json:"node,omitempty"`
	FailureMessage *string                    `json:"failureMessage,omitempty"`
	Conditions     clusterv1.Conditions       `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=oscmachines,scope=Namespaced,categories=cluster-api

// OscMachine is the Schema for the oscmachines API
type OscMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscMachineSpec   `json:"spec,omitempty"`
	Status OscMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OscMachineList contains a list of OscMachine
type OscMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscMachine `json:"items"`
}

// GetConditions return status of the state of the machine resource
func (r *OscMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions set status of the state of the machine resource from machine
func (r *OscMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&OscMachine{}, &OscMachineList{})
}
