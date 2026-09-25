/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package v1beta2

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
)

// +kubebuilder:validation:Pattern:="(cloudgouv-)?(eu|us|ap)-(north|east|south|west|northeast|northwest|southeast|southwest)-[1-2][a-c]"
type OscSubRegion string

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

type OscCredentials struct {
	// Load credentials from this secret instead of the env.
	// +optional
	FromSecret string `json:"fromSecret,omitempty"`
	// Load credentials from this file instead of the env.
	// +optional
	FromFile string `json:"fromFile,omitempty"`
	// Name of profile stored in file (unused when using fromSecret, "default" by default).
	// +optional
	Profile string `json:"profile,omitempty"`
}

type OscReuse struct {
	// If set, net, subnets, internet service, nat services and route tables are externally managed
	// +optional
	Net bool `json:"net,omitempty"`
	// If set, security groups are externally managed.
	// +optional
	SecurityGroups bool `json:"securityGroups,omitempty"`
}

type OscDisable struct {
	// If set, net, subnets, internet service, nat services and route tables are externally managed
	// +optional
	Internet bool `json:"internet,omitempty"`
	// +optional
	Loadbalancer bool `json:"loadbalancer,omitempty"`
}

// +kubebuilder:validation:Enum:=internet-facing;internal
type OscLoadBalancerType string

const (
	LoadBalancerTypeInternetFacing OscLoadBalancerType = "internet-facing"
	LoadBalancerTypeInternal       OscLoadBalancerType = "internal"
)

type OscLoadBalancer struct {
	// The Load Balancer unique name
	// +required
	// +kubebuilder:validation:Pattern:="^[0-9A-Za-z][0-9A-Za-z-]{0,31}$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="field is immutable"
	Name string `json:"name,omitempty"`
	// The Load Balancer type (internet-facing or internal)
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="field is immutable"
	Type OscLoadBalancerType `json:"type,omitempty"`
	// The subnet name where to add the load balancer (deprecated, add loadbalancer role to a subnet)
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The security group name for the load-balancer (deprecated, add loadbalancer role to a security group)
	// +optional
	SecurityGroupName string `json:"securitygroupname,omitempty"`
	// The Listener configuration of the loadBalancer
	// +optional
	Listener OscLoadBalancerListener `json:"listener,omitempty,omitzero"`
	// The healthCheck configuration of the Load Balancer
	// +optional
	HealthCheck OscLoadBalancerHealthCheck `json:"healthCheck,omitempty,omitzero"`
}

// +kubebuilder:validation:Enum:=HTTP;HTTPS;TCP;SSL
type OscLoadBalancerProtocol string

const (
	OscLoadBalancerProtocolHTTP  OscLoadBalancerProtocol = "HTTP"
	OscLoadBalancerProtocolHTTPS OscLoadBalancerProtocol = "HTTPS"
	OscLoadBalancerProtocolTCP   OscLoadBalancerProtocol = "TCP"
	OscLoadBalancerProtocolSSL   OscLoadBalancerProtocol = "SSL"
)

type OscLoadBalancerListener struct {
	// The port on which the backend VMs will listen
	// +optional
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=65535
	BackendPort int32 `json:"backendport,omitempty"`
	// The protocol ('HTTP'|'TCP') to route the traffic to the backend vm
	// +optional
	BackendProtocol OscLoadBalancerProtocol `json:"backendprotocol,omitempty"`
	// The port on which the loadbalancer will listen
	// +optional
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=65535
	LoadBalancerPort int32 `json:"loadbalancerport,omitempty"`
	// the routing protocol ('HTTP'|'TCP')
	// +optional
	LoadBalancerProtocol OscLoadBalancerProtocol `json:"loadbalancerprotocol,omitempty"`
}

type OscLoadBalancerHealthCheck struct {
	// the interval in second between two pings
	// +optional
	// +kubebuilder:validation:Minimum:=5
	// +kubebuilder:validation:Maximum:=600
	CheckInterval int32 `json:"checkinterval,omitempty"`
	// the number of consecutive successful checks required for a VM to be considered healthy
	// +optional
	// +kubebuilder:validation:Minimum:=2
	// +kubebuilder:validation:Maximum:=10
	HealthyThreshold int32 `json:"healthythreshold,omitempty"`
	// the destination port for checks
	// +optional
	// +kubebuilder:validation:Minimum:=5
	// +kubebuilder:validation:Maximum:=600
	Port int32 `json:"port,omitempty"`
	// The check protocol ('HTTP'|'TCP')
	// +optional
	Protocol OscLoadBalancerProtocol `json:"protocol,omitempty"`
	// the timeout for a check
	// +optional
	// +kubebuilder:validation:Minimum:=2
	// +kubebuilder:validation:Maximum:=60
	Timeout int32 `json:"timeout,omitempty"`
	// the number of consecutive successful checks required for a VM to be considered unhealthy
	// +optional
	// +kubebuilder:validation:Minimum:=2
	// +kubebuilder:validation:Maximum:=10
	UnhealthyThreshold int32 `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
	// the network name
	// +optional
	Name string `json:"name,omitempty"`
	// the ip range in CIDR notation of the Net
	// +optional
	// +kubebuilder:validation:XValidation:rule="isCIDR(self)"
	IpRange string `json:"ipRange,omitempty"`
	// The Id of the Net to reuse (if useExisting.net is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

func (n OscNet) IsZero() bool {
	return n.IpRange == "" && n.ResourceId == ""
}

func (n OscNet) GetIpRanges() []string {
	return []string{n.IpRange}
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
	ManagementCredentials OscCredentials `json:"managementCredentials,omitempty,omitzero"`
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
	// +kubebuilder:validation:XValidation:rule="isCIDR(self)"
	IpRange string `json:"ipRange,omitempty"`
	// The subregion name of the Subnet
	// +optional
	SubregionName OscSubRegion `json:"subregionName,omitempty"`
	// The id of the Subnet to reuse (if useExisting.net is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscNatService struct {
	// The name of the Nat Service
	// +optional
	Name string `json:"name,omitempty"`
	// The name of the Subnet to which the Nat Service will be attached (deprecated, add nat role to subnets)
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The name of the Subregion to which the Nat Service will be attached, unless a subnet has been defined
	// +optional
	SubregionName OscSubRegion `json:"subregionName,omitempty"`
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
	SubregionName OscSubRegion `json:"subregionName,omitempty"`
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

// +kubebuilder:validation:Enum:=Inbound;Outbound
type Flow string

const (
	FlowInbound  Flow = "Inbound"
	FlowOutbound Flow = "Outbound"
)

// +kubebuilder:validation:Pattern:="^[a-z0-9-]+(/[0-9]{1,5}(-[0-9]{1,5})?)?( ?#.*)?"
type Port string

var rePort = regexp.MustCompile("^([a-z0-9-]+)(/([0-9]{1,5})(-([0-9]{1,5}))?)?( ?#.*)?")

func (p Port) Parse() (protocol string, fromPort, toPort int, err error) {
	ms := rePort.FindAllStringSubmatch(string(p), 1)
	if len(ms) == 0 {
		return "-1", -1, -1, errors.New("not a port definition")
	}
	protocol, from, to := ms[0][1], ms[0][3], ms[0][5]
	fromPort, toPort = -1, -1
	if from != "" {
		fromPort, _ = strconv.Atoi(from) // the regexp only matches numbers
	}
	if to != "" {
		toPort, _ = strconv.Atoi(to)
	}
	if fromPort != -1 && toPort == -1 {
		toPort = fromPort
	}
	return
}

func BuildPort(protocol string, fromPort, toPort int32) Port {
	b := strings.Builder{}
	b.WriteString(protocol)
	if fromPort != -1 {
		b.WriteString("/")
		b.WriteString(strconv.Itoa(int(fromPort)))
	}
	if toPort != fromPort && toPort != -1 {
		b.WriteString("-")
		b.WriteString(strconv.Itoa(int(toPort)))
	}
	return Port(b.String())
}

type OscSecurityGroupRule struct {
	// The tag name associate with the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The flow of the security group (Inbound or Outbound), default Inbound
	// +optional
	Flow Flow `json:"flow,omitempty"`
	// The list of ports to open (protocol, protocol/port or protocol/fromPort-toPort)
	Ports []Port `json:"ports"`
	// The list of ip ranges of the security group rule
	// +optional
	// +kubebuilder:validation:items:XValidation:rule="isCIDR(self)"
	IpRanges []string `json:"ipRanges"`
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

// +kubebuilder:validation:Enum:=bastion;net;netPeering;netPeering/routes;subnet;internetService;netAccessPoint;natService;routeTable;securityGroup;loadbalancer;vm;*
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
	ReconcilerKeypair          Reconciler = "keypair"

	ReconcilerVm Reconciler = "vm"

	ReconcilerAll Reconciler = "*"
)

type OscReconcilerGeneration map[Reconciler]int64

// +kubebuilder:validation:Enum:=onChange;always;random
type ReconciliationMode string

const (
	ReconciliationModeOnChange ReconciliationMode = "onChange"
	ReconciliationModeAlways   ReconciliationMode = "always"
	ReconciliationModeRandom   ReconciliationMode = "random"
)

type OscReconciliationRule struct {
	// The list of items this rule applies to (bastion, net, netPeering, netPeering/routes, subnet, internetService, netAccessPoint, natService, routeTable, securityGroup, loadbalancer, vm or * for all)
	AppliesTo []Reconciler `json:"appliesTo,omitempty"`
	// The mode of reconciliation: onChange (only when the spec change, default), always, random (onChange + randomPercent% chance)
	Mode ReconciliationMode `json:"mode,omitempty"`
	// The chance (in percent, 1-100) of a reconciliation happening when no change have been detected (when mode=random)
	// +optional
	ReconciliationChance int `json:"reconciliationChance,omitempty"`
}

type OscMachineResources struct {
	Vm        map[string]string `json:"vm,omitempty"`
	FGPU      map[string]string `json:"fGPU,omitempty"`
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
}

// +kubebuilder:validation:Enum:=io1;gp2;standard
type OscVolumeType = osc.VolumeType

type OscVolume struct {
	// The volume name.
	// +optional
	Name string `json:"name,omitempty"`
	// The volume device (/dev/xvdX)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern:="^(/dev/sda1|/dev/sd[a-z]{1}|/dev/xvd[a-z]{1})$"
	Device string `json:"device"`
	// The volume iops (io1 volumes only)
	// +optional
	Iops int32 `json:"iops,omitempty"`
	// The volume size in gibibytes (GiB)
	// +optional
	Size int32 `json:"size,omitempty"`
	// The volume type (io1, gp2 or standard)
	// +optional
	VolumeType osc.VolumeType `json:"volumeType,omitempty"`
	// The id of a snapshot to use as a volume source.
	// +optional
	FromSnapshot string `json:"fromSnapshot,omitempty"`
}

type OscKeypair struct {
	// Name of the keypair to create.
	// +kubebuilder:validation:MaxLength:=255
	// +required
	Name string `json:"name"`
	// Name of the secret where the private key will be stored, defaults to "<cluster name>-keypair".
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// Keep the keypair after cluster deletion ?
	// +optional
	KeepAfterDeletion bool `json:"keepAfterDeletion,omitempty"`
}

// +kubebuilder:validation:Enum:=leastNodes;random
type SubregionMode string

const (
	SubregionModeLeastNodes SubregionMode = "leastNodes"
	SubregionModeRandom     SubregionMode = "random"
)

type OscFGPU struct {
	// The fGPU model to add to the VM (e.g. nvidia-h100).
	// The fGPU will be released when the node VM is deleted.
	Model string `json:"model,omitempty"`
}

type OscVm struct {
	Name    string `json:"name,omitempty"`
	ImageId string `json:"imageId,omitempty"`
	// The keypair name.
	// +required
	KeypairName string `json:"keypairName,omitempty"`
	// The type of vm (tinav7.c4r8p1 by default)
	// +optional
	// +kubebuilder:validation:Pattern:=`^(tinav([3-9]|[1-9][0-9]).c[1-9][0-9]*r[1-9][0-9]*p[1-3]|inference7-(?:l40\.(?:medium|large)|h100\.(?:medium|large|xlarge|2xlarge)|h200\.(?:2xsmall|2xmedium|2xlarge|4xlarge|4xlargeA)))$`
	VmType string `json:"vmType,omitempty"`
	// The subnet of the node (deprecated, use controlplane and/or worker roles on subnets)
	// +optional
	SubnetName string      `json:"subnetName,omitempty"`
	RootDisk   OscRootDisk `json:"rootDisk,omitempty"`
	// If set, a public IP will be configured.
	// +optional
	PublicIp bool `json:"publicIp,omitempty"`
	// The name of the pool from which public IPs will be picked.
	// +optional
	PublicIpPool string `json:"publicIpPool,omitempty"`
	// The fGPU configuration for this VM.
	// +optional
	FGPU *OscFGPU `json:"fGPU,omitempty"`
	// The subregion where the machine needs to be placed (deprecated, use subregionNames).
	// +optional
	SubregionName OscSubRegion `json:"subregionName,omitempty"`
	// The way nodes will be allocated in subregions (leastNodes or random; by default, leastNodes).
	// +optional
	SubregionMode SubregionMode `json:"subregionMode,omitempty"`
	// The subregions where the machines needs to be placed. If empty, the subregions defined at cluster level will be used.
	// +optional
	SubregionNames []OscSubRegion        `json:"subregionNames,omitempty"`
	PrivateIps     []OscPrivateIpElement `json:"privateIps,omitempty"`
	// The list of security groups to use (deprecated, use controlplane and/or worker roles on security groups)
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	// The resource id of the vm (not set anymore)
	ResourceId string `json:"resourceId,omitempty"`
	// The node role (controlplane or worker), defaults to worker.
	// +optional
	Role OscRole `json:"role,omitempty"`
	// Tags to add to the VM.
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
	// VM placement constraints.
	// +optional
	Placement OscPlacement `json:"placement,omitempty"`
}

func (vm *OscVm) GetRole() OscRole {
	if vm.Role != "" {
		return vm.Role
	}
	return RoleWorker
}

func (vm *OscVm) GetSubregions() []string {
	if len(vm.SubregionNames) > 0 {
		return lo.Map(vm.SubregionNames, func(s OscSubRegion, _ int) string { return string(s) })
	}
	return []string{string(vm.SubregionName)}
}

type OscPlacement struct {
	// Try to put VMs with the same repulseServer value on different physical servers. For workers, set by default to the MachineDeployment name unless RepulseCluster is set.
	// Define to an empty string if you want to disable.
	// +optional
	RepulseServer *string `json:"repulseServer,omitempty"`
	// Try to put VMs with the same attractServer value on the same physical server.
	// +optional
	AttractServer string `json:"attractServer,omitempty"`
	// serverStrict makes repulseServer/attractServer mandatory. CreateVm will fail if VM placement is not possible.
	// +optional
	ServerStrict bool `json:"serverStrict,omitempty"`
	// Try to put VMs with the same repulseCluster value on different clusters. Not set by default.
	// +optional
	RepulseCluster string `json:"repulseCluster,omitempty"`
	// Try to put VMs with the same attractCluster value on the same cluster.
	// +optional
	AttractCluster string `json:"attractCluster,omitempty"`
	// clusterStrict makes repulseCluster/attractCluster mandatory. CreateVm will fail if VM placement is not possible.
	// +optional
	ClusterStrict bool `json:"clusterStrict,omitempty"`
}

type OscBastion struct {
	Name           string `json:"name,omitempty"`
	ImageId        string `json:"imageId,omitempty"`
	ImageName      string `json:"imageName,omitempty"`
	ImageAccountId string `json:"imageAccountId,omitempty"`
	KeypairName    string `json:"keypairName,omitempty"`
	// The type of VM (tinav7.c1r1p2 by default)
	// +optional
	VmType string `json:"vmType,omitempty"`
	// The subnet of the vm (deprecated use bastion role in subnets)
	SubnetName string      `json:"subnetName,omitempty"`
	RootDisk   OscRootDisk `json:"rootDisk,omitempty,omitzero"`
	// The ID of an existing public IP to use for this VM.
	// +optional
	PublicIpId string                `json:"PublicIpId,omitempty"`
	PrivateIps []OscPrivateIpElement `json:"privateIps,omitempty"`
	// The list of security groups (deprecated use bastion role in security groups)
	// +optional
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	// the vm id (deprecated, not set anymore)
	ResourceId string `json:"resourceId,omitempty"`
	Enable     bool   `json:"enable,omitempty"`
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
	RootDiskType osc.VolumeType `json:"rootDiskType,omitempty"`
}

type VmState string

const (
	DefaultVmType       string         = "tinav7.c4r8p1"
	DefaultRootDiskType osc.VolumeType = "io1"
	DefaultRootDiskSize int32          = 60
	DefaultRootDiskIops int32          = 1500

	DefaultVmBastionType       string         = "tinav7.c1r1p2"
	DefaultRootDiskBastionType osc.VolumeType = "gp2"
	DefaultRootDiskBastionSize int32          = 15

	DefaultLoadBalancerType     OscLoadBalancerType     = "internet-facing"
	DefaultLoadBalancerProtocol OscLoadBalancerProtocol = "TCP"
	DefaultCheckInterval        int32                   = 10
	DefaultHealthyThreshold     int32                   = 2
	DefaultUnhealthyThreshold   int32                   = 3
	DefaultTimeout              int32                   = 10

	APIPort    int32 = 6443
	APIPortStr       = "tcp/6443"
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
	if lb.Type == "" {
		lb.Type = DefaultLoadBalancerType
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
