/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package v1beta1

import (
	"slices"
	"strings"
)

type OscRole string

const (
	RoleControlPlane    OscRole = "controlplane"
	RoleWorker          OscRole = "worker"
	RoleLoadBalancer    OscRole = "loadbalancer"
	RoleBastion         OscRole = "bastion"
	RoleNat             OscRole = "nat"
	RoleService         OscRole = "service"
	RoleInternalService OscRole = "service.internal"
)

type OscNode struct {
	Vm      OscVm       `json:"vm,omitempty"`
	Image   OscImage    `json:"image,omitempty"`
	Volumes []OscVolume `json:"volumes,omitempty"`
	// deprecated, use vm.keypairName
	KeyPair OscKeypair `json:"keypair,omitempty"`
	// unused
	ClusterName string `json:"clusterName,omitempty"`
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

type OscNetwork struct {
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
	// List of subnet to spread controlPlane nodes (deprecated, add controlplane role to subnets)
	// +optional
	ControlPlaneSubnets []string `json:"controlPlaneSubnets,omitempty"`
	// The Subnets configuration
	// +optional
	Subnets []OscSubnet `json:"subnets,omitempty"`
	// The Internet Service configuration
	// +optional
	InternetService OscInternetService `json:"internetService,omitempty"`
	// The Nat Service configuration
	// +optional
	NatService OscNatService `json:"natService,omitempty"`
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
	// The Public Ip configuration (unused)
	// +optional
	PublicIps []*OscPublicIp `json:"publicIps,omitempty"`
	// The name of the cluster (unused)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The image configuration (unused)
	// +optional
	Image OscImage `json:"image,omitempty"`
	// The bastion configuration
	// + optional
	Bastion OscBastion `json:"bastion,omitempty"`
	// The default subregion name (deprecated, use subregions)
	SubregionName string `json:"subregionName,omitempty"`
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
	// The subnet name where to add the load balancer (deprecated, add loadbalancer role to a subnet)
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The security group name for the load-balancer (deprecated, add loadbalancer role to a security group)
	// +optional
	SecurityGroupName string `json:"securitygroupname,omitempty"`
	// The Listener configuration of the loadBalancer
	// +optional
	Listener OscLoadBalancerListener `json:"listener,omitempty"`
	// The healthCheck configuration of the Load Balancer
	// +optional
	HealthCheck OscLoadBalancerHealthCheck `json:"healthCheck,omitempty"`
	// unused
	ClusterName string `json:"clusterName,omitempty"`
}

type OscLoadBalancerListener struct {
	// The port on which the backend VMs will listen
	// +optional
	BackendPort int32 `json:"backendport,omitempty"`
	// The protocol ('HTTP'|'TCP') to route the traffic to the backend vm
	// +optional
	BackendProtocol string `json:"backendprotocol,omitempty"`
	// The port on which the loadbalancer will listen
	// +optional
	LoadBalancerPort int32 `json:"loadbalancerport,omitempty"`
	// the routing protocol ('HTTP'|'TCP')
	// +optional
	LoadBalancerProtocol string `json:"loadbalancerprotocol,omitempty"`
}

type OscLoadBalancerHealthCheck struct {
	// the time in second between two pings
	// +optional
	CheckInterval int32 `json:"checkinterval,omitempty"`
	// the consecutive number of pings which are successful to consider the vm healthy
	// +optional
	HealthyThreshold int32 `json:"healthythreshold,omitempty"`
	// the HealthCheck port number
	// +optional
	Port int32 `json:"port,omitempty"`
	// The HealthCheck protocol ('HTTP'|'TCP')
	// +optional
	Protocol string `json:"protocol,omitempty"`
	// the Timeout to consider VM unhealthy
	// +optional
	Timeout int32 `json:"timeout,omitempty"`
	// the consecutive number of pings which are failed to consider the vm unhealthy
	// +optional
	UnhealthyThreshold int32 `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
	// the network name
	// +optional
	Name string `json:"name,omitempty"`
	// the ip range in CIDR notation of the Net
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// the name of the cluster (unused)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
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

type OscInternetService struct {
	// The name of the Internet service
	// +optional
	Name string `json:"name,omitempty"`
	// the name of the cluster (unused)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// the Internet Service resource id (unused)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscSubnet struct {
	// The name of the Subnet
	// +optional
	Name string `json:"name,omitempty"`
	// The role of the Subnet (controlplane, worker, loadbalancer, bastion or nat)
	// +optional
	Roles []OscRole `json:"roles,omitempty"`
	// the Ip range in CIDR notation of the Subnet
	// +optional
	IpSubnetRange string `json:"ipSubnetRange,omitempty"`
	// The subregion name of the Subnet
	// +optional
	SubregionName string `json:"subregionName,omitempty"`
	// The id of the Subnet to reuse (if useExisting.net is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscNatService struct {
	// The name of the Nat Service
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip name (unused)
	// +optional
	PublicIpName string `json:"publicipname,omitempty"`
	// The name of the Subnet to which the Nat Service will be attached (deprecated, add nat role to subnets)
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The name of the Subregion to which the Nat Service will be attached, unless a subnet has been defined (unused)
	// +optional
	SubregionName string `json:"subregionName,omitempty"`
	// The name of the cluster (unused)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The resource id (unused)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscRouteTable struct {
	// The tag name associate with the Route Table
	// +optional
	Name string `json:"name,omitempty"`
	// The subnet tag name associate with a Subnet (deprecated, use roles)
	// +optional
	Subnets []string `json:"subnets,omitempty"`
	// The role for this route table
	// +optional
	Role OscRole `json:"role,omitempty"`
	// The subregion for this route table
	// +optional
	SubregionName string `json:"subregionName,omitempty"`
	// The Route configuration
	// +optional
	Routes []OscRoute `json:"routes,omitempty"`
	// The resource id (unused)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
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

type OscPublicIp struct {
	// The tag name associate with the Public Ip (unused)
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip Id response (unused)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
	// unused
	ClusterName string `json:"clusterName,omitempty"`
}

type OscRoute struct {
	// The tag name associate with the Route
	// +optional
	Name string `json:"name,omitempty"`
	// The tag name associate with the target resource type
	// +optional
	TargetName string `json:"targetName,omitempty"`
	// The target resource type which can be Internet Service (gateway) or Nat Service (nat-service)
	// +optional
	TargetType string `json:"targetType,omitempty"`
	// the destination match Ip range with CIDR notation
	// +optional
	Destination string `json:"destination,omitempty"`
	// The Route Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscPrivateIpElement struct {
	Name      string `json:"name,omitempty"`
	PrivateIp string `json:"privateIp,omitempty"`
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

// Map between resourceId and resourceName (tag Name with cluster UID)
type OscResourceReference struct {
	ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

type OscNetworkResource struct {
	// Map between LoadbalancerId and LoadbalancerName (not set anymore)
	LoadbalancerRef OscResourceReference `json:"LoadbalancerRef,omitempty"`
	// Map between NetId  and NetName (not set anymore)
	NetRef OscResourceReference `json:"netref,omitempty"`
	// Map between SubnetId  and SubnetName (not set anymore)
	SubnetRef OscResourceReference `json:"subnetref,omitempty"`
	// Map between InternetServiceId  and InternetServiceName (not set anymore)
	InternetServiceRef OscResourceReference `json:"internetserviceref,omitempty"`
	// Map between RouteTablesId  and RouteTablesName (not set anymore)
	RouteTablesRef OscResourceReference `json:"routetableref,omitempty"`
	// Map between LinkRouteTableId and RouteTablesName (not set anymore)
	LinkRouteTableRef map[string][]string `json:"linkroutetableref,omitempty"`
	// Map between RouteId  and RouteName (not set anymore)
	RouteRef OscResourceReference `json:"routeref,omitempty"`
	// Map between SecurityGroupId  and SecurityGroupName (not set anymore)
	SecurityGroupsRef OscResourceReference `json:"securitygroupref,omitempty"`
	// Map between SecurityGroupRuleId  and SecurityGroupName (not set anymore)
	SecurityGroupRuleRef OscResourceReference `json:"securitygroupruleref,omitempty"`
	// Map between PublicIpId  and PublicIpName (not set anymore)
	PublicIpRef OscResourceReference `json:"publicipref,omitempty"`
	// Map between NatServiceId  and NatServiceName (not set anymore)
	NatServiceRef OscResourceReference `json:"natref,omitempty"`
	// Map between InstanceId  and BastionName (not set anymore)
	BastionRef OscResourceReference `json:"bastionref,omitempty"`
	// Map between LinkPublicIpId  and PublicIpName (not set anymore)
	LinkPublicIpRef OscResourceReference `json:"linkPublicIpRef,omitempty"`
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

type Reconciler string

const (
	ReconcilerBastion          Reconciler = "bastion"
	ReconcilerNet              Reconciler = "net"
	ReconcilerNetPeering       Reconciler = "netPeering"
	ReconcilerNetPeeringRoutes Reconciler = "netPeering/routes"
	ReconcilerSubnet           Reconciler = "subnet"
	ReconcilerInternetService  Reconciler = "internetService"
	ReconcilerNetAccessPoint   Reconciler = "netAccessPoint"
	ReconcilerNatService       Reconciler = "natService"
	ReconcilerRouteTable       Reconciler = "routeTable"
	ReconcilerSecurityGroup    Reconciler = "securityGroup"
	ReconcilerLoadbalancer     Reconciler = "loadbalancer"

	ReconcilerVm Reconciler = "vm"
)

type OscReconcilerGeneration map[Reconciler]int64

type OscNodeResource struct {
	// Volume references (not set anymore)
	VolumeRef OscResourceReference `json:"volumeRef,omitempty"`
	// Image references (not set anymore)
	ImageRef OscResourceReference `json:"imageRef,omitempty"`
	// Keypair references (not set anymore)
	KeypairRef OscResourceReference `json:"keypairRef,omitempty"`
	// Vm references (not set anymore)
	VmRef OscResourceReference `json:"vmRef,omitempty"`
	// LinkPublicIp references (not set anymore)
	LinkPublicIpRef OscResourceReference `json:"linkPublicIpRef,omitempty"`
	// PublicIp references (not set anymore)
	PublicIpIdRef OscResourceReference `json:"publicIpIdRef,omitempty"`
}

type OscMachineResources struct {
	Vm        map[string]string `json:"vm,omitempty"`
	Image     map[string]string `json:"image,omitempty"`
	Volumes   map[string]string `json:"volumes,omitempty"`
	PublicIPs map[string]string `json:"publicIps,omitempty"`
}

type OscImage struct {
	// The image name.
	Name string `json:"name,omitempty"`
	// The image account owner ID.
	AccountId string `json:"accountId,omitempty"`
	// Use an "Outscale Opensource" image
	OutscaleOpenSource bool `json:"outscaleOpenSource,omitempty"`
	// unused
	ResourceId string `json:"resourceId,omitempty"`
}

type OscVolume struct {
	// The volume name.
	Name string `json:"name,omitempty"`
	// The volume device (/dev/xvdX)
	// +kubebuilder:validation:Required
	Device string `json:"device"`
	// The volume iops (io1 volumes only)
	Iops int32 `json:"iops,omitempty"`
	// The volume size in gibibytes (GiB)
	Size int32 `json:"size,omitempty"`
	// (unused)
	SubregionName string `json:"subregionName,omitempty"`
	// The volume type (io1, gp2 or standard)
	VolumeType string `json:"volumeType,omitempty"`
	// (unused)
	ResourceId string `json:"resourceId,omitempty"`
	// The id of a snapshot to use as a volume source.
	FromSnapshot string `json:"fromSnapshot,omitempty"`
}

type OscKeypair struct {
	// Deprecated
	Name string `json:"name,omitempty"`
	// Deprecated
	PublicKey string `json:"publicKey,omitempty"`
	// Deprecated
	ResourceId string `json:"resourceId,omitempty"`
	// Deprecated
	ClusterName string `json:"clusterName,omitempty"`
	// Deprecated
	DeleteKeypair bool `json:"deleteKeypair,omitempty"`
}

type OscVm struct {
	Name    string `json:"name,omitempty"`
	ImageId string `json:"imageId,omitempty"`
	// The keypair name
	// +kubebuilder:validation:Required
	KeypairName string `json:"keypairName,omitempty"`
	// The type of vm (tinav6.c4r8p1 by default)
	// +optional
	VmType string `json:"vmType,omitempty"`
	// unused
	VolumeName string `json:"volumeName,omitempty"`
	// unused
	VolumeDeviceName string `json:"volumeDeviceName,omitempty"`
	// unused
	DeviceName string `json:"deviceName,omitempty"`
	// The subnet of the node (deprecated, use controlplane and/or worker roles on subnets)
	// +optional
	SubnetName string      `json:"subnetName,omitempty"`
	RootDisk   OscRootDisk `json:"rootDisk,omitempty"`
	// unused
	LoadBalancerName string `json:"loadBalancerName,omitempty"`
	// unused
	PublicIpName string `json:"publicIpName,omitempty"`
	// If set, a public IP will be configured.
	// +optional
	PublicIp bool `json:"publicIp,omitempty"`
	// The name of the pool from which public IPs will be picked.
	// +optional
	PublicIpPool  string                `json:"publicIpPool,omitempty"`
	SubregionName string                `json:"subregionName,omitempty"`
	PrivateIps    []OscPrivateIpElement `json:"privateIps,omitempty"`
	// The list of security groups to use (deprecated, use controlplane and/or worker roles on security groups)
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	// The resource id of the vm (not set anymore)
	ResourceId string `json:"resourceId,omitempty"`
	// The node role (controlplane or worker, worker by default).
	// +optional
	Role OscRole `json:"role,omitempty"`
	// unused
	ClusterName string `json:"clusterName,omitempty"`
	// unused
	Replica int32 `json:"replica,omitempty"`
	// Tags to add to the VM.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

func (vm *OscVm) GetRole() OscRole {
	if vm.Role != "" {
		return vm.Role
	}
	return RoleWorker
}

type OscBastion struct {
	Name           string `json:"name,omitempty"`
	ImageId        string `json:"imageId,omitempty"`
	ImageName      string `json:"imageName,omitempty"`
	ImageAccountId string `json:"imageAccountId,omitempty"`
	KeypairName    string `json:"keypairName,omitempty"`
	// The type of VM (tinav6.c1r1p2 by default)
	// +optional
	VmType string `json:"vmType,omitempty"`
	// unused
	DeviceName string `json:"deviceName,omitempty"`
	// The subnet of the vm (deprecated use bastion role in subnets)
	SubnetName string      `json:"subnetName,omitempty"`
	RootDisk   OscRootDisk `json:"rootDisk,omitempty"`
	// unused
	PublicIpName string `json:"publicIpName,omitempty"`
	// The ID of an existing public IP to use for this VM.
	// +optional
	PublicIpId string `json:"PublicIpId,omitempty"`
	// unused
	SubregionName string                `json:"subregionName,omitempty"`
	PrivateIps    []OscPrivateIpElement `json:"privateIps,omitempty"`
	// The list of security groups (deprecated use bastion role in security groups)
	// +optional
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	// the vm id (deprecated, not set anymore)
	ResourceId string `json:"resourceId,omitempty"`
	// unused
	ClusterName string `json:"clusterName,omitempty"`
	Enable      bool   `json:"enable,omitempty"`
}

type OscRootDisk struct {
	// The root disk iops (io1 volumes only) (1500 by default)
	// +optional
	RootDiskIops int32 `json:"rootDiskIops,omitempty"`
	// The volume size in gibibytes (GiB) (60 by default)
	// +optional
	RootDiskSize int32 `json:"rootDiskSize,omitempty"`
	// The volume type (io1, gp2 or standard) (io1 by default)
	// +optional
	RootDiskType string `json:"rootDiskType,omitempty"`
}

type VmState string

const (
	VmStatePending      = VmState("pending")
	VmStateRunning      = VmState("running")
	VmStateShuttingDown = VmState("shutting-down")
	VmStateTerminated   = VmState("terminated")
	VmStateStopping     = VmState("stopping")
	VmStateStopped      = VmState("stopped")

	DefaultVmType       string = "tinav6.c4r8p1"
	DefaultRootDiskType string = "io1"
	DefaultRootDiskSize int32  = 60
	DefaultRootDiskIops int32  = 1500

	DefaultVmBastionType       string = "tinav6.c1r1p2"
	DefaultRootDiskBastionType string = "gp2"
	DefaultRootDiskBastionSize int32  = 15

	DefaultLoadBalancerType     string = "internet-facing"
	DefaultLoadBalancerProtocol string = "TCP"
	DefaultCheckInterval        int32  = 10
	DefaultHealthyThreshold     int32  = 2
	DefaultUnhealthyThreshold   int32  = 3
	DefaultTimeout              int32  = 10

	APIPort int32 = 6443
)

// SetDefaultValue set the vm default values
func (vm *OscVm) SetDefaultValue() {
	if vm.VmType == "" {
		vm.VmType = DefaultVmType
	}
	if vm.RootDisk.RootDiskType == "" {
		vm.RootDisk.RootDiskType = DefaultRootDiskType
	}
	if vm.RootDisk.RootDiskIops == 0 && vm.RootDisk.RootDiskType == "io1" {
		vm.RootDisk.RootDiskIops = DefaultRootDiskIops
	}
	if vm.RootDisk.RootDiskSize == 0 {
		vm.RootDisk.RootDiskSize = DefaultRootDiskSize
	}
}

// SetDefaultValue set the bastion default values
func (bastion *OscBastion) SetDefaultValue() {
	if bastion.Enable {
		if bastion.VmType == "" {
			bastion.VmType = DefaultVmBastionType
		}
		if bastion.RootDisk.RootDiskType == "" {
			bastion.RootDisk.RootDiskType = DefaultRootDiskBastionType
		}
		if bastion.RootDisk.RootDiskSize == 0 {
			bastion.RootDisk.RootDiskSize = DefaultRootDiskBastionSize
		}
	}
}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	if lb.LoadBalancerType == "" {
		lb.LoadBalancerType = DefaultLoadBalancerType
	}
	if lb.Listener.BackendPort == 0 {
		lb.Listener.BackendPort = APIPort
	}
	if lb.Listener.BackendProtocol == "" {
		lb.Listener.BackendProtocol = DefaultLoadBalancerProtocol
	}
	if lb.Listener.LoadBalancerPort == 0 {
		lb.Listener.LoadBalancerPort = APIPort
	}
	if lb.Listener.LoadBalancerProtocol == "" {
		lb.Listener.LoadBalancerProtocol = DefaultLoadBalancerProtocol
	}
	if lb.HealthCheck.CheckInterval == 0 {
		lb.HealthCheck.CheckInterval = DefaultCheckInterval
	}
	if lb.HealthCheck.HealthyThreshold == 0 {
		lb.HealthCheck.HealthyThreshold = DefaultHealthyThreshold
	}
	if lb.HealthCheck.UnhealthyThreshold == 0 {
		lb.HealthCheck.UnhealthyThreshold = DefaultUnhealthyThreshold
	}
	if lb.HealthCheck.Timeout == 0 {
		lb.HealthCheck.Timeout = DefaultTimeout
	}
	if lb.HealthCheck.Protocol == "" {
		lb.HealthCheck.Protocol = DefaultLoadBalancerProtocol
	}
	if lb.HealthCheck.Port == 0 {
		lb.HealthCheck.Port = APIPort
	}
}
