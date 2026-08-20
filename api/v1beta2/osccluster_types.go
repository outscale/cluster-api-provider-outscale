/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	"strings"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func OscReplaceName(name string) string {
	replacer := strings.NewReplacer(".", "-", "/", "-", "_", "-")
	return replacer.Replace(name)
}

// OscClusterSpec defines the desired state of OscCluster
type OscClusterSpec struct {
	// Credentials to use to build this cluster
	// +optional
	Credentials OscCredentials `json:"credentials,omitempty,omitzero"`
	// Reuse externally managed resources ?
	// +optional
	UseExisting OscReuse `json:"useExisting,omitempty,omitzero"`
	// List of disabled features (internet = no internet service, no nat services)
	// +optional
	Disable []OscDisable `json:"disable,omitempty"`
	// The Load Balancer configuration
	// +optional
	LoadBalancer OscLoadBalancer `json:"loadBalancer,omitempty,omitzero"`
	// The Net configuration
	// +optional
	Net OscNet `json:"net,omitempty,omitzero"`
	// The NetPeering configuration, required if the load balancer is internal, and management and workload clusters are on separate VPCs.
	// +optional
	NetPeering OscNetPeering `json:"netPeering,omitempty,omitzero"`
	// The NetAccessPoints configuration, required if internet is disabled.
	// +optional
	NetAccessPoints []OscNetAccessPointService `json:"netAccessPoints,omitempty"`
	// List of subnet to spread controlPlane nodes (deprecated, add controlplane role to subnets)
	// +optional
	ControlPlaneSubnets []string `json:"controlPlaneSubnets,omitempty"`
	// The Subnets configuration
	// +optional
	Subnets []OscSubnet `json:"subnets,omitempty"`
	// The Internet Service configuration
	// +optional
	InternetService OscInternetService `json:"internetService,omitempty,omitzero"`
	// The Nat Service configuration
	// +optional
	NatService OscNatService `json:"natService,omitempty,omitzero"`
	// The Nat Services configuration
	// +optional
	NatServices []OscNatService `json:"natServices,omitempty"`
	// The IP Pool storing the Nat Services public IPs
	// +optional
	NatPublicIpPool string `json:"natPublicIpPool,omitempty"`
	// The Route Table configuration
	// +optional
	RouteTables []OscRouteTable `json:"routeTables,omitempty"`
	// The Security Groups configuration.
	// +optional
	SecurityGroups []OscSecurityGroup `json:"securityGroups,omitempty"`
	// Additional rules to add to the automatic security groups
	// +optional
	AdditionalSecurityRules []OscAdditionalSecurityRules `json:"additionalSecurityRules,omitempty"`
	// The bastion configuration
	// + optional
	Bastion OscBastion `json:"bastion,omitempty,omitzero"`
	// The default subregion name (deprecated, use subregions)
	SubregionName string `json:"subregionName,omitempty"`
	// The list of subregions where to deploy this cluster
	Subregions []string `json:"subregions,omitempty"`
	// The list of IP ranges (in CIDR notation) to restrict bastion/Kubernetes API access to.
	// + optional
	AllowFromIPRanges []string `json:"allowFromIPRanges,omitempty"`
	// The list of IP ranges (in CIDR notation) the nodes can talk to ("0.0.0.0/0" if not set).
	// + optional
	AllowToIPRanges []string `json:"allowToIPRanges,omitempty"`
	// Reconciliation rules (default: {securityGroup, random, 10%}, {*, onChange}). Only the first matching rule applies.
	// + optional
	ReconciliationRules []OscReconciliationRule `json:"reconciliationRules,omitempty"`

	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`
}

// OscClusterStatus defines the observed state of OscCluster
type OscClusterStatus struct {
	Ready                bool                     `json:"ready,omitempty"`
	Resources            OscClusterResources      `json:"resources,omitempty,omitzero"`
	ReconcilerGeneration OscReconcilerGeneration  `json:"reconcilerGeneration,omitempty"`
	FailureDomains       clusterv1.FailureDomains `json:"failureDomains,omitempty"`
	Conditions           clusterv1.Conditions     `json:"conditions,omitempty"`
	VmState              *osc.VmState             `json:"vmState,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=oscclusters,scope=Namespaced,categories=cluster-api
//+kubebuilder:storageversion

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
