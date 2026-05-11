/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	"slices"
	"strings"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// OscClusterSpec defines the desired state of OscCluster
type OscClusterSpec struct {
	// controlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`

	Credentials OscCredentials `json:"credentials,omitempty"`

	// Reuse externally managed resources ?
	// +optional
	UseExisting OscReuse `json:"useExisting,omitempty"`
	// List of disabled features (internet = no internet service, no nat services)
	// +optional
	Disable []OscDisable `json:"disable,omitempty"`
	// The Load Balancer configuration
	// +optional
	LoadBalancer OscLoadBalancer `json:"loadBalancer,omitempty"`
	// The Net configuration
	// +optional
	Net OscNet `json:"net,omitempty"`
	// The NetPeering configuration, required if the load balancer is internal, and management and workload clusters are on separate VPCs.
	// +optional
	NetPeering OscNetPeering `json:"netPeering,omitempty"`
	// The NetAccessPoints configuration, required if internet is disabled.
	// +optional
	NetAccessPoints []OscNetAccessPointService `json:"netAccessPoints,omitempty"`
	// The Subnets configuration
	// +optional
	Subnets []OscSubnet `json:"subnets,omitempty"`
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
	Bastion *OscBastion `json:"bastion,omitempty"`
	// The list of subregions where to deploy this cluster
	Subregions []string `json:"subregions,omitempty"`
	// (unused)
	ExtraSecurityGroupRule bool `json:"extraSecurityGroupRule,omitempty"`
	// The list of IP ranges (in CIDR notation) to restrict bastion/Kubernetes API access to.
	// + optional
	AllowFromIPRanges []string `json:"allowFromIPRanges,omitempty"`
	// The list of IP ranges (in CIDR notation) the nodes can talk to ("0.0.0.0/0" if not set).
	// + optional
	AllowToIPRanges []string `json:"allowToIPRanges,omitempty"`
	// Reconciliation rules (default: {securityGroup, random, 10%}, {*, onChange}). Only the first matching rule applies.
	// + optional
	ReconciliationRules []OscReconciliationRule `json:"reconciliationRules,omitempty"`
}

type OscBastion struct {
	Enable bool   `json:"enable,omitempty"`
	VM     *OscVm `json:"vm,omitempty"`
}
type OscCredentials struct {
	// Load credentials from this secret instead of the env.
	// +optional
	FromSecret string `json:"fromSecret,omitempty"`
	// Load credentials from this file instead of the env.
	// +optional
	FromFile string `json:"fromFile,omitempty"`
	// Name of profile stored in file (unused using fromSecret, "default" by default).
	// +optional
	Profile string `json:"profile,omitempty"`
}

type OscReuse struct {
	// If set, net, subnets, internet service, nat services and route tables are externally managed
	Net bool `json:"net,omitempty"`
	// If set, security groups are externally managed.
	SecurityGroups bool `json:"securityGroups,omitempty"`
}

// +kubebuilder:validation:Enum:=internet;loadbalancer
type OscDisable string

const (
	DisableInternet OscDisable = "internet"
	DisableLB       OscDisable = "loadbalancer"
)

type OscLoadBalancer struct {
	// The Load Balancer unique name
	// +optional
	LoadBalancerName string `json:"loadbalancername,omitempty"`
	// The Load Balancer type (internet-facing or internal)
	// +optional
	LoadBalancerType string `json:"loadbalancertype,omitempty"`
	// The Listener configuration of the loadBalancer
	// +optional
	Listener OscLoadBalancerListener `json:"listener,omitempty"`
	// The healthCheck configuration of the Load Balancer
	// +optional
	HealthCheck OscLoadBalancerHealthCheck `json:"healthCheck,omitempty"`
}

type OscLoadBalancerListener struct {
	// The port on which the backend VMs will listen
	// +optional
	BackendPort int `json:"backendport,omitempty"`
	// The protocol ('HTTP'|'TCP') to route the traffic to the backend vm
	// +optional
	BackendProtocol string `json:"backendprotocol,omitempty"`
	// The port on which the loadbalancer will listen
	// +optional
	LoadBalancerPort int `json:"loadbalancerport,omitempty"`
	// the routing protocol ('HTTP'|'TCP')
	// +optional
	LoadBalancerProtocol string `json:"loadbalancerprotocol,omitempty"`
}

type OscLoadBalancerHealthCheck struct {
	// the time in second between two pings
	// +optional
	CheckInterval int `json:"checkinterval,omitempty"`
	// the consecutive number of pings which are successful to consider the vm healthy
	// +optional
	HealthyThreshold int `json:"healthythreshold,omitempty"`
	// the HealthCheck port number
	// +optional
	Port int `json:"port,omitempty"`
	// The HealthCheck protocol ('HTTP'|'TCP')
	// +optional
	Protocol string `json:"protocol,omitempty"`
	// the Timeout to consider VM unhealthy
	// +optional
	Timeout int `json:"timeout,omitempty"`
	// the consecutive number of pings which are failed to consider the vm unhealthy
	// +optional
	UnhealthyThreshold int `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
	// the ip range in CIDR notation of the Net
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// The Id of the Net to reuse (if useExisting.net is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

func (o *OscNet) IsZero() bool {
	return o.IpRange == "" && o.ResourceId == ""
}

var DefaultNet = OscNet{
	IpRange: "10.0.0.0/16",
}

type OscNetPeering struct {
	// Create a NetPeering between the management and workload VPCs.
	// +optional
	Enable bool `json:"enable,omitempty"`
	// The credentials of the management cluster account. Required if management and workload cluster are not in the same account.
	// +optional
	ManagementCredentials OscCredentials `json:"managementCredentials,omitempty"`
	// The management cluster account ID (optional, fetched from the metadata server if not set).
	// +optional
	ManagementAccountID string `json:"managementAccountId,omitempty"`
	// The management cluster net ID (optional, fetched from the metadata server if not set).
	// +optional
	ManagementNetID string `json:"managementNetId,omitempty"`
	// By default, all subnets of managementNetId are routed to the netPeering. If set, only the specified subnet will be routed.
	// +optional
	ManagementSubnetID string `json:"managementSubnetId,omitempty"`
}

// +kubebuilder:validation:Enum:=api;directlink;eim;kms;lbu;oos
type OscNetAccessPointService string

const (
	ServiceAPI        OscNetAccessPointService = "api"
	ServiceDirectLink OscNetAccessPointService = "directlink"
	ServiceEIM        OscNetAccessPointService = "eim"
	ServiceKMS        OscNetAccessPointService = "kms"
	ServiceLBU        OscNetAccessPointService = "lbu"
	ServiceOOS        OscNetAccessPointService = "oos"
)

type OscSubnet struct {
	// The role of the Subnet (controlplane, worker, loadbalancer, bastion, nat or any user-defined value)
	// +optional
	Roles []OscRole `json:"roles,omitempty"`
	// the Ip range in CIDR notation of the Subnet
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// The subregion name of the Subnet
	// +optional
	Subregion string `json:"subregion,omitempty"`
	// The id of the Subnet to reuse (if useExisting.net is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscRouteTable struct {
	// The role for this route table
	// +optional
	Role OscRole `json:"role,omitempty"`
	// The subregion for this route table
	// +optional
	Subregion string `json:"subregion,omitempty"`
	// The Route configuration
	// +optional
	Routes []OscRoute `json:"routes,omitempty"`
}

type OscSecurityGroup struct {
	// The name of the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The description of the security group
	// +optional
	Description string `json:"description,omitempty"`
	// The list of rules for this securityGroup.
	// +optional
	SecurityGroupRules []OscSecurityGroupRule `json:"securityGroupRules,omitempty"`
	// When useExisting.securityGroup is set, the id of an existing securityGroup to use.
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
	Tag        string `json:"tag,omitempty"`
	// The roles the securityGroup applies to.
	Roles []OscRole `json:"roles,omitempty"`
	// Is the Security Group configuration authoritative ? (if yes, all rules not found in configuration will be deleted).
	// +optional
	Authoritative bool `json:"authoritative,omitempty"`
}

func (sg *OscSecurityGroup) HasRole(role OscRole) bool {
	if len(sg.Roles) > 0 {
		return slices.Contains(sg.Roles, role)
	}
	if strings.Contains(sg.Name, "kcp") {
		return role == RoleControlPlane
	}
	if strings.Contains(sg.Name, "kw") {
		return role == RoleWorker
	}
	if strings.Contains(sg.Name, "node") {
		return role == RoleControlPlane || role == RoleWorker
	}
	return false
}

type OscAdditionalSecurityRules struct {
	// The roles of automatic securityGroup to add rules to.
	// +optional
	Roles []OscRole `json:"roles,omitempty"`
	// The rules to add.
	// +optional
	Rules []OscSecurityGroupRule `json:"rules,omitempty"`
}

// +kubebuilder:validation:Enum:=gateway;nat-service
type OscTargetType string

const (
	TargetTypeInternetService OscTargetType = "gateway"
	TargetTypeNatService      OscTargetType = "nat-service"
)

type OscRoute struct {
	// The tag name associate with the target resource type
	// +optional
	TargetName string `json:"targetName,omitempty"`
	// The target resource type which can be Internet Service (gateway) or Nat Service (nat-service)
	// +optional
	TargetType OscTargetType `json:"targetType,omitempty"`
	// the destination match Ip range with CIDR notation
	// +optional
	Destination string `json:"destination,omitempty"`
}

type OscSecurityGroupElement struct {
	Name string `json:"name,omitempty"`
}

type OscSecurityGroupRule struct {
	// The tag name associate with the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The flow of the security group (inbound or outbound)
	// +optional
	Flow string `json:"flow,omitempty"`
	// The ip protocol name (tcp, udp, icmp or -1)
	// +optional
	IpProtocol string `json:"ipProtocol,omitempty"`
	// The ip range of the security group rule (deprecated, use ipRanges)
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// The list of ip ranges of the security group rule
	// +optional
	IpRanges []string `json:"ipRanges,omitempty"`
	// The beginning of the port range
	// +optional
	FromPortRange int32 `json:"fromPortRange,omitempty"`
	// The end of the port range
	// +optional
	ToPortRange int32 `json:"toPortRange,omitempty"`
	// The security group rule id
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

func (sgr *OscSecurityGroupRule) GetIpRanges() []string {
	if len(sgr.IpRanges) > 0 {
		return sgr.IpRanges
	}
	if sgr.IpRange != "" {
		return []string{sgr.IpRange}
	}
	return nil
}

func (sgr *OscSecurityGroupRule) GetFromPortRange() int32 {
	if sgr.IpProtocol == "-1" {
		return -1
	}
	return sgr.FromPortRange
}

func (sgr *OscSecurityGroupRule) GetToPortRange() int32 {
	if sgr.IpProtocol == "-1" {
		return -1
	}
	return sgr.ToPortRange
}

// OscClusterInitializationStatus provides observations of the OscCluster initialization process.
// +kubebuilder:validation:MinProperties=1
type OscClusterInitializationStatus struct {
	// provisioned is true when the infrastructure provider reports that the Cluster's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Cluster provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

type OscClusterResources struct {
	Net             map[string]string `json:"net,omitempty"`
	NetPeering      map[string]string `json:"netPeering,omitempty"`
	Subnet          map[string]string `json:"subnet,omitempty"`
	InternetService map[string]string `json:"internetService,omitempty"`
	NetAccessPoint  map[string]string `json:"netAccessPoint,omitempty"`
	SecurityGroup   map[string]string `json:"securityGroup,omitempty"`
	NatService      map[string]string `json:"natService,omitempty"`
	Bastion         map[string]string `json:"bastion,omitempty"`
	PublicIPs       map[string]string `json:"publicIps,omitempty"`
}

// OscClusterStatus defines the observed state of OscCluster
type OscClusterStatus struct {
	Initialization       OscClusterInitializationStatus `json:"initialization,omitempty,omitzero"`
	Resources            OscClusterResources            `json:"resources,omitempty"`
	ReconcilerGeneration OscReconcilerGeneration        `json:"reconcilerGeneration,omitempty"`
	FailureDomains       []clusterv1.FailureDomain      `json:"failureDomains,omitempty"`
	Conditions           clusterv1.Conditions           `json:"conditions,omitempty"`
	VmState              *osc.VmState                   `json:"vmState,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=oscclusters,scope=Namespaced,categories=cluster-api

// OscCluster is the Schema for the oscclusters API
type OscCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OscClusterSpec   `json:"spec,omitempty"`
	Status OscClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OscClusterList contains a list of OscCluster
type OscClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OscCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OscCluster{}, &OscClusterList{})
}
