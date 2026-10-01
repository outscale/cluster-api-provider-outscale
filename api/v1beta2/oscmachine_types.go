/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	// TODO: drop this in schema
	"sigs.k8s.io/cluster-api/errors" //nolint
)

// OscMachineSpec defines the desired state of OscMachine
type OscMachineSpec struct {
	ProviderID *string `json:"providerID,omitempty"`

	// The configuration of the VM to deploy
	Vm OscVm `json:"vm,omitempty"`
	// The image to use
	Image OscImage `json:"image,omitempty"`
	// The list of volumes to attach to the VM (including the root volume)
	// +kubebuilder:validation:MaxItems=40
	// +kubebuilder:validation:items:XValidation:rule="(has(self.root) && self.root) || !(has(self.device) && self.device == '/dev/sda1')",message="/dev/sda1 is reserved for the root volume"
	// +kubebuilder:validation:items:XValidation:rule="(has(self.root) && self.root) || (has(self.device) && self.device != '')",message="non root volumes must have a device"
	// +kubebuilder:validation:XValidation:rule="self.exists_one(vol, has(vol.root) && vol.root)",message="there must not be more than one root volume"
	Volumes []OscVolume `json:"volumes,omitempty"`
	// Reconciliation rules (default: {*, onChange})
	// +optional
	ReconciliationRule *OscReconciliationRule `json:"reconciliationRule,omitempty"`
}

// OscMachineStatus defines the observed state of OscMachine
type OscMachineStatus struct {
	Ready                bool                       `json:"ready,omitempty"`
	Addresses            []corev1.NodeAddress       `json:"addresses,omitempty"`
	FailureDomain        *string                    `json:"failureDomain,omitempty"`
	FailureReason        *errors.MachineStatusError `json:"failureReason,omitempty"`
	FailureMessage       *string                    `json:"failureMessage,omitempty"`
	VmState              *osc.VmState               `json:"vmState,omitempty"`
	Resources            OscMachineResources        `json:"resources,omitempty,omitzero"`
	ReconcilerGeneration OscReconcilerGeneration    `json:"reconcilerGeneration,omitempty"`
	Conditions           clusterv1.Conditions       `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oscmachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:printcolumn:name="VM Type",type=string,JSONPath=".spec.vm.type"
// +kubebuilder:printcolumn:name="ProviderID",type=string,JSONPath=".spec.providerID"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.vmState"
// +kubebuilder:storageversion

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
