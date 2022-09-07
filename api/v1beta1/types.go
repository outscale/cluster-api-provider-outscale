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
	"strings"
)

// OscNode is all included vm.
type OscNode struct {
	VM          OscVM        `json:"vm,omitempty"`
	Image       OscImage     `json:"image,omitempty"`
	Volumes     []*OscVolume `json:"volumes,omitempty"`
	KeyPair     OscKeypair   `json:"keypair,omitempty"`
	ClusterName string       `json:"clusterName,omitempty"`
}

// OscNetwork is the cluster.
type OscNetwork struct {
	// The Load Balancer configuration
	// +optional
	LoadBalancer OscLoadBalancer `json:"loadBalancer,omitempty"`
	// The Net configuration
	// +optional
	Net OscNet `json:"net,omitempty"`
	// The Subnet configuration
	// +optional
	Subnets []*OscSubnet `json:"subnets,omitempty"`
	// The Internet Service configuration
	// +optional
	InternetService OscInternetService `json:"internetService,omitempty"`
	// The Nat Service configuration
	// +optional
	NatService OscNatService `json:"natService,omitempty"`
	// The Route Table configuration
	// +optional
	RouteTables []*OscRouteTable `json:"routeTables,omitempty"`

	SecurityGroups []*OscSecurityGroup `json:"securityGroups,omitempty"`
	// The Public Ip configuration
	// +optional
	PublicIPS   []*OscPublicIP ` json:"publicIPs,omitempty" `
	ClusterName string         `json:"clusterName,omitempty"`
}

// OscLoadBalancer is the loadBalancer.
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

// OscLoadBalancerListener  is the loadBalancer listener.
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

// OscLoadBalancerHealthCheck is the loadBalancer HealthCheck.
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

// OscNet is the net.
type OscNet struct {
	// the tag name associate with the Net
	// +optional
	Name string `json:"name,omitempty"`
	// the net ip range with CIDR notation
	// +optional
	IPRange     string `json:"ipRange,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	// The Net Id response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscInternetService is the InternetService.
type OscInternetService struct {
	// The tag name associate with the Subnet
	// +optional
	Name        string `json:"name,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	// the Internet Service response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscSubnet is the subnet.
type OscSubnet struct {
	// The tag name associate with the Subnet
	// +optional
	Name string `json:"name,omitempty"`
	// Subnet Ip range with CIDR notation
	// +optional
	IPSubnetRange string `json:"ipSubnetRange,omitempty"`
	// The Subnet Id response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscNatService is the natService.
type OscNatService struct {
	// The tag name associate with the Nat Service
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip tag name associated with a Public Ip
	// +optional
	PublicIPName string `json:"publicipname,omitempty"`
	// The subnet tag name associate with a Subnet
	// +optional
	SubnetName  string `json:"subnetname,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	// The Nat Service Id response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscRouteTable is the routeTable.
type OscRouteTable struct {
	// The tag name associate with the Route Table
	// +optional
	Name string `json:"name,omitempty"`
	// The subnet tag name associate with a Subnet
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The Route configuration
	// +optional
	Routes []OscRoute `json:"routes,omitempty"`
	// The Route Table Id response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscSecurityGroup is the securityGroup.
type OscSecurityGroup struct {
	// The tag name associate with the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The description of the security group
	// +optional
	Description string `json:"description,omitempty"`
	// The Security Group Rules configuration
	// +optional
	SecurityGroupRules []OscSecurityGroupRule `json:"securityGroupRules,omitempty"`
	// The Security Group Id response
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscPublicIP is the publicIp.
type OscPublicIP struct {
	// The tag name associate with the Public Ip
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip Id response
	// +optional
	ResourceID  string `json:"resourceId,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
}

// OscRoute is the route.
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
	ResourceID string `json:"resourceId,omitempty"`
}

// OscPrivateIPElement is a privateIp element.
type OscPrivateIPElement struct {
	Name      string `json:"name,omitempty"`
	PrivateIP string `json:"privateIP,omitempty"`
}

// OscSecurityGroupElement is a securityGroup element.
type OscSecurityGroupElement struct {
	Name string `json:"name,omitempty"`
}

// OscSecurityGroupRule is a list of securotyGroupRule.
type OscSecurityGroupRule struct {
	// The tag name associate with the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The flow of the security group (inbound or outbound)
	// +optional
	Flow string `json:"flow,omitempty"`
	// The ip protocol name (tcp, udp, icmp or -1)
	// +optional
	IPProtocol string `json:"ipProtocol,omitempty"`
	// The ip range of the security group rule
	// +optional
	IPRange string `json:"ipRange,omitempty"`
	// The beginning of the port range
	// +optional
	FromPortRange int32 `json:"fromPortRange,omitempty"`
	// The end of the port range
	// +optional
	ToPortRange int32 `json:"toPortRange,omitempty"`
	// The security group rule id
	// +optional
	ResourceID string `json:"resourceId,omitempty"`
}

// OscResourceMapReference is a map between resourceId and resourceName (tag Name with cluster UID).
type OscResourceMapReference struct {
	ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

// OscNetworkResource is a colletion of each cluster Ref.
type OscNetworkResource struct {
	// Map between LoadbalancerId  and LoadbalancerName (Load Balancer tag Name with cluster UID)
	LoadbalancerRef OscResourceMapReference `json:"LoadbalancerRef,omitempty"`
	// Map between NetId  and NetName (Net tag Name with cluster UID)
	NetRef OscResourceMapReference `json:"netref,omitempty"`
	// Map between SubnetId  and SubnetName (Subnet tag Name with cluster UID)
	SubnetRef OscResourceMapReference `json:"subnetref,omitempty"`
	// Map between InternetServiceId  and InternetServiceName (Internet Service tag Name with cluster UID)
	InternetServiceRef OscResourceMapReference `json:"internetserviceref,omitempty"`
	// Map between RouteTablesId  and RouteTablesName (Route Tables tag Name with cluster UID)
	RouteTablesRef    OscResourceMapReference `json:"routetableref,omitempty"`
	LinkRouteTableRef OscResourceMapReference `json:"linkroutetableref,omitempty"`
	// Map between RouteId  and RouteName (Route tag Name with cluster UID)
	RouteRef OscResourceMapReference `json:"routeref,omitempty"`
	// Map between PublicIpId  and PublicIPName (Public IP tag Name with cluster UID)
	SecurityGroupsRef    OscResourceMapReference `json:"securitygroupref,omitempty"`
	SecurityGroupRuleRef OscResourceMapReference `json:"securitygroupruleref,omitempty"`
	PublicIPRef          OscResourceMapReference `json:"publicipref,omitempty"`
	// Map between NatServiceId  and NatServiceName (Nat Service tag Name with cluster UID)
	NatServiceRef OscResourceMapReference `json:"natref,omitempty"`
}

// OscNodeResource is a collection of node Ref.
type OscNodeResource struct {
	VolumeRef       OscResourceMapReference `json:"volumeRef,omitempty"`
	ImageRef        OscResourceMapReference `json:"imageRef,omitempty"`
	KeypairRef      OscResourceMapReference `json:"keypairRef,omitempty"`
	VMRef           OscResourceMapReference `json:"vmRef,omitempty"`
	LinkPublicIPRef OscResourceMapReference `json:"linkPublicIPRef,omitempty"`
}

// OscImage is the image.
type OscImage struct {
	Name       string `json:"name,omitempty"`
	ResourceID string `json:"resourceId,omitempty"`
}

// OscVolume  is the volume.
type OscVolume struct {
	Name          string `json:"name,omitempty"`
	Iops          int32  `json:"iops,omitempty"`
	Size          int32  `json:"size,omitempty"`
	SubregionName string `json:"subregionName,omitempty"`
	VolumeType    string `json:"volumeType,omitempty"`
	ResourceID    string `json:"resourceId,omitempty"`
}

// OscKeypair is the keypair.
type OscKeypair struct {
	Name        string `json:"name,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ResourceID  string `json:"resourceId,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
}

// OscVM is the vm.
type OscVM struct {
	Name               string                    `json:"name,omitempty"`
	ImageID            string                    `json:"imageId,omitempty"`
	KeypairName        string                    `json:"keypairName,omitempty"`
	VMType             string                    `json:"vmType,omitempty"`
	VolumeName         string                    `json:"volumeName,omitempty"`
	DeviceName         string                    `json:"deviceName,omitempty"`
	SubnetName         string                    `json:"subnetName,omitempty"`
	LoadBalancerName   string                    `json:"loadBalancerName,omitempty"`
	PublicIPName       string                    `json:"publicIPName,omitempty"`
	SubregionName      string                    `json:"subregionName,omitempty"`
	PrivateIPS         []OscPrivateIPElement     `json:"privateIPs,omitempty"`
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	ResourceID         string                    `json:"resourceId,omitempty"`
	Role               string                    `json:"role,omitempty"`
	ClusterName        string                    `json:"clusterName,omitempty"`
}

// VMState is the status of vm.
type VMState string

var (
	// VMStatePending is the pending state of the vm.
	VMStatePending = VMState("pending")
	// VMStateRunning is the running state of the vm.
	VMStateRunning = VMState("running")
	// VMStateShuttingDown is the shutting-down state of the vm.
	VMStateShuttingDown = VMState("shutting-down")
	// VMStateTerminated is the terminated state of the vm.
	VMStateTerminated = VMState("terminated")
	// VMStateStopping is the stopping state of the vm.
	VMStateStopping = VMState("stopping")
	// VMStateStopped is the stopped state of the vm.
	VMStateStopped = VMState("stopped")
	// DefaultClusterName is the default cluster Name.
	DefaultClusterName = "cluster-api"

	// DefaultVMSubregionName is the default VMSubregionName.
	DefaultVMSubregionName = "eu-west-2a"
	// DefaultVMImageID is the default VMImageID.
	DefaultVMImageID = "ami-2fe74243"
	// DefaultVMKeypairName is the default VMKeypairName.
	DefaultVMKeypairName = "cluster-api"
	// DefaultVMType is the default VMType.
	DefaultVMType = "tinav5.c4r8p1"
	// DefaultVMDeviceName is the default VMDeviceName.
	DefaultVMDeviceName = "/dev/xvdb"
	// DefaultVMPrivateIPKwName is the default VMPrivateIPKwName.
	DefaultVMPrivateIPKwName = "cluster-api-privateip-kw"
	// DefaultVMPrivateIPKcpName is the default VMPrivateIPKcpName.
	DefaultVMPrivateIPKcpName = "cluster-api-privateip-kcp"
	// DefaultVMPrivateIPKcp is the default VMPrivateIPKcp.
	DefaultVMPrivateIPKcp = "10.0.0.38"
	// DefaultVMPrivateIPKw is the default VMPrivateIPKw.
	DefaultVMPrivateIPKw = "10.0.0.138"
	// DefaultVMKwName is the default VMKwName.
	DefaultVMKwName = "cluster-api-vm-kw"
	// DefaultVMKwType is the default VMSubregionName.
	DefaultVMKwType = "tinav5.c4r8p1"
	// DefaultVMKcpName is the default VMKcpName.
	DefaultVMKcpName = "cluster-api-vm-kcp"
	// DefaultVMKcpType is the default VMKcpType.
	DefaultVMKcpType = "tinav5.c4r8p1"
	// DefaultVolumeKcpName is the default VolumeKcpName.
	DefaultVolumeKcpName = "cluster-api-volume-kcp"
	// DefaultVolumeKcpIops is the default VolumeKcpIops.
	DefaultVolumeKcpIops int32 = 1000
	// DefaultVolumeKcpSize is the default VMSubregionName.
	DefaultVolumeKcpSize int32 = 30
	// DefaultVolumeKcpSubregionName is the default VMSubregionName.
	DefaultVolumeKcpSubregionName = "eu-west-2a"
	// DefaultVolumeKcpType is the default VMSubregionName.
	DefaultVolumeKcpType = "io1"
	// DefaultVolumeKwName is the default VMSubregionName.
	DefaultVolumeKwName = "cluster-api-volume-kw"
	// DefaultVolumeKwIops is the default VMSubregionName.
	DefaultVolumeKwIops int32 = 1000
	// DefaultVolumeKwSize is the default VMSubregionName.
	DefaultVolumeKwSize int32 = 30
	// DefaultVolumeKwSubregionName is the default VMSubregionName.
	DefaultVolumeKwSubregionName = "eu-west-2a"
	// DefaultVolumeKwType is the default VMSubregionName.
	DefaultVolumeKwType = "io1"
	// DefaultLoadBalancerName is the default VMSubregionName.
	DefaultLoadBalancerName = "OscClusterApi-1"
	// DefaultLoadBalancerType is the default VMSubregionName.
	DefaultLoadBalancerType = "internet-facing"
	// DefaultBackendPort is the default VMSubregionName.
	DefaultBackendPort int32 = 6443
	// DefaultBackendProtocol is the default VMSubregionName.
	DefaultBackendProtocol = "TCP"
	// DefaultLoadBalancerPort is the default VMSubregionName.
	DefaultLoadBalancerPort int32 = 6443
	// DefaultLoadBalancerProtocol is the default VMSubregionName.
	DefaultLoadBalancerProtocol = "TCP"
	// DefaultCheckInterval is the default VMSubregionName.
	DefaultCheckInterval int32 = 5
	// DefaultHealthyThreshold is the default VMSubregionName.
	DefaultHealthyThreshold int32 = 5
	// DefaultUnhealthyThreshold is the default VMSubregionName.
	DefaultUnhealthyThreshold int32 = 2
	// DefaultTimeout is the default VMSubregionName.
	DefaultTimeout int32 = 5
	// DefaultProtocol is the default VMSubregionName.
	DefaultProtocol = "TCP"
	// DefaultPort is the default VMSubregionName.
	DefaultPort int32 = 6443
	// DefaultIPRange is the default VMSubregionName.
	DefaultIPRange = "10.0.0.0/24"
	// DefaultIPSubnetRange is the default VMSubregionName.
	DefaultIPSubnetRange = "10.0.0.0/24"
	// DefaultIPSubnetKcpRange is the default VMSubregionName.
	DefaultIPSubnetKcpRange = "10.0.0.32/28"
	// DefaultIPSubnetKwRange is the default VMSubregionName.
	DefaultIPSubnetKwRange = "10.0.0.128/26"
	// DefaultIPSubnetPublicRange is the default VMSubregionName.
	DefaultIPSubnetPublicRange = "10.0.0.8/29"
	// DefaultIPSubnetNatRange is the default VMSubregionName.
	DefaultIPSubnetNatRange = "10.0.0.0/29"
	// DefaultTargetType is the default VMSubregionName.
	DefaultTargetType = "gateway"
	// DefaultTargetKwName is the default VMSubregionName.
	DefaultTargetKwName = "cluster-api-natservice"
	// DefaultTargetKwType is the default VMSubregionName.
	DefaultTargetKwType = "nat"
	// DefaultDestinationKw is the default VMSubregionName.
	DefaultDestinationKw = "0.0.0.0/0"
	// DefaultRouteTableKwName is the default VMSubregionName.
	DefaultRouteTableKwName = "cluster-api-routetable-kw"
	// DefaultRouteKwName is the default VMSubregionName.
	DefaultRouteKwName = "cluster-api-route-kw"
	// DefaultTargetKcpName is the default VMSubregionName.
	DefaultTargetKcpName = "cluster-api-natservice"
	// DefaultTargetKcpType is the default VMSubregionName.
	DefaultTargetKcpType = "nat"
	// DefaultDestinationKcp is the default VMSubregionName.
	DefaultDestinationKcp = "0.0.0.0/0"
	// DefaultRouteTableKcpName is the default VMSubregionName.
	DefaultRouteTableKcpName = "cluster-api-routetable-kcp"
	// DefaultRouteKcpName is the default VMSubregionName.
	DefaultRouteKcpName = "cluster-api-route-kcp"
	// DefaultTargetPublicName is the default VMSubregionName.
	DefaultTargetPublicName = "cluster-api-internetservice"
	// DefaultTargetPublicType is the default VMSubregionName.
	DefaultTargetPublicType = "gateway"
	// DefaultDestinationPublic is the default VMSubregionName.
	DefaultDestinationPublic = "0.0.0.0/0"
	// DefaultRouteTablePublicName is the default VMSubregionName.
	DefaultRouteTablePublicName = "cluster-api-routetable-public"
	// DefaultRoutePublicName is the default VMSubregionName.
	DefaultRoutePublicName = "cluster-api-route-public"
	// DefaultTargetNatName is the default VMSubregionName.
	DefaultTargetNatName = "cluster-api-internetservice"
	// DefaultTargetNatType is the default VMSubregionName.
	DefaultTargetNatType = "gateway"
	// DefaultDestinationNat is the default VMSubregionName.
	DefaultDestinationNat = "0.0.0.0/0"
	// DefaultRouteTableNatName is the default VMSubregionName.
	DefaultRouteTableNatName = "cluster-api-routetable-nat"
	// DefaultRouteNatName is the default VMSubregionName.
	DefaultRouteNatName = "cluster-api-route-nat"
	// DefaultPublicIPNatName is the default VMSubregionName.
	DefaultPublicIPNatName = "cluster-api-publicip-nat"
	// DefaultNatServiceName is the default VMSubregionName.
	DefaultNatServiceName = "cluster-api-natservice"
	// DefaultSubnetName is the default VMSubregionName.
	DefaultSubnetName = "cluster-api-subnet"
	// DefaultSubnetKcpName is the default VMSubregionName.
	DefaultSubnetKcpName = "cluster-api-subnet-kcp"
	// DefaultSubnetKwName is the default VMSubregionName.
	DefaultSubnetKwName = "cluster-api-subnet-kw"
	// DefaultSubnetPublicName is the default VMSubregionName.
	DefaultSubnetPublicName = "cluster-api-subnet-public"
	// DefaultSubnetNatName is the default VMSubregionName.
	DefaultSubnetNatName = "cluster-api-subnet-nat"
	// DefaultNetName is the default VMSubregionName.
	DefaultNetName = "cluster-api-net"
	// DefaultInternetServiceName is the default VMSubregionName.
	DefaultInternetServiceName = "cluster-api-internetservice"
	// DefaultSecurityGroupKwName is the default VMSubregionName.
	DefaultSecurityGroupKwName = "cluster-api-securitygroup-kw"
	// DefaultDescriptionKw is the default VMSubregionName.
	DefaultDescriptionKw = "Security Group Kw with cluster-api"
	// DefaultSecurityGroupRuleAPIKubeletKwName is the default VMSubregionName.
	DefaultSecurityGroupRuleAPIKubeletKwName = "cluster-api-securitygrouprule-api-kubelet-kw"
	// DefaultFlowAPIKubeletKw is the default VMSubregionName.
	DefaultFlowAPIKubeletKw = "Inbound"
	// DefaultIPProtocolAPIKubeletKw is the default VMSubregionName.
	DefaultIPProtocolAPIKubeletKw = "tcp"
	// DefaultRuleIPRangeAPIKubeletKw is the default VMSubregionName.
	DefaultRuleIPRangeAPIKubeletKw = "10.0.0.128/26"
	// DefaultFromPortRangeAPIKubeletKw is the default VMSubregionName.
	DefaultFromPortRangeAPIKubeletKw int32 = 10250
	// DefaultToPortRangeAPIKubeletKw is the default VMSubregionName.
	DefaultToPortRangeAPIKubeletKw int32 = 10250
	// DefaultSecurityGroupRuleAPIKubeletKcpName is the default VMSubregionName.
	DefaultSecurityGroupRuleAPIKubeletKcpName = "cluster-api-securitygrouprule-api-kubelet-kcp"
	// DefaultFlowAPIKubeletKcp is the default VMSubregionName.
	DefaultFlowAPIKubeletKcp = "Inbound"
	// DefaultIPProtocolAPIKubeletKcp is the default VMSubregionName.
	DefaultIPProtocolAPIKubeletKcp = "tcp"
	// DefaultRuleIPRangeAPIKubeletKcp is the default VMSubregionName.
	DefaultRuleIPRangeAPIKubeletKcp = "10.0.0.32/28"
	// DefaultFromPortRangeAPIKubeletKcp is the default VMSubregionName.
	DefaultFromPortRangeAPIKubeletKcp int32 = 10250
	// DefaultToPortRangeAPIKubeletKcp is the default VMSubregionName.
	DefaultToPortRangeAPIKubeletKcp int32 = 10250
	// DefaultSecurityGroupRuleNodeIPKwName is the default VMSubregionName.
	DefaultSecurityGroupRuleNodeIPKwName = "cluster-api-securitygrouprule-nodeip-kw"
	// DefaultFlowNodeIPKw is the default VMSubregionName.
	DefaultFlowNodeIPKw = "Inbound"
	// DefaultIPProtocolNodeIPKw is the default VMSubregionName.
	DefaultIPProtocolNodeIPKw = "tcp"
	// DefaultRuleIPRangeNodeIPKw is the default VMSubregionName.
	DefaultRuleIPRangeNodeIPKw = "10.0.0.128/26"
	// DefaultFromPortRangeNodeIPKw is the default VMSubregionName.
	DefaultFromPortRangeNodeIPKw int32 = 30000
	// DefaultToPortRangeNodeIPKw is the default VMSubregionName.
	DefaultToPortRangeNodeIPKw int32 = 32767
	// DefaultSecurityGroupRuleNodeIPKcpName is the default VMSubregionName.
	DefaultSecurityGroupRuleNodeIPKcpName = "cluster-api-securitygrouprule-nodeip-kcp"
	// DefaultFlowNodeIPKcp is the default VMSubregionName.
	DefaultFlowNodeIPKcp = "Inbound"
	// DefaultIPProtocolNodeIPKcp is the default VMSubregionName.
	DefaultIPProtocolNodeIPKcp = "tcp"
	// DefaultRuleIPRangeNodeIPKcp is the default VMSubregionName.
	DefaultRuleIPRangeNodeIPKcp = "10.0.0.32/28"
	// DefaultFromPortRangeNodeIPKcp is the default VMSubregionName.
	DefaultFromPortRangeNodeIPKcp int32 = 30000
	// DefaultToPortRangeNodeIPKcp is the default VMSubregionName.
	DefaultToPortRangeNodeIPKcp int32 = 32767
	// DefaultSecurityGroupKcpName is the default VMSubregionName.
	DefaultSecurityGroupKcpName = "cluster-api-securitygroup-kcp"
	// DefaultDescriptionKcp is the default VMSubregionName.
	DefaultDescriptionKcp = "Security Group Kcp with cluster-api"
	// DefaultSecurityGroupRuleAPIKwName is the default VMSubregionName.
	DefaultSecurityGroupRuleAPIKwName = "cluster-api-securitygrouprule-api-kw"
	// DefaultFlowAPIKw is the default VMSubregionName.
	DefaultFlowAPIKw = "Inbound"
	// DefaultIPProtocolAPIKw is the default VMSubregionName.
	DefaultIPProtocolAPIKw = "tcp"
	// DefaultRuleIPRangeAPIKw is the default VMSubregionName.
	DefaultRuleIPRangeAPIKw = "10.0.0.128/26"
	// DefaultFromPortRangeAPIKw is the default VMSubregionName.
	DefaultFromPortRangeAPIKw int32 = 6443
	// DefaultToPortRangeAPIKw is the default VMSubregionName.
	DefaultToPortRangeAPIKw int32 = 6443
	// DefaultSecurityGroupRuleAPIKcpName is the default VMSubregionName.
	DefaultSecurityGroupRuleAPIKcpName = "cluster-api-securitygrouprule-api-kcp"
	// DefaultFlowAPIKcp is the default VMSubregionName.
	DefaultFlowAPIKcp = "Inbound"
	// DefaultIPProtocolAPIKcp is the default VMSubregionName.
	DefaultIPProtocolAPIKcp = "tcp"
	// DefaultRuleIPRangeAPIKcp is the default VMSubregionName.
	DefaultRuleIPRangeAPIKcp = "10.0.0.32/28"
	// DefaultFromPortRangeAPIKcp is the default VMSubregionName.
	DefaultFromPortRangeAPIKcp int32 = 6443
	// DefaultToPortRangeAPIKcp is the default VMSubregionName.
	DefaultToPortRangeAPIKcp int32 = 6443
	// DefaultSecurityGroupRuleEtcdName is the default VMSubregionName.
	DefaultSecurityGroupRuleEtcdName = "cluster-api-securitygrouprule-etcd"
	// DefaultFlowEtcd is the default VMSubregionName.
	DefaultFlowEtcd = "Inbound"
	// DefaultIPProtocolEtcd is the default VMSubregionName.
	DefaultIPProtocolEtcd = "tcp"
	// DefaultRuleIPRangeEtcd is the default VMSubregionName.
	DefaultRuleIPRangeEtcd = "10.0.0.32/28"
	// DefaultFromPortRangeEtcd is the default VMSubregionName.
	DefaultFromPortRangeEtcd int32 = 2378
	// DefaultToPortRangeEtcd is the default VMSubregionName.
	DefaultToPortRangeEtcd int32 = 2379
	// DefaultSecurityGroupRuleKubeletKcpName is the default VMSubregionName.
	DefaultSecurityGroupRuleKubeletKcpName = "cluster-api-securitygrouprule-kubelet-kcp"
	// DefaultFlowKubeletKcp is the default VMSubregionName.
	DefaultFlowKubeletKcp = "Inbound"
	// DefaultIPProtocolKubeletKcp is the default VMSubregionName.
	DefaultIPProtocolKubeletKcp = "tcp"
	// DefaultRuleIPRangeKubeletKcp is the default VMSubregionName.
	DefaultRuleIPRangeKubeletKcp = "10.0.0.32/28"
	// DefaultFromPortRangeKubeletKcp is the default VMSubregionName.
	DefaultFromPortRangeKubeletKcp int32 = 10250
	// DefaultToPortRangeKubeletKcp is the default VMSubregionName.
	DefaultToPortRangeKubeletKcp int32 = 10252
	// DefaultSecurityGroupLbName is the default VMSubregionName.
	DefaultSecurityGroupLbName = "cluster-api-securitygroup-lb"
	// DefaultDescriptionLb is the default VMSubregionName.
	DefaultDescriptionLb = "Security Group Lb with cluster-api"
	// DefaultSecurityGroupRuleLbName is the default VMSubregionName.
	DefaultSecurityGroupRuleLbName = "cluster-api-securitygrouprule-lb"
	// DefaultFlowLb is the default VMSubregionName.
	DefaultFlowLb = "Inbound"
	// DefaultIPProtocolLb is the default VMSubregionName.
	DefaultIPProtocolLb = "tcp"
	// DefaultRuleIPRangeLb is the default VMSubregionName.
	DefaultRuleIPRangeLb = "0.0.0.0/0"
	// DefaultFromPortRangeLb is the default VMSubregionName.
	DefaultFromPortRangeLb int32 = 6443
	// DefaultToPortRangeLb is the default VMSubregionName.
	DefaultToPortRangeLb int32 = 6443
)

// SetDefaultValue set the Net default values.
func (net *OscNet) SetDefaultValue() {
	var netName = DefaultNetName
	if net.ClusterName != "" {
		netName = strings.ReplaceAll(DefaultNetName, DefaultClusterName, net.ClusterName)
	}
	if net.IPRange == "" {
		net.IPRange = DefaultIPRange
	}
	if net.Name == "" {
		net.Name = netName
	}
}

// SetVolumeDefaultValue set the Volume default values from volume configuration.
func (node *OscNode) SetVolumeDefaultValue() {
	if len(node.Volumes) == 0 {
		var volume OscVolume
		var volumeKcpName = DefaultVolumeKcpName
		var volumeKwName = DefaultVolumeKwName
		var volumeKcpSubregionName = DefaultVolumeKcpSubregionName
		var volumeKwSubregionName = DefaultVolumeKwSubregionName
		if node.ClusterName != "" {
			volumeKcpName = strings.ReplaceAll(DefaultVolumeKcpName, DefaultClusterName, node.ClusterName)
			volumeKwName = strings.ReplaceAll(DefaultVolumeKwName, DefaultClusterName, node.ClusterName)
			volumeKcpSubregionName = strings.ReplaceAll(DefaultVolumeKcpSubregionName, DefaultClusterName, node.ClusterName)
			volumeKwSubregionName = strings.ReplaceAll(DefaultVolumeKwSubregionName, DefaultClusterName, node.ClusterName)
		}
		if node.VM.Role == "controlplane" {
			volume = OscVolume{
				Name:          volumeKcpName,
				Iops:          DefaultVolumeKcpIops,
				Size:          DefaultVolumeKcpSize,
				SubregionName: volumeKcpSubregionName,
				VolumeType:    DefaultVolumeKcpType,
			}
		} else {
			volume = OscVolume{
				Name:          volumeKwName,
				Iops:          DefaultVolumeKwIops,
				Size:          DefaultVolumeKwSize,
				SubregionName: volumeKwSubregionName,
				VolumeType:    DefaultVolumeKwType,
			}
		}
		node.Volumes = append(node.Volumes, &volume)
	}
}

// SetDefaultValue set the Internet Service default values.
func (igw *OscInternetService) SetDefaultValue() {
	var internetServiceName = DefaultInternetServiceName
	if igw.ClusterName != "" {
		internetServiceName = strings.ReplaceAll(DefaultInternetServiceName, DefaultClusterName, igw.ClusterName)
	}
	if igw.Name == "" {
		igw.Name = internetServiceName
	}
}

// SetDefaultValue set the vm default values.
func (vm *OscVM) SetDefaultValue() {
	var vmKcpName = DefaultVMKcpName
	var vmKwName = DefaultVMKwName
	var subnetKcpName = DefaultSubnetKcpName
	var subnetKwName = DefaultSubnetKwName
	var securityGroupKcpName = DefaultSecurityGroupKcpName
	var securityGroupKwName = DefaultSecurityGroupKwName
	var VMPrivateIPKcpName = DefaultVMPrivateIPKcpName
	var VMPrivateIPKwName = DefaultVMPrivateIPKwName
	var volumeKcpName = DefaultVolumeKcpName
	var volumeKwName = DefaultVolumeKwName
	if vm.ClusterName != "" {
		volumeKcpName = strings.ReplaceAll(DefaultVolumeKcpName, DefaultClusterName, vm.ClusterName)
		volumeKwName = strings.ReplaceAll(DefaultVolumeKwName, DefaultClusterName, vm.ClusterName)
		vmKcpName = strings.ReplaceAll(DefaultVMKcpName, DefaultClusterName, vm.ClusterName)
		vmKwName = strings.ReplaceAll(DefaultVMKwName, DefaultClusterName, vm.ClusterName)
		subnetKcpName = strings.ReplaceAll(DefaultSubnetKcpName, DefaultClusterName, vm.ClusterName)
		subnetKwName = strings.ReplaceAll(DefaultSubnetKwName, DefaultClusterName, vm.ClusterName)
		securityGroupKcpName = strings.ReplaceAll(DefaultSecurityGroupKcpName, DefaultClusterName, vm.ClusterName)
		securityGroupKwName = strings.ReplaceAll(DefaultSecurityGroupKwName, DefaultClusterName, vm.ClusterName)
		VMPrivateIPKcpName = strings.ReplaceAll(DefaultVMPrivateIPKcpName, DefaultClusterName, vm.ClusterName)
		VMPrivateIPKwName = strings.ReplaceAll(DefaultVMPrivateIPKwName, DefaultClusterName, vm.ClusterName)
	}
	if vm.Role == "controlplane" {
		if vm.Name == "" {
			vm.Name = vmKcpName
		}
		if vm.VMType == "" {
			vm.VMType = DefaultVMKcpType
		}
		if vm.SubnetName == "" {
			vm.SubnetName = subnetKcpName
		}
		if vm.VolumeName == "" {
			vm.VolumeName = volumeKcpName
		}
		if vm.LoadBalancerName == "" {
			vm.LoadBalancerName = DefaultLoadBalancerName
		}
		if len(vm.SecurityGroupNames) == 0 {
			securityGroup := OscSecurityGroupElement{
				Name: securityGroupKcpName,
			}
			vm.SecurityGroupNames = []OscSecurityGroupElement{securityGroup}
		}
		if len(vm.PrivateIPS) == 0 {
			privateIP := OscPrivateIPElement{
				Name:      VMPrivateIPKcpName,
				PrivateIP: DefaultVMPrivateIPKcp,
			}
			vm.PrivateIPS = []OscPrivateIPElement{privateIP}
		}
	} else {
		if vm.Name == "" {
			vm.Name = vmKwName
		}
		if vm.VMType == "" {
			vm.VMType = DefaultVMKwType
		}

		if vm.VolumeName == "" {
			vm.VolumeName = volumeKwName
		}

		if vm.SubnetName == "" {
			vm.SubnetName = subnetKwName
		}
		if len(vm.SecurityGroupNames) == 0 {
			securityGroup := OscSecurityGroupElement{
				Name: securityGroupKwName,
			}
			vm.SecurityGroupNames = []OscSecurityGroupElement{securityGroup}
		}
		if len(vm.PrivateIPS) == 0 {
			privateIP := OscPrivateIPElement{
				Name:      VMPrivateIPKwName,
				PrivateIP: DefaultVMPrivateIPKw,
			}
			vm.PrivateIPS = []OscPrivateIPElement{privateIP}
		}
	}
	if vm.ImageID == "" {
		vm.ImageID = DefaultVMImageID
	}
	if vm.KeypairName == "" {
		vm.KeypairName = DefaultVMKeypairName
	}
	if vm.DeviceName == "" {
		vm.DeviceName = DefaultVMDeviceName
	}
	if vm.SubregionName == "" {
		vm.SubregionName = DefaultVMSubregionName
	}
}

// SetDefaultValue set the Subnet default values.
func (sub *OscSubnet) SetDefaultValue() {
	if sub.IPSubnetRange == "" {
		sub.IPSubnetRange = DefaultIPSubnetRange
	}
	if sub.Name == "" {
		sub.Name = DefaultSubnetName
	}
}

// SetDefaultValue set the Nat Service default values.
func (nat *OscNatService) SetDefaultValue() {
	var natServiceName = DefaultNatServiceName
	var publicIPNatName = DefaultPublicIPNatName
	var subnetNatName = DefaultSubnetNatName
	if nat.ClusterName != "" {
		natServiceName = strings.ReplaceAll(DefaultNatServiceName, DefaultClusterName, nat.ClusterName)
		publicIPNatName = strings.ReplaceAll(DefaultPublicIPNatName, DefaultClusterName, nat.ClusterName)
		subnetNatName = strings.ReplaceAll(DefaultSubnetNatName, DefaultClusterName, nat.ClusterName)
	}
	if nat.Name == "" {
		nat.Name = natServiceName
	}
	if nat.PublicIPName == "" {
		nat.PublicIPName = publicIPNatName
	}
	if nat.SubnetName == "" {
		nat.SubnetName = subnetNatName
	}
}

// SetRouteTableDefaultValue set the Route Table default values from network configuration.
func (network *OscNetwork) SetRouteTableDefaultValue() {
	if len(network.RouteTables) == 0 {
		var routeKwName = DefaultRouteKwName
		var targetKwName = DefaultTargetKwName
		var routeTableKwName = DefaultRouteTableKwName
		var subnetKwName = DefaultSubnetKwName

		var routeKcpName = DefaultRouteKcpName
		var targetKcpName = DefaultTargetKcpName
		var routeTableKcpName = DefaultRouteTableKcpName
		var subnetKcpName = DefaultSubnetKcpName

		var routePublicName = DefaultRoutePublicName
		var targetPublicName = DefaultTargetPublicName
		var routeTablePublicName = DefaultRouteTablePublicName
		var subnetPublicName = DefaultSubnetPublicName

		var routeNatName = DefaultRouteNatName
		var targetNatName = DefaultTargetNatName
		var routeTableNatName = DefaultRouteTableNatName
		var subnetNatName = DefaultSubnetNatName

		if network.ClusterName != "" {
			routeKwName = strings.ReplaceAll(DefaultRouteKwName, DefaultClusterName, network.ClusterName)
			targetKwName = strings.ReplaceAll(DefaultTargetKwName, DefaultClusterName, network.ClusterName)
			routeTableKwName = strings.ReplaceAll(DefaultRouteTableKwName, DefaultClusterName, network.ClusterName)
			subnetKwName = strings.ReplaceAll(DefaultSubnetKwName, DefaultClusterName, network.ClusterName)

			routeKcpName = strings.ReplaceAll(DefaultRouteKcpName, DefaultClusterName, network.ClusterName)
			targetKcpName = strings.ReplaceAll(DefaultTargetKcpName, DefaultClusterName, network.ClusterName)
			routeTableKcpName = strings.ReplaceAll(DefaultRouteTableKcpName, DefaultClusterName, network.ClusterName)
			subnetKcpName = strings.ReplaceAll(DefaultSubnetKcpName, DefaultClusterName, network.ClusterName)

			routePublicName = strings.ReplaceAll(DefaultRoutePublicName, DefaultClusterName, network.ClusterName)
			targetPublicName = strings.ReplaceAll(DefaultTargetPublicName, DefaultClusterName, network.ClusterName)
			routeTablePublicName = strings.ReplaceAll(DefaultRouteTablePublicName, DefaultClusterName, network.ClusterName)
			subnetPublicName = strings.ReplaceAll(DefaultSubnetPublicName, DefaultClusterName, network.ClusterName)

			routeNatName = strings.ReplaceAll(DefaultRouteNatName, DefaultClusterName, network.ClusterName)
			targetNatName = strings.ReplaceAll(DefaultTargetNatName, DefaultClusterName, network.ClusterName)
			routeTableNatName = strings.ReplaceAll(DefaultRouteTableNatName, DefaultClusterName, network.ClusterName)
			subnetNatName = strings.ReplaceAll(DefaultSubnetNatName, DefaultClusterName, network.ClusterName)
		}

		routeKw := OscRoute{
			Name:        routeKwName,
			TargetName:  targetKwName,
			TargetType:  DefaultTargetKwType,
			Destination: DefaultDestinationKw,
		}

		routeKcp := OscRoute{
			Name:        routeKcpName,
			TargetName:  targetKcpName,
			TargetType:  DefaultTargetKcpType,
			Destination: DefaultDestinationKcp,
		}
		routePublic := OscRoute{
			Name:        routePublicName,
			TargetName:  targetPublicName,
			TargetType:  DefaultTargetPublicType,
			Destination: DefaultDestinationPublic,
		}

		routeNat := OscRoute{
			Name:        routeNatName,
			TargetName:  targetNatName,
			TargetType:  DefaultTargetNatType,
			Destination: DefaultDestinationNat,
		}

		routeTableKw := OscRouteTable{
			Name:       routeTableKwName,
			SubnetName: subnetKwName,
			Routes:     []OscRoute{routeKw},
		}
		network.RouteTables = append(network.RouteTables, &routeTableKw)
		routeTableKcp := OscRouteTable{
			Name:       routeTableKcpName,
			SubnetName: subnetKcpName,
			Routes:     []OscRoute{routeKcp},
		}
		network.RouteTables = append(network.RouteTables, &routeTableKcp)

		routeTablePublic := OscRouteTable{
			Name:       routeTablePublicName,
			SubnetName: subnetPublicName,
			Routes:     []OscRoute{routePublic},
		}
		network.RouteTables = append(network.RouteTables, &routeTablePublic)

		routeTableNat := OscRouteTable{
			Name:       routeTableNatName,
			SubnetName: subnetNatName,
			Routes:     []OscRoute{routeNat},
		}
		network.RouteTables = append(network.RouteTables, &routeTableNat)
	}
}

// SetSecurityGroupDefaultValue set the security group default value.
func (network *OscNetwork) SetSecurityGroupDefaultValue() {
	if len(network.SecurityGroups) == 0 {
		var securityGroupRuleAPIKubeletKwName = DefaultSecurityGroupRuleAPIKubeletKwName
		var securityGroupRuleAPIKubeletKcpName = DefaultSecurityGroupRuleAPIKubeletKcpName
		var securityGroupRuleNodeIPKwName = DefaultSecurityGroupRuleNodeIPKwName
		var securityGroupRuleNodeIPKcpName = DefaultSecurityGroupRuleNodeIPKcpName
		var securityGroupKwName = DefaultSecurityGroupKwName
		var securityGroupRuleAPIKwName = DefaultSecurityGroupRuleAPIKwName
		var securityGroupRuleAPIKcpName = DefaultSecurityGroupRuleAPIKcpName
		var securityGroupRuleEtcdName = DefaultSecurityGroupRuleEtcdName
		var securityGroupRuleKubeletKcpName = DefaultSecurityGroupRuleKubeletKcpName
		var securityGroupKcpName = DefaultSecurityGroupKcpName
		var securityGroupRuleLbName = DefaultSecurityGroupRuleLbName
		var securityGroupLbName = DefaultSecurityGroupLbName
		if network.ClusterName != "" {
			securityGroupRuleAPIKubeletKwName = strings.ReplaceAll(DefaultSecurityGroupRuleAPIKubeletKwName, DefaultClusterName, network.ClusterName)
			securityGroupRuleAPIKubeletKcpName = strings.ReplaceAll(DefaultSecurityGroupRuleAPIKubeletKcpName, DefaultClusterName, network.ClusterName)
			securityGroupRuleNodeIPKwName = strings.ReplaceAll(DefaultSecurityGroupRuleNodeIPKwName, DefaultClusterName, network.ClusterName)
			securityGroupRuleNodeIPKcpName = strings.ReplaceAll(DefaultSecurityGroupRuleNodeIPKcpName, DefaultClusterName, network.ClusterName)
			securityGroupKwName = strings.ReplaceAll(DefaultSecurityGroupKwName, DefaultClusterName, network.ClusterName)
			securityGroupRuleAPIKwName = strings.ReplaceAll(DefaultSecurityGroupRuleAPIKwName, DefaultClusterName, network.ClusterName)
			securityGroupRuleAPIKcpName = strings.ReplaceAll(DefaultSecurityGroupRuleAPIKcpName, DefaultClusterName, network.ClusterName)
			securityGroupRuleEtcdName = strings.ReplaceAll(DefaultSecurityGroupRuleEtcdName, DefaultClusterName, network.ClusterName)
			securityGroupRuleKubeletKcpName = strings.ReplaceAll(DefaultSecurityGroupRuleKubeletKcpName, DefaultClusterName, network.ClusterName)
			securityGroupKcpName = strings.ReplaceAll(DefaultSecurityGroupKcpName, DefaultClusterName, network.ClusterName)
			securityGroupRuleLbName = strings.ReplaceAll(DefaultSecurityGroupRuleLbName, DefaultClusterName, network.ClusterName)
			securityGroupLbName = strings.ReplaceAll(DefaultSecurityGroupLbName, DefaultClusterName, network.ClusterName)
		}
		securityGroupRuleAPIKubeletKw := OscSecurityGroupRule{
			Name:          securityGroupRuleAPIKubeletKwName,
			Flow:          DefaultFlowAPIKubeletKw,
			IPProtocol:    DefaultIPProtocolAPIKubeletKw,
			IPRange:       DefaultRuleIPRangeAPIKubeletKw,
			FromPortRange: DefaultFromPortRangeAPIKubeletKw,
			ToPortRange:   DefaultToPortRangeAPIKubeletKw,
		}

		securityGroupRuleAPIKubeletKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleAPIKubeletKcpName,
			Flow:          DefaultFlowAPIKubeletKcp,
			IPProtocol:    DefaultIPProtocolAPIKubeletKcp,
			IPRange:       DefaultRuleIPRangeAPIKubeletKcp,
			FromPortRange: DefaultFromPortRangeAPIKubeletKcp,
			ToPortRange:   DefaultToPortRangeAPIKubeletKcp,
		}

		securityGroupRuleNodeIPKw := OscSecurityGroupRule{
			Name:          securityGroupRuleNodeIPKwName,
			Flow:          DefaultFlowNodeIPKw,
			IPProtocol:    DefaultIPProtocolNodeIPKw,
			IPRange:       DefaultRuleIPRangeNodeIPKw,
			FromPortRange: DefaultFromPortRangeNodeIPKw,
			ToPortRange:   DefaultToPortRangeNodeIPKw,
		}

		securityGroupRuleNodeIPKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleNodeIPKcpName,
			Flow:          DefaultFlowNodeIPKcp,
			IPProtocol:    DefaultIPProtocolNodeIPKcp,
			IPRange:       DefaultRuleIPRangeNodeIPKcp,
			FromPortRange: DefaultFromPortRangeNodeIPKcp,
			ToPortRange:   DefaultToPortRangeNodeIPKcp,
		}

		securityGroupKw := OscSecurityGroup{
			Name:               securityGroupKwName,
			Description:        DefaultDescriptionKw,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleAPIKubeletKw, securityGroupRuleAPIKubeletKcp, securityGroupRuleNodeIPKw, securityGroupRuleNodeIPKcp},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupKw)

		securityGroupRuleAPIKw := OscSecurityGroupRule{
			Name:          securityGroupRuleAPIKwName,
			Flow:          DefaultFlowAPIKw,
			IPProtocol:    DefaultIPProtocolAPIKw,
			IPRange:       DefaultRuleIPRangeAPIKw,
			FromPortRange: DefaultFromPortRangeAPIKw,
			ToPortRange:   DefaultToPortRangeAPIKw,
		}

		securityGroupRuleAPIKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleAPIKcpName,
			Flow:          DefaultFlowAPIKcp,
			IPProtocol:    DefaultIPProtocolAPIKcp,
			IPRange:       DefaultRuleIPRangeAPIKcp,
			FromPortRange: DefaultFromPortRangeAPIKcp,
			ToPortRange:   DefaultToPortRangeAPIKcp,
		}

		securityGroupRuleEtcd := OscSecurityGroupRule{
			Name:          securityGroupRuleEtcdName,
			Flow:          DefaultFlowEtcd,
			IPProtocol:    DefaultIPProtocolEtcd,
			IPRange:       DefaultRuleIPRangeEtcd,
			FromPortRange: DefaultFromPortRangeEtcd,
			ToPortRange:   DefaultToPortRangeEtcd,
		}

		securityGroupRuleKubeletKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleKubeletKcpName,
			Flow:          DefaultFlowKubeletKcp,
			IPProtocol:    DefaultIPProtocolKubeletKcp,
			IPRange:       DefaultRuleIPRangeKubeletKcp,
			FromPortRange: DefaultFromPortRangeKubeletKcp,
			ToPortRange:   DefaultToPortRangeKubeletKcp,
		}

		securityGroupKcp := OscSecurityGroup{
			Name:               securityGroupKcpName,
			Description:        DefaultDescriptionKcp,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleAPIKw, securityGroupRuleAPIKcp, securityGroupRuleEtcd, securityGroupRuleKubeletKcp},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupKcp)

		securityGroupRuleLb := OscSecurityGroupRule{
			Name:          securityGroupRuleLbName,
			Flow:          DefaultFlowLb,
			IPProtocol:    DefaultIPProtocolLb,
			IPRange:       DefaultRuleIPRangeLb,
			FromPortRange: DefaultFromPortRangeLb,
			ToPortRange:   DefaultToPortRangeLb,
		}
		securityGroupLb := OscSecurityGroup{
			Name:               securityGroupLbName,
			Description:        DefaultDescriptionLb,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleLb},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupLb)
	}
}

// SetPublicIPDefaultValue set the Public Ip default values from publicip configuration.
func (network *OscNetwork) SetPublicIPDefaultValue() {
	if len(network.PublicIPS) == 0 {
		var publicIPNatName = DefaultPublicIPNatName
		if network.ClusterName != "" {
			publicIPNatName = strings.ReplaceAll(DefaultPublicIPNatName, DefaultClusterName, network.ClusterName)
		}
		publicIP := OscPublicIP{
			Name: publicIPNatName,
		}
		network.PublicIPS = append(network.PublicIPS, &publicIP)
	}
}

// SetSubnetDefaultValue set the Subnet default values from subnet configuration.
func (network *OscNetwork) SetSubnetDefaultValue() {
	if len(network.Subnets) == 0 {
		var subnetKcpName = DefaultSubnetKcpName
		var subnetKwName = DefaultSubnetKwName
		var subnetPublicName = DefaultSubnetPublicName
		var subnetNatName = DefaultSubnetNatName
		if network.ClusterName != "" {
			subnetKcpName = strings.ReplaceAll(DefaultSubnetKcpName, DefaultClusterName, network.ClusterName)
			subnetKwName = strings.ReplaceAll(DefaultSubnetKwName, DefaultClusterName, network.ClusterName)
			subnetPublicName = strings.ReplaceAll(DefaultSubnetPublicName, DefaultClusterName, network.ClusterName)
			subnetNatName = strings.ReplaceAll(DefaultSubnetNatName, DefaultClusterName, network.ClusterName)
		}
		subnetKcp := OscSubnet{
			Name:          subnetKcpName,
			IPSubnetRange: DefaultIPSubnetKcpRange,
		}
		subnetKw := OscSubnet{
			Name:          subnetKwName,
			IPSubnetRange: DefaultIPSubnetKwRange,
		}
		subnetPublic := OscSubnet{
			Name:          subnetPublicName,
			IPSubnetRange: DefaultIPSubnetPublicRange,
		}
		subnetNat := OscSubnet{
			Name:          subnetNatName,
			IPSubnetRange: DefaultIPSubnetNatRange,
		}
		network.Subnets = []*OscSubnet{
			&subnetKcp,
			&subnetKw,
			&subnetPublic,
			&subnetNat,
		}
	}
}

// SetDefaultValue set the LoadBalancer Service default values.
func (lb *OscLoadBalancer) SetDefaultValue() {
	var subnetPublicName = DefaultSubnetPublicName
	var securityGroupLbName = DefaultSecurityGroupLbName
	if lb.ClusterName != "" {
		subnetPublicName = strings.ReplaceAll(DefaultSubnetPublicName, DefaultClusterName, lb.ClusterName)
		securityGroupLbName = strings.ReplaceAll(DefaultSecurityGroupLbName, DefaultClusterName, lb.ClusterName)
	}
	if lb.LoadBalancerName == "" {
		lb.LoadBalancerName = DefaultLoadBalancerName
	}
	if lb.LoadBalancerType == "" {
		lb.LoadBalancerType = DefaultLoadBalancerType
	}
	if lb.SubnetName == "" {
		lb.SubnetName = subnetPublicName
	}
	if lb.SecurityGroupName == "" {
		lb.SecurityGroupName = securityGroupLbName
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
