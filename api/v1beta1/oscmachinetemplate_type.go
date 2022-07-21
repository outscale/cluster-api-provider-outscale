package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

//OscMachineTemplateSpec define oscMachine template
type OscMachineTemplateSpec struct {
	Template OscMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=oscmachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// OscMachineTemplate is the Schema for the OscMachineTemplate API
type OscMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OscMachineTemplateSpec `json:"spec,omitempty"`
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

func init() {
	SchemeBuilder.Register(&OscMachineTemplate{}, &OscMachineTemplateList{})
}
