/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< Updated upstream
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
=======
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	// TODO: drop this in schema
	"sigs.k8s.io/cluster-api/errors" //nolint
>>>>>>> Stashed changes
)

// OscMachineSpec defines the desired state of OscMachine
type OscMachineSpec struct {
	// providerID is the ID of the OscMachine. It is part of the Cluster API contract.
	ProviderID *string `json:"providerID,omitempty"`

	// vm is the definition of the VM.
	Vm OscVm `json:"vm,omitempty"`

	// reconciliationRule definition de rule used to reconcile OscMachine resources (default: {*, onChange})
	// +optional
	ReconciliationRule *OscReconciliationRule `json:"reconciliationRule,omitempty"`
}

type OscMachineResources struct {
	Vm        map[string]string `json:"vm,omitempty"`
	FGPU      map[string]string `json:"fGPU,omitempty"`
	Image     map[string]string `json:"image,omitempty"`
	Volumes   map[string]string `json:"volumes,omitempty"`
	PublicIPs map[string]string `json:"publicIps,omitempty"`
}

// OscMachineInitializationStatus provides observations of the OscMachine initialization process.
// +kubebuilder:validation:MinProperties=1
type OscMachineInitializationStatus struct {
	// provisioned is true when the infrastructure provider reports that the Machine's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Machine provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// OscMachineStatus defines the observed state of OscMachine
type OscMachineStatus struct {
	// initialization provides observations of the OscMachine initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Machine provisioning.
	// +optional
	Initialization OscMachineInitializationStatus `json:"initialization,omitempty,omitzero"`
	Conditions     clusterv1.Conditions           `json:"conditions,omitempty"`

	// addresses contains the associated addresses for the machine.
	// +optional
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`
	// failureDomain is the unique identifier of the failure domain where this Machine has been placed in.
	// +optional
	FailureDomain string `json:"failureDomain,omitempty"`

	// vmState is the state of the underlying VM.
	// +optional
	VmState *osc.VmState `json:"vmState,omitempty"`

	// resources tracks the IaaS resources used by the OscMachine.
	// +optional
	Resources OscMachineResources `json:"resources,omitempty"`
	// reconcilerGeneration tracks the last resource generation with a successful reconciliation.
	// +optional
	ReconcilerGeneration OscReconcilerGeneration `json:"reconcilerGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=oscmachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:printcolumn:name="VM Type",type=string,JSONPath=".spec.vm.vmType"
// +kubebuilder:printcolumn:name="ProviderID",type=string,JSONPath=".spec.providerID"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.vmState"

// OscMachine is the Schema for the oscmachines API
type OscMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscMachineSpec   `json:"spec,omitempty"`
	Status OscMachineStatus `json:"status,omitempty"`
}

func (s *OscMachine) GetV1Beta1Conditions() clusterv1.Conditions {
	return s.Status.Conditions
}

func (s *OscMachine) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	s.Status.Conditions = conditions
}

var _ v1beta1conditions.Setter = (*OscMachine)(nil)

//+kubebuilder:object:root=true

// OscMachineList contains a list of OscMachine
type OscMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OscMachine{}, &OscMachineList{})
}
