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
	"slices"
	"strings"
)

type OscRole string

const (
	RoleControlPlane OscRole = "controlplane"
	RoleWorker       OscRole = "worker"
	RoleLoadBalancer OscRole = "loadbalancer"
	RoleBastion      OscRole = "bastion"
	RoleNat          OscRole = "nat"
)

type OscNode struct {
	Vm          OscVm       `json:"vm,omitempty"`
	Image       OscImage    `json:"image,omitempty"`
	Volumes     []OscVolume `json:"volumes,omitempty"`
	KeyPair     OscKeypair  `json:"keypair,omitempty"`
	ClusterName string      `json:"clusterName,omitempty"`
}

type OscNetwork struct {
	// The Load Balancer configuration
	// +optional
	LoadBalancer OscLoadBalancer `json:"loadBalancer,omitempty"`
	// The Net configuration
	// +optional
	Net OscNet `json:"net,omitempty"`
	// List of subnet to spread controlPlane nodes
	ControlPlaneSubnets []string `json:"controlPlaneSubnets,omitempty"`
	// The Subnet configuration
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
	// The Route Table configuration
	// +optional
	RouteTables    []OscRouteTable    `json:"routeTables,omitempty"`
	SecurityGroups []OscSecurityGroup `json:"securityGroups,omitempty"`
	// The Public Ip configuration
	// +optional
	PublicIps []*OscPublicIp `json:"publicIps,omitempty"`
	// The name of the cluster (unused)
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The image configuration
	// +optional
	Image OscImage `json:"image,omitempty"`
	// The bastion configuration
	// + optional
	Bastion OscBastion `json:"bastion,omitempty"`
	// The subregion name
	SubregionName string `json:"subregionName,omitempty"`
	// Add SecurityGroup Rule after the cluster is created
	// + optional
	ExtraSecurityGroupRule bool `json:"extraSecurityGroupRule,omitempty"`
}

type OscLoadBalancer struct {
	// The Load Balancer unique name
	// +optional
	LoadBalancerName string `json:"loadbalancername,omitempty"`
	// The Load Balancer Type internet-facing or internal
	// +optional
	LoadBalancerType string `json:"loadbalancertype,omitempty"`
	// The subnet tag name associate with a Subnet
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The security group tag name associate with a security group
	// +optional
	SecurityGroupName string `json:"securitygroupname,omitempty"`
	// The Listener cofiguration of the loadBalancer
	// +optional
	Listener OscLoadBalancerListener `json:"listener,omitempty"`
	// The healthCheck configuration  of the Load Balancer
	// +optional
	HealthCheck OscLoadBalancerHealthCheck `json:"healthCheck,omitempty"`
	ClusterName string                     `json:"clusterName,omitempty"`
}

type OscLoadBalancerListener struct {
	// The port on which the backend vm will listen
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
	// The Id of the Net to reuise (if useExisting is set)
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
	// Reuse an existing network defined by resourceId ?
	// +optional
	UseExisting bool `json:"useExisting,omitempty"`
}

func (o *OscNet) IsZero() bool {
	return o.IpRange == "" && o.ResourceId == ""
}

var DefaultNet = OscNet{
	IpRange: "10.0.0.0/16",
}

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
	// The id of the Subnet to reuse (if net.useExisting is set)
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
	// The name of the Subnet to which the Nat Service will be attached
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The name of the Subregion to which the Nat Service will be attached, unless a subnet has been defined.
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
	// The subnet tag name associate with a Subnet
	Subnets       []string `json:"subnets,omitempty"`
	Role          OscRole  `json:"role,omitempty"`
	SubregionName string   `json:"subregionName,omitempty"`
	// The Route configuration
	// +optional
	Routes []OscRoute `json:"routes,omitempty"`
	// The Route Table Id response
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
	// When reusing network, the id of an existing securityGroup to use.
	ResourceId string `json:"resourceId,omitempty"`
	Tag        string `json:"tag,omitempty"`

	// The roles the securityGroup applies to.
	Roles []OscRole `json:"roles,omitempty"`
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

type OscPublicIp struct {
	// The tag name associate with the Public Ip
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip Id response
	// +optional
	ResourceId  string `json:"resourceId,omitempty"`
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
	// The ip range of the security group rule (deprecated)
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

// Map between resourceId and resourceName (tag Name with cluster UID)
type OscResourceReference struct {
	ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

type OscNetworkResource struct {
	// Map between LoadbalancerId  and LoadbalancerName (Load Balancer tag Name with cluster UID)
	LoadbalancerRef OscResourceReference `json:"LoadbalancerRef,omitempty"`
	// Map between NetId  and NetName (Net tag Name with cluster UID)
	NetRef OscResourceReference `json:"netref,omitempty"`
	// Map between SubnetId  and SubnetName (Subnet tag Name with cluster UID)
	SubnetRef OscResourceReference `json:"subnetref,omitempty"`
	// Map between InternetServiceId  and InternetServiceName (Internet Service tag Name with cluster UID)
	InternetServiceRef OscResourceReference `json:"internetserviceref,omitempty"`
	// Map between RouteTablesId  and RouteTablesName (Route Tables tag Name with cluster UID)
	RouteTablesRef OscResourceReference `json:"routetableref,omitempty"`
	// Map between LinkRouteTableId and RouteTablesName (Route Table tag Name with cluster UID)
	LinkRouteTableRef map[string][]string `json:"linkroutetableref,omitempty"`
	// Map between RouteId  and RouteName (Route tag Name with cluster UID)
	RouteRef OscResourceReference `json:"routeref,omitempty"`
	// Map between SecurityGroupId  and SecurityGroupName (Security Group tag Name with cluster UID)
	SecurityGroupsRef OscResourceReference `json:"securitygroupref,omitempty"`
	// Map between SecurityGroupRuleId  and SecurityGroupName (Security Group Rule tag Name with cluster UID)
	SecurityGroupRuleRef OscResourceReference `json:"securitygroupruleref,omitempty"`
	// Map between PublicIpId  and PublicIpName (Public IP tag Name with cluster UID)
	PublicIpRef OscResourceReference `json:"publicipref,omitempty"`
	// Map between NatServiceId  and NatServiceName (Nat Service tag Name with cluster UID)
	NatServiceRef OscResourceReference `json:"natref,omitempty"`
	// Map between InstanceId  and BastionName (Bastion tag Name with cluster UID)
	BastionRef OscResourceReference `json:"bastionref,omitempty"`
	// Map between LinkPublicIpId  and PublicIpName (Public IP tag Name with cluster UID)
	LinkPublicIpRef OscResourceReference `json:"linkPublicIpRef,omitempty"`
}

type OscClusterResources struct {
	Net             map[string]string `json:"net,omitempty"`
	Subnet          map[string]string `json:"subnet,omitempty"`
	InternetService map[string]string `json:"internetService,omitempty"`
	SecurityGroup   map[string]string `json:"securityGroup,omitempty"`
	NatService      map[string]string `json:"natService,omitempty"`
	Bastion         map[string]string `json:"bastion,omitempty"`
	PublicIPs       map[string]string `json:"publicIps,omitempty"`
}

type Reconciler string

const (
	ReconcilerBastion         Reconciler = "bastion"
	ReconcilerNet             Reconciler = "net"
	ReconcilerSubnet          Reconciler = "subnet"
	ReconcilerInternetService Reconciler = "internetService"
	ReconcilerNatService      Reconciler = "natService"
	ReconcilerRouteTable      Reconciler = "routeTable"
	ReconcilerSecurityGroup   Reconciler = "securityGroup"
	ReconcilerLoadbalancer    Reconciler = "loadbalancer"

	ReconcilerVm Reconciler = "vm"
)

type OscReconcilerGeneration map[Reconciler]int64

type OscNodeResource struct {
	VolumeRef       OscResourceReference `json:"volumeRef,omitempty"`
	ImageRef        OscResourceReference `json:"imageRef,omitempty"`
	KeypairRef      OscResourceReference `json:"keypairRef,omitempty"`
	VmRef           OscResourceReference `json:"vmRef,omitempty"`
	LinkPublicIpRef OscResourceReference `json:"linkPublicIpRef,omitempty"`
	PublicIpIdRef   OscResourceReference `json:"publicIpIdRef,omitempty"`
}

type OscMachineResources struct {
	Vm      map[string]string `json:"vm,omitempty"`
	Image   map[string]string `json:"image,omitempty"`
	Volumes map[string]string `json:"volumes,omitempty"`
}

type OscImage struct {
	Name       string `json:"name,omitempty"`
	AccountId  string `json:"accountId,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
}

type OscVolume struct {
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Required
	Device string `json:"device"`
	Iops   int32  `json:"iops,omitempty"`
	// +kubebuilder:validation:Required
	Size int32 `json:"size,omitempty"`
	// Deprecated
	SubregionName string `json:"subregionName,omitempty"`
	VolumeType    string `json:"volumeType,omitempty"`
	ResourceId    string `json:"resourceId,omitempty"`
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
	// +kubebuilder:validation:Required
	KeypairName        string                    `json:"keypairName,omitempty"`
	VmType             string                    `json:"vmType,omitempty"`
	VolumeName         string                    `json:"volumeName,omitempty"`
	VolumeDeviceName   string                    `json:"volumeDeviceName,omitempty"`
	DeviceName         string                    `json:"deviceName,omitempty"`
	SubnetName         string                    `json:"subnetName,omitempty"`
	RootDisk           OscRootDisk               `json:"rootDisk,omitempty"`
	LoadBalancerName   string                    `json:"loadBalancerName,omitempty"`
	PublicIpName       string                    `json:"publicIpName,omitempty"`
	PublicIp           bool                      `json:"publicIp,omitempty"`
	SubregionName      string                    `json:"subregionName,omitempty"`
	PrivateIps         []OscPrivateIpElement     `json:"privateIps,omitempty"`
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	ResourceId         string                    `json:"resourceId,omitempty"`
	Role               OscRole                   `json:"role,omitempty"`
	ClusterName        string                    `json:"clusterName,omitempty"`
	Replica            int32                     `json:"replica,omitempty"`
	Tags               map[string]string         `json:"tags,omitempty"`
}

func (vm *OscVm) GetRole() OscRole {
	if vm.Role != "" {
		return vm.Role
	}
	return RoleWorker
}

type OscBastion struct {
	Name           string      `json:"name,omitempty"`
	ImageId        string      `json:"imageId,omitempty"`
	ImageName      string      `json:"imageName,omitempty"`
	ImageAccountId string      `json:"imageAccountId,omitempty"`
	KeypairName    string      `json:"keypairName,omitempty"`
	VmType         string      `json:"vmType,omitempty"`
	DeviceName     string      `json:"deviceName,omitempty"`
	SubnetName     string      `json:"subnetName,omitempty"`
	RootDisk       OscRootDisk `json:"rootDisk,omitempty"`
	// unused
	PublicIpName string `json:"publicIpName,omitempty"`
	// The ID of an existing public IP to use for this VM.
	// +optional
	PublicIpId         string                    `json:"PublicIpId,omitempty"`
	SubregionName      string                    `json:"subregionName,omitempty"`
	PrivateIps         []OscPrivateIpElement     `json:"privateIps,omitempty"`
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	ResourceId         string                    `json:"resourceId,omitempty"`
	ClusterName        string                    `json:"clusterName,omitempty"`
	Enable             bool                      `json:"enable,omitempty"`
}

type OscRootDisk struct {
	RootDiskIops int32  `json:"rootDiskIops,omitempty"`
	RootDiskSize int32  `json:"rootDiskSize,omitempty"`
	RootDiskType string `json:"rootDiskType,omitempty"`
}

type VmState string

var (
	VmStatePending      = VmState("pending")
	VmStateRunning      = VmState("running")
	VmStateShuttingDown = VmState("shutting-down")
	VmStateTerminated   = VmState("terminated")
	VmStateStopping     = VmState("stopping")
	VmStateStopped      = VmState("stopped")

	DefaultClusterName string = "cluster-api"

	DefaultKeypairName string = "cluster-api-keypair"

	DefaultVmName          string = "cluster-api-vm"
	DefaultVmSubregionName string = "eu-west-2a"

	DefaultVmKeypairName string = "cluster-api"
	DefaultVmType        string = "tinav3.c4r8p1"

	DefaultVmBastionImageId       string = "ami-bb490c7e"
	DefaultVmBastionKeypairName   string = "cluster-api"
	DefaultVmBastionSubregionName string = "eu-west-2a"

	DefaultVmKwName      string = "cluster-api-vm-kw"
	DefaultVmKwType      string = "tinav3.c4r8p1"
	DefaultVmKcpName     string = "cluster-api-vm-kcp"
	DefaultVmKcpType     string = "tinav3.c4r8p1"
	DefaultVmBastionName string = "cluster-api-vm-bastion"
	DefaultVmBastionType string = "tinav3.c2r2p1"

	DefaultRootDiskKwType      string = "io1"
	DefaultRootDiskKwSize      int32  = 60
	DefaultRootDiskKwIops      int32  = 1500
	DefaultRootDiskKcpType     string = "io1"
	DefaultRootDiskKcpSize     int32  = 60
	DefaultRootDiskKcpIops     int32  = 1500
	DefaultRootDiskBastionType string = "io1"
	DefaultRootDiskBastionSize int32  = 15
	DefaultRootDiskBastionIops int32  = 1000

	DefaultSubregionName        string = "eu-west-2a"
	DefaultLoadBalancerName     string = "OscClusterApi-1"
	DefaultLoadBalancerType     string = "internet-facing"
	DefaultBackendPort          int32  = 6443
	DefaultBackendProtocol      string = "TCP"
	DefaultLoadBalancerPort     int32  = 6443
	DefaultLoadBalancerProtocol string = "TCP"
	DefaultCheckInterval        int32  = 5
	DefaultHealthyThreshold     int32  = 5
	DefaultUnhealthyThreshold   int32  = 2
	DefaultTimeout              int32  = 5
	DefaultProtocol             string = "TCP"
	DefaultPort                 int32  = 6443
)

// SetDefaultValue set the vm default values
func (vm *OscVm) SetDefaultValue() {
	if vm.Role == RoleControlPlane {
		if vm.VmType == "" {
			vm.VmType = DefaultVmKcpType
		}

		if vm.RootDisk.RootDiskType == "" {
			vm.RootDisk.RootDiskType = DefaultRootDiskKcpType
		}
		if vm.RootDisk.RootDiskIops == 0 && vm.RootDisk.RootDiskType == "io1" {
			vm.RootDisk.RootDiskIops = DefaultRootDiskKcpIops
		}
		if vm.RootDisk.RootDiskSize == 0 {
			vm.RootDisk.RootDiskSize = DefaultRootDiskKcpSize
		}
	} else {
		if vm.VmType == "" {
			vm.VmType = DefaultVmKwType
		}

		if vm.RootDisk.RootDiskType == "" {
			vm.RootDisk.RootDiskType = DefaultRootDiskKwType
		}
		if vm.RootDisk.RootDiskIops == 0 && vm.RootDisk.RootDiskType == "io1" {
			vm.RootDisk.RootDiskIops = DefaultRootDiskKwIops
		}

		if vm.RootDisk.RootDiskSize == 0 {
			vm.RootDisk.RootDiskSize = DefaultRootDiskKwSize
		}
	}
	if vm.KeypairName == "" {
		vm.KeypairName = DefaultVmKeypairName
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
		if bastion.RootDisk.RootDiskIops == 0 && bastion.RootDisk.RootDiskType == "io1" {
			bastion.RootDisk.RootDiskIops = DefaultRootDiskBastionIops
		}
		if bastion.KeypairName == "" {
			bastion.KeypairName = DefaultVmBastionKeypairName
		}
	}
}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	if lb.LoadBalancerType == "" {
		lb.LoadBalancerType = DefaultLoadBalancerType
	}
	if lb.Listener.BackendPort == 0 {
		lb.Listener.BackendPort = DefaultBackendPort
	}
	if lb.Listener.BackendProtocol == "" {
		lb.Listener.BackendProtocol = DefaultBackendProtocol
	}
	if lb.Listener.LoadBalancerPort == 0 {
		lb.Listener.LoadBalancerPort = DefaultLoadBalancerPort
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
		lb.HealthCheck.Protocol = DefaultProtocol
	}
	if lb.HealthCheck.Port == 0 {
		lb.HealthCheck.Port = DefaultPort
	}
}
