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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	base64 "encoding/base64"
	"log"
	"strings"

	"golang.org/x/crypto/ssh"
)

type OscNode struct {
	Vm          OscVm        `json:"vm,omitempty"`
	Image       OscImage     `json:"image,omitempty"`
	Volumes     []*OscVolume `json:"volumes,omitempty"`
	KeyPair     OscKeypair   `json:"keypair,omitempty"`
	ClusterName string       `json:"clusterName,omitempty"`
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
	Subnets []*OscSubnet `json:"subnets,omitempty"`
	// The Internet Service configuration
	// +optional
	InternetService OscInternetService `json:"internetService,omitempty"`
	// The Nat Service configuration
	// +optional
	NatService OscNatService `json:"natService,omitempty"`
	// The Nat Services configuration
	// +optional
	NatServices []*OscNatService `json:"natServices,omitempty"`
	// The Route Table configuration
	// +optional
	RouteTables    []*OscRouteTable    `json:"routeTables,omitempty"`
	SecurityGroups []*OscSecurityGroup `json:"securityGroups,omitempty"`
	// The Public Ip configuration
	// +optional
	PublicIps []*OscPublicIp `json:"publicIps,omitempty"`
	// The name of the cluster
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The image configuration
	// +optional
	Image OscImage `json:"image,omitempty"`
	// The bastion configuration
	// + optional
	Bastion OscBastion `json:"bastion,omitempty"`
	// The subregion name
	// + optional
	SubregionName string `json:"subregionName,omitempty"`
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
	LoadBalancerProtocol string `json:"loadbalancerprotocol,omiempty"`
}

type OscLoadBalancerHealthCheck struct {
	// the time in second between two pings
	// +optional
	CheckInterval int32 `json:"checkinterval,omitempty"`
	// the consecutive number of pings which are sucessful to consider the vm healthy
	// +optional
	HealthyThreshold int32 `json:"healthythreshold,omitempty"`
	// the HealthCheck port number
	// +optional
	Port int32 `json:"port,omitempty"`
	// The HealthCheck protocol ('HTTP'|'TCP')
	// +optional
	Protocol string `json:"protocol,omitepty"`
	// the Timeout to consider VM unhealthy
	// +optional
	Timeout int32 `json:"timeout,omitempty"`
	// the consecutive number of pings which are failed to consider the vm unhealthy
	// +optional
	UnhealthyThreshold int32 `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
	// the tag name associate with the Net
	// +optional
	Name string `json:"name,omitempty"`
	// the net ip range with CIDR notation
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// the name of the cluster
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The Net Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscInternetService struct {
	// The tag name associate with the Subnet
	// +optional
	Name string `json:"name,omitempty"`
	// the name of the cluster
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// the Internet Service response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscSubnet struct {
	// The tag name associate with the Subnet
	// +optional
	Name string `json:"name,omitempty"`
	// Subnet Ip range with CIDR notation
	// +optional
	IpSubnetRange string `json:"ipSubnetRange,omitempty"`
	// The subregion name of the Subnet
	// +optional
	SubregionName string `json:"subregionName,omitempty"`
	// The Subnet Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscNatService struct {
	// The tag name associate with the Nat Service
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip tag name associated wtih a Public Ip
	// +optional
	PublicIpName string `json:"publicipname,omitempty"`
	// The subnet tag name associate with a Subnet
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
	// The name of the cluster
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// The Nat Service Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscRouteTable struct {
	// The tag name associate with the Route Table
	// +optional
	Name string `json:"name,omitempty"`
	// The subnet tag name associate with a Subnet
	Subnets []string `json:"subnets,omitempty"`
	// The Route configuration
	// +optional
	Routes []OscRoute `json:"routes,omitempty"`
	// The Route Table Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscSecurityGroup struct {
	// The tag name associate with the security group
	// +optional
	Name string `json:"name,omitempty"`
	// The description of the security group
	// +optional
	Description string `json:"description,omitempty"`
	// Should the default allow all outbound rule be deleted
	// +optional
	DeleteDefaultOutboundRule bool `json:"deleteDefaultOutboundRule,omitempty"`
	// The Security Group Rules configuration
	// +optional
	SecurityGroupRules []OscSecurityGroupRule `json:"securityGroupRules,omitempty"`
	// The Security Group Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
	Tag        string `json:"tag,omitempty"`
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
	PrivateIp string `json:"privateIp,omiteempty"`
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
	// The ip range of the security group rule
	// +optional
	IpRange string `json:"ipRange,omitempty"`
	// The name of the security group to use as target
	// +optional
	TargetSecurityGroupName string `json:"targetSecurityGroupName,omitempty"`
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

type OscNodeResource struct {
	VolumeRef       OscResourceReference `json:"volumeRef,omitempty"`
	ImageRef        OscResourceReference `json:"imageRef,omitempty"`
	KeypairRef      OscResourceReference `json:"keypairRef,omitempty"`
	VmRef           OscResourceReference `json:"vmRef,omitempty"`
	LinkPublicIpRef OscResourceReference `json:"linkPublicIpRef,omitempty"`
	PublicIpIdRef   OscResourceReference `json:"publicIpIdRef,omitempty"`
}

type OscImage struct {
	Name       string `json:"name,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
}

type OscVolume struct {
	Name          string `json:"name,omitempty"`
	Iops          int32  `json:"iops,omitempty"`
	Size          int32  `json:"size,omitempty"`
	SubregionName string `json:"subregionName,omitempty"`
	VolumeType    string `json:"volumeType,omitempty"`
	ResourceId    string `json:"resourceId,omitempty"`
}

type OscKeypair struct {
	Name          string `json:"name,omitempty"`
	PublicKey     string `json:"publicKey,omitempty"`
	ResourceId    string `json:"resourceId,omitempty"`
	ClusterName   string `json:"clusterName,omitempty"`
	DeleteKeypair bool   `json:"deleteKeypair,omitempty"`
}

type OscVm struct {
	Name               string                    `json:"name,omitempty"`
	ImageId            string                    `json:"imageId,omitempty"`
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
	Role               string                    `json:"role,omitempty"`
	ClusterName        string                    `json:"clusterName,omitempty"`
	Replica            int32                     `json:"replica,omitempty"`
	Tags               map[string]string         `json:"tags,omitempty"`
}

type OscBastion struct {
	Name               string                    `json:"name,omitempty"`
	ImageId            string                    `json:"imageId,omitempty"`
	ImageName          string                    `json:"imageName,omitempty"`
	KeypairName        string                    `json:"keypairName,omitempty"`
	VmType             string                    `json:"vmType,omitempty"`
	DeviceName         string                    `json:"deviceName,omitempty"`
	SubnetName         string                    `json:"subnetName,omitempty"`
	RootDisk           OscRootDisk               `json:"rootDisk,omitempty"`
	PublicIpName       string                    `json:"publicIpName,omitempty"`
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
	DefaultVmImageId       string = "ami-e1a786f1"

	DefaultVmKeypairName string = "cluster-api"
	DefaultVmType        string = "tinav3.c4r8p1"
	DefaultVmDeviceName  string = "/dev/sda1"

	DefaultVmBastionImageId       string = "ami-bb490c7e"
	DefaultVmBastionKeypairName   string = "cluster-api"
	DefaultVmBastionSubregionName string = "eu-west-2a"

	DefaultVmKwName               string = "cluster-api-vm-kw"
	DefaultVmKwType               string = "tinav3.c4r8p1"
	DefaultVmKcpName              string = "cluster-api-vm-kcp"
	DefaultVmKcpType              string = "tinav3.c4r8p1"
	DefaultVmBastionName          string = "cluster-api-vm-bastion"
	DefaultVmBastionType          string = "tinav3.c2r2p1"
	DefaultVolumeKcpName          string = "cluster-api-volume-kcp"
	DefaultVolumeKcpIops          int32  = 1000
	DefaultVolumeKcpSize          int32  = 30
	DefaultVolumeKcpSubregionName string = "eu-west-2a"
	DefaultVolumeKcpType          string = "io1"

	DefaultRootDiskKwType string = "io1"
	DefaultRootDiskKwSize int32  = 60
	DefaultRootDiskKwIops int32  = 1500

	DefaultRootDiskKcpType                      string = "io1"
	DefaultRootDiskKcpSize                      int32  = 60
	DefaultRootDiskKcpIops                      int32  = 1500
	DefaultRootDiskBastionType                  string = "io1"
	DefaultRootDiskBastionSize                  int32  = 15
	DefaultRootDiskBastionIops                  int32  = 1000
	DefaultVolumeKwName                         string = "cluster-api-volume-kw"
	DefaultVolumeKwIops                         int32  = 1000
	DefaultVolumeKwSize                         int32  = 30
	DefaultVolumeKwSubregionName                string = "eu-west-2a"
	DefaultVolumeKwType                         string = "io1"
	DefaultSubregionName                        string = "eu-west-2a"
	DefaultLoadBalancerName                     string = "OscClusterApi-1"
	DefaultLoadBalancerType                     string = "internet-facing"
	DefaultBackendPort                          int32  = 6443
	DefaultBackendProtocol                      string = "TCP"
	DefaultLoadBalancerPort                     int32  = 6443
	DefaultLoadBalancerProtocol                 string = "TCP"
	DefaultCheckInterval                        int32  = 5
	DefaultHealthyThreshold                     int32  = 5
	DefaultUnhealthyThreshold                   int32  = 2
	DefaultTimeout                              int32  = 5
	DefaultProtocol                             string = "TCP"
	DefaultPort                                 int32  = 6443
	DefaultIpRange                              string = "10.0.0.0/16"
	DefaultIpSubnetKcpRange                     string = "10.0.4.0/24"
	DefaultIpSubnetKwRange                      string = "10.0.3.0/24"
	DefaultIpSubnetPublicRange                  string = "10.0.2.0/24"
	DefaultTargetType                           string = "gateway"
	DefaultTargetKwName                         string = "cluster-api-natservice"
	DefaultTargetKwType                         string = "nat"
	DefaultDestinationKw                        string = "0.0.0.0/0"
	DefaultRouteTableKwName                     string = "cluster-api-routetable-kw"
	DefaultRouteKwName                          string = "cluster-api-route-kw"
	DefaultTargetKcpName                        string = "cluster-api-natservice"
	DefaultTargetKcpType                        string = "nat"
	DefaultDestinationKcp                       string = "0.0.0.0/0"
	DefaultRouteTableKcpName                    string = "cluster-api-routetable-kcp"
	DefaultRouteKcpName                         string = "cluster-api-route-kcp"
	DefaultTargetPublicName                     string = "cluster-api-internetservice"
	DefaultTargetPublicType                     string = "gateway"
	DefaultDestinationPublic                    string = "0.0.0.0/0"
	DefaultRouteTablePublicName                 string = "cluster-api-routetable-public"
	DefaultRoutePublicName                      string = "cluster-api-route-public"
	DefaultTargetNatName                        string = "cluster-api-internetservice"
	DefaultTargetNatType                        string = "gateway"
	DefaultDestinationNat                       string = "0.0.0.0/0"
	DefaultRouteTableNatName                    string = "cluster-api-routetable-nat"
	DefaultRouteNatName                         string = "cluster-api-route-nat"
	DefaultPublicIpNatName                      string = "cluster-api-publicip-nat"
	DefaultNatServiceName                       string = "cluster-api-natservice"
	DefaultSubnetName                           string = "cluster-api-subnet"
	DefaultSubnetKcpName                        string = "cluster-api-subnet-kcp"
	DefaultSubnetKwName                         string = "cluster-api-subnet-kw"
	DefaultSubnetPublicName                     string = "cluster-api-subnet-public"
	DefaultSubnetNatName                        string = "cluster-api-subnet-nat"
	DefaultNetName                              string = "cluster-api-net"
	DefaultInternetServiceName                  string = "cluster-api-internetservice"
	DefaultSecurityGroupKwName                  string = "cluster-api-securitygroup-kw"
	DefaultDescriptionKw                        string = "Security Group Kw with cluster-api"
	DefaultSecurityGroupRuleApiKubeletKwName    string = "cluster-api-securitygrouprule-api-kubelet-kw"
	DefaultFlowApiKubeletKw                     string = "Inbound"
	DefaultIpProtocolApiKubeletKw               string = "tcp"
	DefaultRuleIpRangeApiKubeletKw              string = "10.0.3.0/24"
	DefaultFromPortRangeApiKubeletKw            int32  = 10250
	DefaultToPortRangeApiKubeletKw              int32  = 10250
	DefaultSecurityGroupRuleApiKubeletKcpName   string = "cluster-api-securitygrouprule-api-kubelet-kcp"
	DefaultFlowApiKubeletKcp                    string = "Inbound"
	DefaultIpProtocolApiKubeletKcp              string = "tcp"
	DefaultRuleIpRangeApiKubeletKcp             string = "10.0.4.0/24"
	DefaultFromPortRangeApiKubeletKcp           int32  = 10250
	DefaultToPortRangeApiKubeletKcp             int32  = 10250
	DefaultSecurityGroupRuleKwNodeIpKwName      string = "cluster-api-securitygrouprule-kw-nodeip-kw"
	DefaultSecurityGroupRuleKcpNodeIpKwName     string = "cluster-api-securitygrouprule-kcp-nodeip-kw"
	DefaultFlowNodeIpKw                         string = "Inbound"
	DefaultIpProtocolNodeIpKw                   string = "tcp"
	DefaultRuleIpRangeNodeIpKw                  string = "10.0.3.0/24"
	DefaultFromPortRangeNodeIpKw                int32  = 30000
	DefaultToPortRangeNodeIpKw                  int32  = 32767
	DefaultSecurityGroupRuleKcpNodeIpKcpName    string = "cluster-api-securitugrouprule-kcp-nodeip-kcp"
	DefaultSecurityGroupRuleKwNodeIpKcpName     string = "cluster-api-securitygrouprule-kw-nodeip-kcp"
	DefaultFlowNodeIpKcp                        string = "Inbound"
	DefaultIpProtocolNodeIpKcp                  string = "tcp"
	DefaultRuleIpRangeNodeIpKcp                 string = "10.0.4.0/24"
	DefaultFromPortRangeNodeIpKcp               int32  = 30000
	DefaultToPortRangeNodeIpKcp                 int32  = 32767
	DefaultSecurityGroupKcpName                 string = "cluster-api-securitygroup-kcp"
	DefaultDescriptionKcp                       string = "Security Group Kcp with cluster-api"
	DefaultSecurityGroupRuleApiKwName           string = "cluster-api-securitygrouprule-api-kw"
	DefaultFlowApiKw                            string = "Inbound"
	DefaultIpProtocolApiKw                      string = "tcp"
	DefaultRuleIpRangeApiKw                     string = "10.0.3.0/24"
	DefaultFromPortRangeApiKw                   int32  = 6443
	DefaultToPortRangeApiKw                     int32  = 6443
	DefaultSecurityGroupRuleApiKcpName          string = "cluster-api-securitygrouprule-api-kcp"
	DefaultFlowApiKcp                           string = "Inbound"
	DefaultIpProtocolApiKcp                     string = "tcp"
	DefaultRuleIpRangeApiKcp                    string = "10.0.4.0/24"
	DefaultFromPortRangeApiKcp                  int32  = 6443
	DefaultToPortRangeApiKcp                    int32  = 6443
	DefaultSecurityGroupRuleEtcdName            string = "cluster-api-securitygrouprule-etcd"
	DefaultFlowEtcd                             string = "Inbound"
	DefaultIpProtocolEtcd                       string = "tcp"
	DefaultRuleIpRangeEtcd                      string = "10.0.4.0/24"
	DefaultFromPortRangeEtcd                    int32  = 2378
	DefaultToPortRangeEtcd                      int32  = 2379
	DefaultSecurityGroupRuleKcpBgpName          string = "cluster-api-securitygrouprule-kcp-bgp"
	DefaultFlowKcpBgp                           string = "Inbound"
	DefaultIpProtocolKcpBgp                     string = "tcp"
	DefaultRuleIpRangeKcpBgp                    string = "10.0.0.0/16"
	DefaultFromPortRangeKcpBgp                  int32  = 179
	DefaultToPortRangeKcpBgp                    int32  = 179
	DefaultSecurityGroupRuleKwBgpName           string = "cluster-api-securitygrouprule-kw-bgp"
	DefaultFlowKwBgp                            string = "Inbound"
	DefaultIpProtocolKwBgp                      string = "tcp"
	DefaultRuleIpRangeKwBgp                     string = "10.0.0.0/16"
	DefaultFromPortRangeKwBgp                   int32  = 179
	DefaultToPortRangeKwBgp                     int32  = 179
	DefaultSecurityGroupRuleKubeletKcpName      string = "cluster-api-securitygrouprule-kubelet-kcp"
	DefaultFlowKubeletKcp                       string = "Inbound"
	DefaultIpProtocolKubeletKcp                 string = "tcp"
	DefaultRuleIpRangeKubeletKcp                string = "10.0.4.0/24"
	DefaultFromPortRangeKubeletKcp              int32  = 10250
	DefaultToPortRangeKubeletKcp                int32  = 10252
	DefaultSecurityGroupPublicName              string = "cluster-api-securitygroup-lb"
	DefaultDescriptionLb                        string = "Security Group Lb with cluster-api"
	DefaultSecurityGroupRuleLbName              string = "cluster-api-securitygrouprule-lb"
	DefaultFlowLb                               string = "Inbound"
	DefaultIpProtocolLb                         string = "tcp"
	DefaultRuleIpRangeLb                        string = "0.0.0.0/0"
	DefaultFromPortRangeLb                      int32  = 6443
	DefaultToPortRangeLb                        int32  = 6443
	DefaultSecurityGroupNodeName                string = "cluster-api-securitygroup-node"
	DefaultDescriptionNode                      string = "Security Group Node with cluster-api"
	DefaultSecurityGroupRuleCalicoVxlanName     string = "cluster-api-securitygroup-calico-vxlan"
	DefaultFlowCalicoVxlan                      string = "Inbound"
	DefaultIpProtocolCalicoVxlan                string = "udp"
	DefaultRuleIpRangeCalicoVxlan               string = "10.0.0.0/16"
	DefaultFromPortRangeCalicoVxlan             int32  = 4789
	DefaultToPortRangeCalicoVxlan               int32  = 4789
	DefaultSecurityGroupRuleCalicoTypha         string = "cluster-api-securitygroup-typha"
	DefaultFlowCalicoTypha                      string = "Inbound"
	DefaultIpProtocolCalicoTypha                string = "udp"
	DefaultRuleIpRangeCalicoTypha               string = "10.0.0.0/16"
	DefaultFromPortRangeCalicoTypha             int32  = 5473
	DefaultToPortRangeCalicoTypha               int32  = 5473
	DefaultSecurityGroupRuleCalicoWireguard     string = "cluster-api-securitygroup-wireguard"
	DefaultFlowCalicoWireguard                  string = "Inbound"
	DefaultIpProtocolCalicoWireguard            string = "udp"
	DefaultRuleIpRangeCalicoWireguard           string = "10.0.0.0/16"
	DefaultFromPortRangeCalicoWireguard         int32  = 51820
	DefaultToPortRangeCalicoWireguard           int32  = 51820
	DefaultSecurityGroupRuleCalicoWireguardIpv6 string = "cluster-api-securitygroup-wireguard-ipv6"
	DefaultFlowCalicoWireguardIpv6              string = "Inbound"
	DefaultIpProtocolCalicoWireguardIpv6        string = "udp"
	DefaultRuleIpRangeCalicoWireguardIpv6       string = "10.0.0.0/16"
	DefaultFromPortRangeCalicoWireguardIpv6     int32  = 51821
	DefaultToPortRangeCalicoWireguardIpv6       int32  = 51821
	DefaultSecurityGroupRuleFlannel             string = "cluster-api-securitygroup-flannel"
	DefaultFlowFlannel                          string = "Inbound"
	DefaultIpProtocolFlannel                    string = "udp"
	DefaultRuleIpRangeFlannel                   string = "10.0.0.0/16"
	DefaultFromPortRangeFlannel                 int32  = 4789
	DefaultToPortRangeFlannel                   int32  = 4789
	DefaultSecurityGroupRuleFlannelUdp          string = "cluster-api-securitygroup-flannel-udp"
	DefaultFlowFlannelUdp                       string = "Inbound"
	DefaultIpProtocolFlannelUdp                 string = "udp"
	DefaultRuleIpRangeFlannelUdp                string = "10.0.0.0/16"
	DefaultFromPortRangeFlannelUdp              int32  = 8285
	DefaultToPortRangeFlannelUdp                int32  = 8285
	DefaultSecurityGroupRuleFlannelVxlan        string = "cluster-api-securityrgroup-flannel-vxlan"
	DefaultFlowFlannelVxlan                     string = "Inbound"
	DefaultIpProtocolFlannelVxlan               string = "udp"
	DefaultRuleIpRangeFlannelVxlan              string = "10.0.0.0/16"
	DefaultFromPortRangeFlannelVxlan            int32  = 8472
	DefaultToPortRangeFlannelVxlan              int32  = 8472
)

// SetDefaultValue set the Net default values
func (net *OscNet) SetDefaultValue() {
	var netName string = DefaultNetName
	if net.ClusterName != "" {
		netName = strings.Replace(DefaultNetName, DefaultClusterName, net.ClusterName, -1)
	}
	if net.IpRange == "" {
		net.IpRange = DefaultIpRange
	}
	if net.Name == "" {
		net.Name = netName
	}
}

// SetVolumeDefaultValue set the Volume default values from volume configuration
func (node *OscNode) SetVolumeDefaultValue() {
	if len(node.Volumes) == 0 {
		var volume OscVolume
		var volumeKcpName string = DefaultVolumeKcpName
		var volumeKwName string = DefaultVolumeKwName
		var volumeKcpSubregionName string = DefaultVolumeKcpSubregionName
		var volumeKwSubregionName string = DefaultVolumeKwSubregionName
		if node.ClusterName != "" {
			volumeKcpName = strings.Replace(DefaultVolumeKcpName, DefaultClusterName, node.ClusterName, -1)
			volumeKwName = strings.Replace(DefaultVolumeKwName, DefaultClusterName, node.ClusterName, -1)
			volumeKcpSubregionName = strings.Replace(DefaultVolumeKcpSubregionName, DefaultClusterName, node.ClusterName, -1)
			volumeKwSubregionName = strings.Replace(DefaultVolumeKwSubregionName, DefaultClusterName, node.ClusterName, -1)

		}
		if node.Vm.Role == "controlplane" {
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

// SetKeyPairDefaultValue set the KeyPair default values
func (node *OscNode) SetKeyPairDefaultValue() {
	if len(node.KeyPair.PublicKey) == 0 {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
		if err != nil {
			log.Fatal(err)
		}

		node.KeyPair.PublicKey = base64.StdEncoding.EncodeToString(ssh.MarshalAuthorizedKey(publicKey))
	}
	if len(node.KeyPair.Name) == 0 {
		node.KeyPair.Name = DefaultKeypairName
	}

}

// SetDefaultValue set the Internet Service default values
func (igw *OscInternetService) SetDefaultValue() {
	var internetServiceName string = DefaultInternetServiceName
	if igw.ClusterName != "" {
		internetServiceName = strings.Replace(DefaultInternetServiceName, DefaultClusterName, igw.ClusterName, -1)
	}
	if igw.Name == "" {
		igw.Name = internetServiceName
	}
}

// SetDefaultValue set the vm default values
func (vm *OscVm) SetDefaultValue() {
	var vmKcpName string = DefaultVmKcpName
	var vmKwName string = DefaultVmKwName
	var subnetKcpName string = DefaultSubnetKcpName
	var subnetKwName string = DefaultSubnetKwName
	var securityGroupKcpName string = DefaultSecurityGroupKcpName
	var securityGroupKwName string = DefaultSecurityGroupKwName
	var securityGroupNodeName string = DefaultSecurityGroupNodeName
	if vm.ClusterName != "" {
		vmKcpName = strings.Replace(DefaultVmKcpName, DefaultClusterName, vm.ClusterName, -1)
		vmKwName = strings.Replace(DefaultVmKwName, DefaultClusterName, vm.ClusterName, -1)
		subnetKcpName = strings.Replace(DefaultSubnetKcpName, DefaultClusterName, vm.ClusterName, -1)
		subnetKwName = strings.Replace(DefaultSubnetKwName, DefaultClusterName, vm.ClusterName, -1)
		securityGroupKcpName = strings.Replace(DefaultSecurityGroupKcpName, DefaultClusterName, vm.ClusterName, -1)
		securityGroupKwName = strings.Replace(DefaultSecurityGroupKwName, DefaultClusterName, vm.ClusterName, -1)
		securityGroupNodeName = strings.Replace(DefaultSecurityGroupNodeName, DefaultClusterName, vm.ClusterName, -1)
	}
	if vm.Role == "controlplane" {
		if vm.Name == "" {
			vm.Name = vmKcpName
		}
		if vm.VmType == "" {
			vm.VmType = DefaultVmKcpType
		}
		if vm.SubnetName == "" {
			vm.SubnetName = subnetKcpName
		}

		if vm.RootDisk.RootDiskIops == 0 && vm.RootDisk.RootDiskType == "io1" {
			vm.RootDisk.RootDiskIops = DefaultRootDiskKcpIops
		}

		if vm.RootDisk.RootDiskType == "" {
			vm.RootDisk.RootDiskType = DefaultRootDiskKcpType
		}

		if vm.RootDisk.RootDiskSize == 0 {
			vm.RootDisk.RootDiskSize = DefaultRootDiskKcpSize
		}

		if vm.LoadBalancerName == "" {
			vm.LoadBalancerName = DefaultLoadBalancerName
		}
		if len(vm.SecurityGroupNames) == 0 {
			securityGroupKw := OscSecurityGroupElement{
				Name: securityGroupKcpName,
			}
			securityGroupNode := OscSecurityGroupElement{
				Name: securityGroupNodeName,
			}
			vm.SecurityGroupNames = []OscSecurityGroupElement{securityGroupKw, securityGroupNode}
		}
	} else {
		if vm.Name == "" {
			vm.Name = vmKwName
		}
		if vm.VmType == "" {
			vm.VmType = DefaultVmKwType
		}

		if vm.RootDisk.RootDiskIops == 0 && vm.RootDisk.RootDiskType == "io1" {
			vm.RootDisk.RootDiskIops = DefaultRootDiskKwIops
		}

		if vm.RootDisk.RootDiskType == "" {
			vm.RootDisk.RootDiskType = DefaultRootDiskKwType
		}

		if vm.RootDisk.RootDiskSize == 0 {
			vm.RootDisk.RootDiskSize = DefaultRootDiskKwSize
		}

		if vm.SubnetName == "" {
			vm.SubnetName = subnetKwName
		}
		if len(vm.SecurityGroupNames) == 0 {
			securityGroupKw := OscSecurityGroupElement{
				Name: securityGroupKwName,
			}
			securityGroupNode := OscSecurityGroupElement{
				Name: securityGroupNodeName,
			}
			vm.SecurityGroupNames = []OscSecurityGroupElement{securityGroupKw, securityGroupNode}
		}
	}
	if vm.ImageId == "" {
		vm.ImageId = DefaultVmImageId
	}
	if vm.KeypairName == "" {
		vm.KeypairName = DefaultVmKeypairName
	}
	if vm.DeviceName == "" {
		vm.DeviceName = DefaultVmDeviceName
	}
	if vm.SubregionName == "" {
		vm.SubregionName = DefaultVmSubregionName
	}

}

// SetDefaultValue set the bastion default values
func (bastion *OscBastion) SetDefaultValue() {
	var vmBastionName string = DefaultVmBastionName
	var subnetPublicName string = DefaultSubnetPublicName
	var securityGroupPublicName string = DefaultSecurityGroupPublicName
	if bastion.Enable {
		if bastion.ClusterName != "" {
			vmBastionName = strings.Replace(DefaultVmBastionName, DefaultClusterName, bastion.ClusterName, -1)
			subnetPublicName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, bastion.ClusterName, -1)
			securityGroupPublicName = strings.Replace(DefaultSecurityGroupPublicName, DefaultClusterName, bastion.ClusterName, -1)
		}
		if bastion.Name == "" {
			bastion.Name = vmBastionName
		}
		if bastion.VmType == "" {
			bastion.VmType = DefaultVmBastionType
		}
		if bastion.RootDisk.RootDiskIops == 0 && bastion.RootDisk.RootDiskType == "io1" {
			bastion.RootDisk.RootDiskIops = DefaultRootDiskBastionIops
		}
		if bastion.RootDisk.RootDiskType == "" {
			bastion.RootDisk.RootDiskType = DefaultRootDiskBastionType
		}
		if bastion.SubnetName == "" {
			bastion.SubnetName = subnetPublicName
		}
		if len(bastion.SecurityGroupNames) == 0 {
			securityGroupPublic := OscSecurityGroupElement{
				Name: securityGroupPublicName,
			}
			bastion.SecurityGroupNames = []OscSecurityGroupElement{securityGroupPublic}
		}
		if bastion.ImageId == "" {
			bastion.ImageId = DefaultVmBastionImageId
		}
		if bastion.KeypairName == "" {
			bastion.KeypairName = DefaultVmBastionKeypairName
		}
		if bastion.DeviceName == "" {
			bastion.DeviceName = DefaultVmDeviceName
		}
		if bastion.SubregionName == "" {
			bastion.SubregionName = DefaultVmBastionSubregionName
		}
	}
}

// SetDefaultValue set the image default values
func (node *OscNode) SetImageDefaultValue() {
	if node.Image.Name == "" {
		node.Image.Name = DefaultVmImageId
	}
}

// SetDefaultValue set the Nat Service default values
func (nat *OscNatService) SetDefaultValue() {
	var natServiceName string = DefaultNatServiceName
	var publicIpNatName string = DefaultPublicIpNatName
	var subnetNatName string = DefaultSubnetPublicName
	if nat.ClusterName != "" {
		natServiceName = strings.Replace(DefaultNatServiceName, DefaultClusterName, nat.ClusterName, -1)
		publicIpNatName = strings.Replace(DefaultPublicIpNatName, DefaultClusterName, nat.ClusterName, -1)
		subnetNatName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, nat.ClusterName, -1)
	}
	if nat.Name == "" {
		nat.Name = natServiceName
	}
	if nat.PublicIpName == "" {
		nat.PublicIpName = publicIpNatName
	}
	if nat.SubnetName == "" {
		nat.SubnetName = subnetNatName
	}
}

// SetRouteTableDefaultValue set the Route Table default values from network configuration
func (network *OscNetwork) SetRouteTableDefaultValue() {
	if len(network.RouteTables) == 0 {
		var routeKwName string = DefaultRouteKwName
		var targetKwName string = DefaultTargetKwName
		var routeTableKwName string = DefaultRouteTableKwName
		var subnetKwName string = DefaultSubnetKwName

		var routeKcpName string = DefaultRouteKcpName
		var targetKcpName string = DefaultTargetKcpName
		var routeTableKcpName string = DefaultRouteTableKcpName
		var subnetKcpName string = DefaultSubnetKcpName

		var routePublicName string = DefaultRoutePublicName
		var targetPublicName string = DefaultTargetPublicName
		var routeTablePublicName string = DefaultRouteTablePublicName
		var subnetPublicName string = DefaultSubnetPublicName

		if network.ClusterName != "" {
			routeKwName = strings.Replace(DefaultRouteKwName, DefaultClusterName, network.ClusterName, -1)
			targetKwName = strings.Replace(DefaultTargetKwName, DefaultClusterName, network.ClusterName, -1)
			routeTableKwName = strings.Replace(DefaultRouteTableKwName, DefaultClusterName, network.ClusterName, -1)
			subnetKwName = strings.Replace(DefaultSubnetKwName, DefaultClusterName, network.ClusterName, -1)
			routeKcpName = strings.Replace(DefaultRouteKcpName, DefaultClusterName, network.ClusterName, -1)
			targetKcpName = strings.Replace(DefaultTargetKcpName, DefaultClusterName, network.ClusterName, -1)
			routeTableKcpName = strings.Replace(DefaultRouteTableKcpName, DefaultClusterName, network.ClusterName, -1)
			subnetKcpName = strings.Replace(DefaultSubnetKcpName, DefaultClusterName, network.ClusterName, -1)

			routePublicName = strings.Replace(DefaultRoutePublicName, DefaultClusterName, network.ClusterName, -1)
			targetPublicName = strings.Replace(DefaultTargetPublicName, DefaultClusterName, network.ClusterName, -1)
			routeTablePublicName = strings.Replace(DefaultRouteTablePublicName, DefaultClusterName, network.ClusterName, -1)
			subnetPublicName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, network.ClusterName, -1)
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

		subnetKw := []string{subnetKwName}
		subnetKcp := []string{subnetKcpName}
		subnetPublic := []string{subnetPublicName}
		routeTableKw := OscRouteTable{
			Name:   routeTableKwName,
			Routes: []OscRoute{routeKw},
		}

		network.RouteTables = append(network.RouteTables, &routeTableKw)
		routeTableKw.Subnets = subnetKw
		routeTableKcp := OscRouteTable{
			Name:   routeTableKcpName,
			Routes: []OscRoute{routeKcp},
		}
		network.RouteTables = append(network.RouteTables, &routeTableKcp)
		routeTableKcp.Subnets = subnetKcp

		routeTablePublic := OscRouteTable{
			Name:   routeTablePublicName,
			Routes: []OscRoute{routePublic},
		}
		network.RouteTables = append(network.RouteTables, &routeTablePublic)
		routeTablePublic.Subnets = subnetPublic
	}
}

// SetSecurityGroupDefaultValue set the security group default value
func (network *OscNetwork) SetSecurityGroupDefaultValue() {
	if len(network.SecurityGroups) == 0 {
		var securityGroupRuleApiKubeletKwName string = DefaultSecurityGroupRuleApiKubeletKwName
		var securityGroupRuleApiKubeletKcpName string = DefaultSecurityGroupRuleApiKubeletKcpName
		var securityGroupRuleKwNodeIpKwName string = DefaultSecurityGroupRuleKwNodeIpKwName
		var securityGroupRuleKcpNodeIpKwName string = DefaultSecurityGroupRuleKcpNodeIpKwName
		var securityGroupRuleKcpNodeIpKcpName string = DefaultSecurityGroupRuleKcpNodeIpKcpName
		var securityGroupRuleKwNodeIpKcpName string = DefaultSecurityGroupRuleKwNodeIpKcpName
		var securityGroupKwName string = DefaultSecurityGroupKwName
		var securityGroupRuleApiKwName string = DefaultSecurityGroupRuleApiKwName
		var securityGroupRuleApiKcpName string = DefaultSecurityGroupRuleApiKcpName
		var securityGroupRuleEtcdName string = DefaultSecurityGroupRuleEtcdName
		var securityGroupRuleKwBgpName string = DefaultSecurityGroupRuleKwBgpName
		var securityGroupRuleKcpBgpName string = DefaultSecurityGroupRuleKcpBgpName
		var securityGroupRuleKubeletKcpName string = DefaultSecurityGroupRuleKubeletKcpName
		var securityGroupKcpName string = DefaultSecurityGroupKcpName
		var securityGroupRuleLbName string = DefaultSecurityGroupRuleLbName
		var securityGroupLbName string = DefaultSecurityGroupPublicName
		var securityGroupNodeName string = DefaultSecurityGroupNodeName
		if network.ClusterName != "" {
			securityGroupRuleApiKubeletKwName = strings.Replace(DefaultSecurityGroupRuleApiKubeletKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKubeletKcpName = strings.Replace(DefaultSecurityGroupRuleApiKubeletKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKwNodeIpKwName = strings.Replace(DefaultSecurityGroupRuleKwNodeIpKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKcpNodeIpKwName = strings.Replace(DefaultSecurityGroupRuleKcpNodeIpKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKcpNodeIpKcpName = strings.Replace(DefaultSecurityGroupRuleKcpNodeIpKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKwNodeIpKcpName = strings.Replace(DefaultSecurityGroupRuleKwNodeIpKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupKwName = strings.Replace(DefaultSecurityGroupKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKwName = strings.Replace(DefaultSecurityGroupRuleApiKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKcpName = strings.Replace(DefaultSecurityGroupRuleApiKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleEtcdName = strings.Replace(DefaultSecurityGroupRuleEtcdName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKwBgpName = strings.Replace(DefaultSecurityGroupRuleKwBgpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKcpBgpName = strings.Replace(DefaultSecurityGroupRuleKcpBgpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKubeletKcpName = strings.Replace(DefaultSecurityGroupRuleKubeletKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupKcpName = strings.Replace(DefaultSecurityGroupKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleLbName = strings.Replace(DefaultSecurityGroupRuleLbName, DefaultClusterName, network.ClusterName, -1)
			securityGroupLbName = strings.Replace(DefaultSecurityGroupPublicName, DefaultClusterName, network.ClusterName, -1)
			securityGroupNodeName = strings.Replace(DefaultSecurityGroupNodeName, DefaultClusterName, network.ClusterName, -1)

		}
		securityGroupRuleApiKubeletKw := OscSecurityGroupRule{
			Name:          securityGroupRuleApiKubeletKwName,
			Flow:          DefaultFlowApiKubeletKw,
			IpProtocol:    DefaultIpProtocolApiKubeletKw,
			IpRange:       DefaultRuleIpRangeApiKubeletKw,
			FromPortRange: DefaultFromPortRangeApiKubeletKw,
			ToPortRange:   DefaultToPortRangeApiKubeletKw,
		}

		securityGroupRuleApiKubeletKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleApiKubeletKcpName,
			Flow:          DefaultFlowApiKubeletKcp,
			IpProtocol:    DefaultIpProtocolApiKubeletKcp,
			IpRange:       DefaultRuleIpRangeApiKubeletKcp,
			FromPortRange: DefaultFromPortRangeApiKubeletKcp,
			ToPortRange:   DefaultToPortRangeApiKubeletKcp,
		}

		securityGroupRuleKwNodeIpKw := OscSecurityGroupRule{
			Name:          securityGroupRuleKwNodeIpKwName,
			Flow:          DefaultFlowNodeIpKw,
			IpProtocol:    DefaultIpProtocolNodeIpKw,
			IpRange:       DefaultRuleIpRangeNodeIpKw,
			FromPortRange: DefaultFromPortRangeNodeIpKw,
			ToPortRange:   DefaultToPortRangeNodeIpKw,
		}

		securityGroupRuleKwNodeIpKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleKwNodeIpKcpName,
			Flow:          DefaultFlowNodeIpKcp,
			IpProtocol:    DefaultIpProtocolNodeIpKcp,
			IpRange:       DefaultRuleIpRangeNodeIpKcp,
			FromPortRange: DefaultFromPortRangeNodeIpKcp,
			ToPortRange:   DefaultToPortRangeNodeIpKcp,
		}

		securityGroupRuleKwBgp := OscSecurityGroupRule{
			Name:          securityGroupRuleKwBgpName,
			Flow:          DefaultFlowKwBgp,
			IpProtocol:    DefaultIpProtocolKwBgp,
			IpRange:       DefaultRuleIpRangeKwBgp,
			FromPortRange: DefaultFromPortRangeKwBgp,
			ToPortRange:   DefaultToPortRangeKwBgp,
		}

		securityGroupKw := OscSecurityGroup{
			Name:               securityGroupKwName,
			Description:        DefaultDescriptionKw,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleKwBgp, securityGroupRuleApiKubeletKw, securityGroupRuleKwNodeIpKcp, securityGroupRuleApiKubeletKcp, securityGroupRuleKwNodeIpKw},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupKw)

		securityGroupRuleApiKw := OscSecurityGroupRule{
			Name:          securityGroupRuleApiKwName,
			Flow:          DefaultFlowApiKw,
			IpProtocol:    DefaultIpProtocolApiKw,
			IpRange:       DefaultRuleIpRangeApiKw,
			FromPortRange: DefaultFromPortRangeApiKw,
			ToPortRange:   DefaultToPortRangeApiKw,
		}

		securityGroupRuleApiKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleApiKcpName,
			Flow:          DefaultFlowApiKcp,
			IpProtocol:    DefaultIpProtocolApiKcp,
			IpRange:       DefaultRuleIpRangeApiKcp,
			FromPortRange: DefaultFromPortRangeApiKcp,
			ToPortRange:   DefaultToPortRangeApiKcp,
		}

		securityGroupRuleEtcd := OscSecurityGroupRule{
			Name:          securityGroupRuleEtcdName,
			Flow:          DefaultFlowEtcd,
			IpProtocol:    DefaultIpProtocolEtcd,
			IpRange:       DefaultRuleIpRangeEtcd,
			FromPortRange: DefaultFromPortRangeEtcd,
			ToPortRange:   DefaultToPortRangeEtcd,
		}

		securityGroupRuleKubeletKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleKubeletKcpName,
			Flow:          DefaultFlowKubeletKcp,
			IpProtocol:    DefaultIpProtocolKubeletKcp,
			IpRange:       DefaultRuleIpRangeKubeletKcp,
			FromPortRange: DefaultFromPortRangeKubeletKcp,
			ToPortRange:   DefaultToPortRangeKubeletKcp,
		}

		securityGroupRuleKcpBgp := OscSecurityGroupRule{
			Name:          securityGroupRuleKcpBgpName,
			Flow:          DefaultFlowKcpBgp,
			IpProtocol:    DefaultIpProtocolKcpBgp,
			IpRange:       DefaultRuleIpRangeKcpBgp,
			FromPortRange: DefaultFromPortRangeKcpBgp,
			ToPortRange:   DefaultToPortRangeKcpBgp,
		}

		securityGroupRuleKcpNodeIpKw := OscSecurityGroupRule{
			Name:          securityGroupRuleKcpNodeIpKwName,
			Flow:          DefaultFlowNodeIpKw,
			IpProtocol:    DefaultIpProtocolNodeIpKw,
			IpRange:       DefaultRuleIpRangeNodeIpKw,
			FromPortRange: DefaultFromPortRangeNodeIpKw,
			ToPortRange:   DefaultToPortRangeNodeIpKw,
		}

		securityGroupRuleKcpNodeIpKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleKcpNodeIpKcpName,
			Flow:          DefaultFlowNodeIpKcp,
			IpProtocol:    DefaultIpProtocolNodeIpKcp,
			IpRange:       DefaultRuleIpRangeNodeIpKcp,
			FromPortRange: DefaultFromPortRangeNodeIpKcp,
			ToPortRange:   DefaultToPortRangeNodeIpKcp,
		}

		securityGroupKcp := OscSecurityGroup{
			Name:               securityGroupKcpName,
			Description:        DefaultDescriptionKcp,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleKcpBgp, securityGroupRuleApiKw, securityGroupRuleApiKcp, securityGroupRuleKcpNodeIpKw, securityGroupRuleEtcd, securityGroupRuleKubeletKcp, securityGroupRuleKcpNodeIpKcp},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupKcp)

		securityGroupRuleLb := OscSecurityGroupRule{
			Name:          securityGroupRuleLbName,
			Flow:          DefaultFlowLb,
			IpProtocol:    DefaultIpProtocolLb,
			IpRange:       DefaultRuleIpRangeLb,
			FromPortRange: DefaultFromPortRangeLb,
			ToPortRange:   DefaultToPortRangeLb,
		}
		securityGroupLb := OscSecurityGroup{
			Name:               securityGroupLbName,
			Description:        DefaultDescriptionLb,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleLb},
		}

		securityGroupRuleCalicoVxlan := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleCalicoVxlanName,
			Flow:          DefaultFlowCalicoVxlan,
			IpProtocol:    DefaultIpProtocolCalicoVxlan,
			IpRange:       DefaultRuleIpRangeCalicoVxlan,
			FromPortRange: DefaultFromPortRangeCalicoVxlan,
			ToPortRange:   DefaultToPortRangeCalicoVxlan,
		}

		securityGroupRuleCalicoTypha := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleCalicoTypha,
			Flow:          DefaultFlowCalicoTypha,
			IpProtocol:    DefaultIpProtocolCalicoTypha,
			IpRange:       DefaultRuleIpRangeCalicoTypha,
			FromPortRange: DefaultFromPortRangeCalicoTypha,
			ToPortRange:   DefaultToPortRangeCalicoTypha,
		}

		securityGroupRuleCalicoWireguard := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleCalicoWireguard,
			Flow:          DefaultFlowCalicoWireguard,
			IpProtocol:    DefaultIpProtocolCalicoWireguard,
			IpRange:       DefaultRuleIpRangeCalicoWireguard,
			FromPortRange: DefaultFromPortRangeCalicoWireguard,
			ToPortRange:   DefaultToPortRangeCalicoWireguard,
		}

		securityGroupRuleCalicoWireguardIpv6 := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleCalicoWireguardIpv6,
			Flow:          DefaultFlowCalicoWireguardIpv6,
			IpProtocol:    DefaultIpProtocolCalicoWireguardIpv6,
			IpRange:       DefaultRuleIpRangeCalicoWireguardIpv6,
			FromPortRange: DefaultFromPortRangeCalicoWireguardIpv6,
			ToPortRange:   DefaultToPortRangeCalicoWireguardIpv6,
		}

		securityGroupRuleFlannel := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleFlannel,
			Flow:          DefaultFlowFlannel,
			IpProtocol:    DefaultIpProtocolFlannel,
			IpRange:       DefaultRuleIpRangeFlannel,
			FromPortRange: DefaultFromPortRangeFlannel,
			ToPortRange:   DefaultToPortRangeFlannel,
		}

		securityGroupRuleFlannelUdp := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleFlannelUdp,
			Flow:          DefaultFlowFlannelUdp,
			IpProtocol:    DefaultIpProtocolFlannelUdp,
			IpRange:       DefaultRuleIpRangeFlannelUdp,
			FromPortRange: DefaultFromPortRangeFlannelUdp,
			ToPortRange:   DefaultToPortRangeFlannelUdp,
		}

		securityGroupRuleFlannelVxlan := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleFlannelVxlan,
			Flow:          DefaultFlowFlannelVxlan,
			IpProtocol:    DefaultIpProtocolFlannelVxlan,
			IpRange:       DefaultRuleIpRangeFlannelVxlan,
			FromPortRange: DefaultFromPortRangeFlannelVxlan,
			ToPortRange:   DefaultToPortRangeFlannelVxlan,
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupLb)
		securityGroupNode := OscSecurityGroup{
			Name:               securityGroupNodeName,
			Description:        DefaultDescriptionNode,
			Tag:                "OscK8sMainSG",
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleCalicoVxlan, securityGroupRuleCalicoTypha, securityGroupRuleCalicoWireguard, securityGroupRuleCalicoWireguardIpv6, securityGroupRuleFlannel, securityGroupRuleFlannelUdp, securityGroupRuleFlannelVxlan},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupNode)
	}
}

// SetPublicIpDefaultDefaultValue set the Public Ip default values from publicip configuration
func (network *OscNetwork) SetPublicIpDefaultValue() {
	if len(network.PublicIps) == 0 {
		var publicIpNatName string = DefaultPublicIpNatName
		if network.ClusterName != "" {
			publicIpNatName = strings.Replace(DefaultPublicIpNatName, DefaultClusterName, network.ClusterName, -1)
		}
		publicIp := OscPublicIp{
			Name: publicIpNatName,
		}
		network.PublicIps = append(network.PublicIps, &publicIp)
	}
}

// SetSubnetDefaultValue set the Subnet default values from subnet configuration
func (network *OscNetwork) SetSubnetDefaultValue() {
	if len(network.Subnets) == 0 {
		var subnetKcpName string = DefaultSubnetKcpName
		var subnetKwName string = DefaultSubnetKwName
		var subnetPublicName string = DefaultSubnetPublicName

		if network.ClusterName != "" {
			subnetKcpName = strings.Replace(subnetKcpName, DefaultClusterName, network.ClusterName, -1)
			subnetKwName = strings.Replace(subnetKwName, DefaultClusterName, network.ClusterName, -1)
			subnetPublicName = strings.Replace(subnetPublicName, DefaultClusterName, network.ClusterName, -1)
		}
		subnetKcp := OscSubnet{
			Name:          subnetKcpName,
			IpSubnetRange: DefaultIpSubnetKcpRange,
		}
		subnetKw := OscSubnet{
			Name:          subnetKwName,
			IpSubnetRange: DefaultIpSubnetKwRange,
		}
		subnetPublic := OscSubnet{
			Name:          subnetPublicName,
			IpSubnetRange: DefaultIpSubnetPublicRange,
		}
		network.Subnets = []*OscSubnet{
			&subnetKcp,
			&subnetKw,
			&subnetPublic,
		}
	}
}

// SetSubnetSubregionNameValue set the Subnet Subregion Name values from OscNetwork configuration
func (network *OscNetwork) SetSubnetSubregionNameDefaultValue() {
	var defaultSubregionName string = DefaultSubregionName
	if network.SubregionName != "" {
		defaultSubregionName = network.SubregionName
	}
	for _, subnet := range network.Subnets {
		if subnet.SubregionName == "" {
			subnet.SubregionName = defaultSubregionName
		}
	}
}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	var subnetPublicName string = DefaultSubnetPublicName
	var securityGroupLbName string = DefaultSecurityGroupPublicName
	if lb.ClusterName != "" {
		subnetPublicName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, lb.ClusterName, -1)
		securityGroupLbName = strings.Replace(DefaultSecurityGroupPublicName, DefaultClusterName, lb.ClusterName, -1)
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
