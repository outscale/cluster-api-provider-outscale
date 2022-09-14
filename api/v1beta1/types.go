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
	PublicIps   []*OscPublicIp `json:"publicIps,omitempty"`
	ClusterName string         `json:"clusterName,omitempty"`
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
	IpRange     string `json:"ipRange,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
	// The Net Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscInternetService struct {
	// The tag name associate with the Subnet
	// +optional
	Name        string `json:"name,omitempty"`
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
	SubnetName  string `json:"subnetname,omitempty"`
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
	// +optional
	SubnetName string `json:"subnetname,omitempty"`
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
type OscResourceMapReference struct {
	ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

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
	// Map between PublicIpId  and PublicIpName (Public IP tag Name with cluster UID)
	SecurityGroupsRef    OscResourceMapReference `json:"securitygroupref,omitempty"`
	SecurityGroupRuleRef OscResourceMapReference `json:"securitygroupruleref,omitempty"`
	PublicIpRef          OscResourceMapReference `json:"publicipref,omitempty"`
	// Map between NatServiceId  and NatServiceName (Nat Service tag Name with cluster UID)
	NatServiceRef OscResourceMapReference `json:"natref,omitempty"`
}

type OscNodeResource struct {
	VolumeRef       OscResourceMapReference `json:"volumeRef,omitempty"`
	ImageRef        OscResourceMapReference `json:"imageRef,omitempty"`
	KeypairRef      OscResourceMapReference `json:"keypairRef,omitempty"`
	VmRef           OscResourceMapReference `json:"vmRef,omitempty"`
	LinkPublicIpRef OscResourceMapReference `json:"linkPublicIpRef,omitempty"`
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
	Name        string `json:"name,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ResourceId  string `json:"resourceId,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
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
	SubregionName      string                    `json:"subregionName,omitempty"`
	PrivateIps         []OscPrivateIpElement     `json:"privateIps,omitempty"`
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	ResourceId         string                    `json:"resourceId,omitempty"`
	Role               string                    `json:"role,omitempty"`
	ClusterName        string                    `json:"clusterName,omitempty"`
	Replica            int32                     `json:"replica,omitempty"`
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

	DefaultVmName             string = "cluster-api-vm"
	DefaultVmSubregionName    string = "eu-west-2a"
	DefaultVmImageId          string = "ami-e1a786f1"
	DefaultVmKeypairName      string = "cluster-api"
	DefaultVmType             string = "tinav5.c2r2p1"
	DefaultVmDeviceName       string = "/dev/sda1"
	DefaultVmPrivateIpKwName  string = "cluster-api-privateip-kw"
	DefaultVmPrivateIpKcpName string = "cluster-api-privateip-kcp"
	DefaultVmPrivateIpKcp     string = "10.0.0.38"
	DefaultVmPrivateIpKw      string = "10.0.0.138"

	DefaultVmKwName  string = "cluster-api-vm-kw"
	DefaultVmKwType  string = "tinav5.c4r8p1"
	DefaultVmKcpName string = "cluster-api-vm-kcp"
	DefaultVmKcpType string = "tinav5.c4r8p1"

	DefaultVolumeKcpName          string = "cluster-api-volume-kcp"
	DefaultVolumeKcpIops          int32  = 1000
	DefaultVolumeKcpSize          int32  = 30
	DefaultVolumeKcpSubregionName string = "eu-west-2a"
	DefaultVolumeKcpType          string = "io1"

	DefaultRootDiskKwType string = "io1"
	DefaultRootDiskKwSize int32  = 60
	DefaultRootDiskKwIops int32  = 1500

	DefaultRootDiskKcpType string = "io1"
	DefaultRootDiskKcpSize int32  = 60
	DefaultRootDiskKcpIops int32  = 1500

	DefaultVolumeKwName          string = "cluster-api-volume-kw"
	DefaultVolumeKwIops          int32  = 1000
	DefaultVolumeKwSize          int32  = 30
	DefaultVolumeKwSubregionName string = "eu-west-2a"
	DefaultVolumeKwType          string = "io1"

	DefaultLoadBalancerName                   string = "OscClusterApi-1"
	DefaultLoadBalancerType                   string = "internet-facing"
	DefaultBackendPort                        int32  = 6443
	DefaultBackendProtocol                    string = "TCP"
	DefaultLoadBalancerPort                   int32  = 6443
	DefaultLoadBalancerProtocol               string = "TCP"
	DefaultCheckInterval                      int32  = 5
	DefaultHealthyThreshold                   int32  = 5
	DefaultUnhealthyThreshold                 int32  = 2
	DefaultTimeout                            int32  = 5
	DefaultProtocol                           string = "TCP"
	DefaultPort                               int32  = 6443
	DefaultIpRange                            string = "10.0.0.0/24"
	DefaultIpSubnetRange                      string = "10.0.0.0/24"
	DefaultIpSubnetKcpRange                   string = "10.0.0.32/28"
	DefaultIpSubnetKwRange                    string = "10.0.0.128/26"
	DefaultIpSubnetPublicRange                string = "10.0.0.8/29"
	DefaultIpSubnetNatRange                   string = "10.0.0.0/29"
	DefaultTargetType                         string = "gateway"
	DefaultTargetKwName                       string = "cluster-api-natservice"
	DefaultTargetKwType                       string = "nat"
	DefaultDestinationKw                      string = "0.0.0.0/0"
	DefaultRouteTableKwName                   string = "cluster-api-routetable-kw"
	DefaultRouteKwName                        string = "cluster-api-route-kw"
	DefaultTargetKcpName                      string = "cluster-api-natservice"
	DefaultTargetKcpType                      string = "nat"
	DefaultDestinationKcp                     string = "0.0.0.0/0"
	DefaultRouteTableKcpName                  string = "cluster-api-routetable-kcp"
	DefaultRouteKcpName                       string = "cluster-api-route-kcp"
	DefaultTargetPublicName                   string = "cluster-api-internetservice"
	DefaultTargetPublicType                   string = "gateway"
	DefaultDestinationPublic                  string = "0.0.0.0/0"
	DefaultRouteTablePublicName               string = "cluster-api-routetable-public"
	DefaultRoutePublicName                    string = "cluster-api-route-public"
	DefaultTargetNatName                      string = "cluster-api-internetservice"
	DefaultTargetNatType                      string = "gateway"
	DefaultDestinationNat                     string = "0.0.0.0/0"
	DefaultRouteTableNatName                  string = "cluster-api-routetable-nat"
	DefaultRouteNatName                       string = "cluster-api-route-nat"
	DefaultPublicIpNatName                    string = "cluster-api-publicip-nat"
	DefaultNatServiceName                     string = "cluster-api-natservice"
	DefaultSubnetName                         string = "cluster-api-subnet"
	DefaultSubnetKcpName                      string = "cluster-api-subnet-kcp"
	DefaultSubnetKwName                       string = "cluster-api-subnet-kw"
	DefaultSubnetPublicName                   string = "cluster-api-subnet-public"
	DefaultSubnetNatName                      string = "cluster-api-subnet-nat"
	DefaultNetName                            string = "cluster-api-net"
	DefaultInternetServiceName                string = "cluster-api-internetservice"
	DefaultSecurityGroupKwName                string = "cluster-api-securitygroup-kw"
	DefaultDescriptionKw                      string = "Security Group Kw with cluster-api"
	DefaultSecurityGroupRuleApiKubeletKwName  string = "cluster-api-securitygrouprule-api-kubelet-kw"
	DefaultFlowApiKubeletKw                   string = "Inbound"
	DefaultIpProtocolApiKubeletKw             string = "tcp"
	DefaultRuleIpRangeApiKubeletKw            string = "10.0.0.128/26"
	DefaultFromPortRangeApiKubeletKw          int32  = 10250
	DefaultToPortRangeApiKubeletKw            int32  = 10250
	DefaultSecurityGroupRuleApiKubeletKcpName string = "cluster-api-securitygrouprule-api-kubelet-kcp"
	DefaultFlowApiKubeletKcp                  string = "Inbound"
	DefaultIpProtocolApiKubeletKcp            string = "tcp"
	DefaultRuleIpRangeApiKubeletKcp           string = "10.0.0.32/28"
	DefaultFromPortRangeApiKubeletKcp         int32  = 10250
	DefaultToPortRangeApiKubeletKcp           int32  = 10250
	DefaultSecurityGroupRuleNodeIpKwName      string = "cluster-api-securitygrouprule-nodeip-kw"
	DefaultFlowNodeIpKw                       string = "Inbound"
	DefaultIpProtocolNodeIpKw                 string = "tcp"
	DefaultRuleIpRangeNodeIpKw                string = "10.0.0.128/26"
	DefaultFromPortRangeNodeIpKw              int32  = 30000
	DefaultToPortRangeNodeIpKw                int32  = 32767
	DefaultSecurityGroupRuleNodeIpKcpName     string = "cluster-api-securitygrouprule-nodeip-kcp"
	DefaultFlowNodeIpKcp                      string = "Inbound"
	DefaultIpProtocolNodeIpKcp                string = "tcp"
	DefaultRuleIpRangeNodeIpKcp               string = "10.0.0.32/28"
	DefaultFromPortRangeNodeIpKcp             int32  = 30000
	DefaultToPortRangeNodeIpKcp               int32  = 32767
	DefaultSecurityGroupKcpName               string = "cluster-api-securitygroup-kcp"
	DefaultDescriptionKcp                     string = "Security Group Kcp with cluster-api"
	DefaultSecurityGroupRuleApiKwName         string = "cluster-api-securitygrouprule-api-kw"
	DefaultFlowApiKw                          string = "Inbound"
	DefaultIpProtocolApiKw                    string = "tcp"
	DefaultRuleIpRangeApiKw                   string = "10.0.0.128/26"
	DefaultFromPortRangeApiKw                 int32  = 6443
	DefaultToPortRangeApiKw                   int32  = 6443
	DefaultSecurityGroupRuleApiKcpName        string = "cluster-api-securitygrouprule-api-kcp"
	DefaultFlowApiKcp                         string = "Inbound"
	DefaultIpProtocolApiKcp                   string = "tcp"
	DefaultRuleIpRangeApiKcp                  string = "10.0.0.32/28"
	DefaultFromPortRangeApiKcp                int32  = 6443
	DefaultToPortRangeApiKcp                  int32  = 6443
	DefaultSecurityGroupRuleEtcdName          string = "cluster-api-securitygrouprule-etcd"
	DefaultFlowEtcd                           string = "Inbound"
	DefaultIpProtocolEtcd                     string = "tcp"
	DefaultRuleIpRangeEtcd                    string = "10.0.0.32/28"
	DefaultFromPortRangeEtcd                  int32  = 2378
	DefaultToPortRangeEtcd                    int32  = 2379
	DefaultSecurityGroupRuleKcpBgpName        string = "cluster-api-securitygrouprule-kcp-bgp"
	DefaultFlowKcpBgp                         string = "Inbound"
	DefaultIpProtocolKcpBgp                   string = "tcp"
	DefaultRuleIpRangeKcpBgp                  string = "10.0.0.0/24"
	DefaultFromPortRangeKcpBgp                int32  = 179
	DefaultToPortRangeKcpBgp                  int32  = 179
	DefaultSecurityGroupRuleKwBgpName         string = "cluster-api-securitygrouprule-kw-bgp"
	DefaultFlowKwBgp                          string = "Inbound"
	DefaultIpProtocolKwBgp                    string = "tcp"
	DefaultRuleIpRangeKwBgp                   string = "10.0.0.0/24"
	DefaultFromPortRangeKwBgp                 int32  = 179
	DefaultToPortRangeKwBgp                   int32  = 179
	DefaultSecurityGroupRuleKubeletKcpName    string = "cluster-api-securitygrouprule-kubelet-kcp"
	DefaultFlowKubeletKcp                     string = "Inbound"
	DefaultIpProtocolKubeletKcp               string = "tcp"
	DefaultRuleIpRangeKubeletKcp              string = "10.0.0.32/28"
	DefaultFromPortRangeKubeletKcp            int32  = 10250
	DefaultToPortRangeKubeletKcp              int32  = 10252
	DefaultSecurityGroupLbName                string = "cluster-api-securitygroup-lb"
	DefaultDescriptionLb                      string = "Security Group Lb with cluster-api"
	DefaultSecurityGroupRuleLbName            string = "cluster-api-securitygrouprule-lb"
	DefaultFlowLb                             string = "Inbound"
	DefaultIpProtocolLb                       string = "tcp"
	DefaultRuleIpRangeLb                      string = "0.0.0.0/0"
	DefaultFromPortRangeLb                    int32  = 6443
	DefaultToPortRangeLb                      int32  = 6443
	DefaultSecurityGroupNodeName              string = "cluster-api-securitygroup-node"
	DefaultDescriptionNode                    string = "Security Group Node with cluster-api"
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

// SetDefaultValue set the Subnet default values
func (sub *OscSubnet) SetDefaultValue() {
	if sub.IpSubnetRange == "" {
		sub.IpSubnetRange = DefaultIpSubnetRange
	}
	if sub.Name == "" {
		sub.Name = DefaultSubnetName
	}
}

// SetDefaultValue set the Nat Service default values
func (nat *OscNatService) SetDefaultValue() {
	var natServiceName string = DefaultNatServiceName
	var publicIpNatName string = DefaultPublicIpNatName
	var subnetNatName string = DefaultSubnetNatName
	if nat.ClusterName != "" {
		natServiceName = strings.Replace(DefaultNatServiceName, DefaultClusterName, nat.ClusterName, -1)
		publicIpNatName = strings.Replace(DefaultPublicIpNatName, DefaultClusterName, nat.ClusterName, -1)
		subnetNatName = strings.Replace(DefaultSubnetNatName, DefaultClusterName, nat.ClusterName, -1)
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

		var routeNatName string = DefaultRouteNatName
		var targetNatName string = DefaultTargetNatName
		var routeTableNatName string = DefaultRouteTableNatName
		var subnetNatName string = DefaultSubnetNatName
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
			routeNatName = strings.Replace(DefaultRouteNatName, DefaultClusterName, network.ClusterName, -1)
			targetNatName = strings.Replace(DefaultTargetNatName, DefaultClusterName, network.ClusterName, -1)
			routeTableNatName = strings.Replace(DefaultRouteTableNatName, DefaultClusterName, network.ClusterName, -1)
			subnetNatName = strings.Replace(DefaultSubnetNatName, DefaultClusterName, network.ClusterName, -1)
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

// SetSecurityGroupDefaultValue set the security group default value
func (network *OscNetwork) SetSecurityGroupDefaultValue() {
	if len(network.SecurityGroups) == 0 {
		var securityGroupRuleApiKubeletKwName string = DefaultSecurityGroupRuleApiKubeletKwName
		var securityGroupRuleApiKubeletKcpName string = DefaultSecurityGroupRuleApiKubeletKcpName
		var securityGroupRuleNodeIpKwName string = DefaultSecurityGroupRuleNodeIpKwName
		var securityGroupRuleNodeIpKcpName string = DefaultSecurityGroupRuleNodeIpKcpName
		var securityGroupKwName string = DefaultSecurityGroupKwName
		var securityGroupRuleApiKwName string = DefaultSecurityGroupRuleApiKwName
		var securityGroupRuleApiKcpName string = DefaultSecurityGroupRuleApiKcpName
		var securityGroupRuleEtcdName string = DefaultSecurityGroupRuleEtcdName
		var securityGroupRuleKwBgpName string = DefaultSecurityGroupRuleKwBgpName
		var securityGroupRuleKcpBgpName string = DefaultSecurityGroupRuleKcpBgpName
		var securityGroupRuleKubeletKcpName string = DefaultSecurityGroupRuleKubeletKcpName
		var securityGroupKcpName string = DefaultSecurityGroupKcpName
		var securityGroupRuleLbName string = DefaultSecurityGroupRuleLbName
		var securityGroupLbName string = DefaultSecurityGroupLbName
		var securityGroupNodeName string = DefaultSecurityGroupNodeName
		if network.ClusterName != "" {
			securityGroupRuleApiKubeletKwName = strings.Replace(DefaultSecurityGroupRuleApiKubeletKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKubeletKcpName = strings.Replace(DefaultSecurityGroupRuleApiKubeletKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleNodeIpKwName = strings.Replace(DefaultSecurityGroupRuleNodeIpKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleNodeIpKcpName = strings.Replace(DefaultSecurityGroupRuleNodeIpKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupKwName = strings.Replace(DefaultSecurityGroupKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKwName = strings.Replace(DefaultSecurityGroupRuleApiKwName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleApiKcpName = strings.Replace(DefaultSecurityGroupRuleApiKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleEtcdName = strings.Replace(DefaultSecurityGroupRuleEtcdName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKwBgpName = strings.Replace(DefaultSecurityGroupRuleKwBgpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKcpBgpName = strings.Replace(DefaultSecurityGroupRuleKcpBgpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleKubeletKcpName = strings.Replace(DefaultSecurityGroupRuleKubeletKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupKcpName = strings.Replace(DefaultSecurityGroupKcpName, DefaultClusterName, network.ClusterName, -1)
			securityGroupRuleLbName = strings.Replace(DefaultSecurityGroupRuleLbName, DefaultClusterName, network.ClusterName, -1)
			securityGroupLbName = strings.Replace(DefaultSecurityGroupLbName, DefaultClusterName, network.ClusterName, -1)
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

		securityGroupRuleNodeIpKw := OscSecurityGroupRule{
			Name:          securityGroupRuleNodeIpKwName,
			Flow:          DefaultFlowNodeIpKw,
			IpProtocol:    DefaultIpProtocolNodeIpKw,
			IpRange:       DefaultRuleIpRangeNodeIpKw,
			FromPortRange: DefaultFromPortRangeNodeIpKw,
			ToPortRange:   DefaultToPortRangeNodeIpKw,
		}

		securityGroupRuleNodeIpKcp := OscSecurityGroupRule{
			Name:          securityGroupRuleNodeIpKcpName,
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
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleApiKubeletKw, securityGroupRuleApiKubeletKcp, securityGroupRuleNodeIpKw, securityGroupRuleNodeIpKcp, securityGroupRuleKwBgp},
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

		securityGroupKcp := OscSecurityGroup{
			Name:               securityGroupKcpName,
			Description:        DefaultDescriptionKcp,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRuleApiKw, securityGroupRuleApiKcp, securityGroupRuleEtcd, securityGroupRuleKubeletKcp, securityGroupRuleKcpBgp},
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
		network.SecurityGroups = append(network.SecurityGroups, &securityGroupLb)
		securityGroupNode := OscSecurityGroup{
			Name:               securityGroupNodeName,
			Description:        DefaultDescriptionNode,
			SecurityGroupRules: []OscSecurityGroupRule{},
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
		var subnetNatName string = DefaultSubnetNatName
		if network.ClusterName != "" {
			subnetKcpName = strings.Replace(DefaultSubnetKcpName, DefaultClusterName, network.ClusterName, -1)
			subnetKwName = strings.Replace(DefaultSubnetKwName, DefaultClusterName, network.ClusterName, -1)
			subnetPublicName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, network.ClusterName, -1)
			subnetNatName = strings.Replace(DefaultSubnetNatName, DefaultClusterName, network.ClusterName, -1)
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
		subnetNat := OscSubnet{
			Name:          subnetNatName,
			IpSubnetRange: DefaultIpSubnetNatRange,
		}
		network.Subnets = []*OscSubnet{
			&subnetKcp,
			&subnetKw,
			&subnetPublic,
			&subnetNat,
		}
	}
}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	var subnetPublicName string = DefaultSubnetPublicName
	var securityGroupLbName string = DefaultSecurityGroupLbName
	if lb.ClusterName != "" {
		subnetPublicName = strings.Replace(DefaultSubnetPublicName, DefaultClusterName, lb.ClusterName, -1)
		securityGroupLbName = strings.Replace(DefaultSecurityGroupLbName, DefaultClusterName, lb.ClusterName, -1)
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
