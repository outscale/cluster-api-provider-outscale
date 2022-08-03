package v1beta1

type OscNode struct {
	Vm      OscVm        `json:"vm,omitempty"`
	Image   OscImage     `json:"image,omitempty"`
	Volumes []*OscVolume `json:"volumes,omitempty"`
	KeyPair OscKeypair   `json:"keypair,omitempty"`
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
	PublicIps []*OscPublicIp `json:"publicIps,omitempty"`
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
	// The Net Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscInternetService struct {
	// The tag name associate with the Subnet
	// +optional
	Name string `json:"name,omitempty"`
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
	SubnetName string `json:"subnetname,omitempty"`
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
}

type OscPublicIp struct {
	// The tag name associate with the Public Ip
	// +optional
	Name string `json:"name,omitempty"`
	// The Public Ip Id response
	// +optional
	ResourceId string `json:"resourceId,omitempty"`
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
	Name       string `json:"name,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	ResourceId string `json:"resourceId,omitempty"`
}

type OscVm struct {
	Name               string                    `json:"name,omitempty"`
	ImageId            string                    `json:"imageId,omitempty"`
	KeypairName        string                    `json:"keypairName,omitempty"`
	VmType             string                    `json:"vmType,omitempty"`
	VolumeName         string                    `json:"volumeName,omitempty"`
	DeviceName         string                    `json:"deviceName,omitempty"`
	SubnetName         string                    `json:"subnetName,omitempty"`
	LoadBalancerName   string                    `json:"loadBalancerName,omitempty"`
	PublicIpName       string                    `json:"publicIpName,omitempty"`
	SubregionName      string                    `json:"subregionName,omitempty"`
	PrivateIps         []OscPrivateIpElement     `json:"privateIps,omitempty"`
	SecurityGroupNames []OscSecurityGroupElement `json:"securityGroupNames,omitempty"`
	ResourceId         string                    `json:"resourceId,omitempty"`
	Role               string                    `json:"role,omitempty"`
}

type VmState string

var (
	VmStatePending                      = VmState("pending")
	VmStateRunning                      = VmState("running")
	VmStateShuttingDown                 = VmState("shutting-down")
	VmStateTerminated                   = VmState("terminated")
	VmStateStopping                     = VmState("stopping")
	VmStateStopped                      = VmState("stopped")
	DefaultVmName                string = "cluster-api-vm"
	DefaultVmSubregionName       string = "eu-west-2a"
	DefaultVmImageId             string = "ami-bb490c7"
	DefaultVmKeypairName         string = "rke2"
	DefaultVmType                string = "tinav5.c4r8p1"
	DefaultVmDeviceName          string = "/dev/xvdb"
	DefaultVmPrivateIpName       string = "cluster-api-privateip"
	DefaultVmPrivateIp           string = "10.0.0.15"
	DefaultVolumeName            string = "cluster-api-volume"
	DefaultVolumeIops            int32  = 1000
	DefaultVolumeSize            int32  = 30
	DefaultVolumeSubregionName   string = "eu-west-2b"
	DefaultVolumeType            string = "gp2"
	DefaultLoadBalancerName      string = "OscClusterApi-1"
	DefaultLoadBalancerType      string = "internet-facing"
	DefaultBackendPort           int32  = 6443
	DefaultBackendProtocol       string = "TCP"
	DefaultLoadBalancerPort      int32  = 6443
	DefaultLoadBalancerProtocol  string = "TCP"
	DefaultCheckInterval         int32  = 5
	DefaultHealthyThreshold      int32  = 5
	DefaultUnhealthyThreshold    int32  = 2
	DefaultTimeout               int32  = 5
	DefaultProtocol              string = "TCP"
	DefaultPort                  int32  = 6443
	DefaultIpRange               string = "10.0.0.0/16"
	DefaultIpSubnetRange         string = "10.0.0.0/24"
	DefaultTargetName            string = "cluster-api-internetservice"
	DefaultTargetType            string = "gateway"
	DefaultDestination           string = "0.0.0.0/0"
	DefaultRouteTableName        string = "cluster-api-routetable"
	DefaultRouteName             string = "cluster-api-route"
	DefaultPublicIpName          string = "cluster-api-publicip"
	DefaultNatServiceName        string = "cluster-api-natservice"
	DefaultSubnetName            string = "cluster-api-subnet"
	DefaultNetName               string = "cluster-api-net"
	DefaultInternetServiceName   string = "cluster-api-internetservice"
	DefaultSecurityGroupName     string = "cluster-api-securitygroup"
	DefaultDescription           string = "Security Group with cluster-api"
	DefaultSecurityGroupRuleName string = "cluster-api-securitygrouprule"
	DefaultFlow                  string = "Inbound"
	DefaultIpProtocol            string = "tcp"
	DefaultRuleIpRange           string = "0.0.0.0/0"
	DefaultFromPortRange         int32  = 6443
	DefaultToPortRange           int32  = 6443
)

// SetDefaultValue set the Net default values
func (net *OscNet) SetDefaultValue() {
	if net.IpRange == "" {
		net.IpRange = DefaultIpRange
	}
	if net.Name == "" {
		net.Name = DefaultNetName
	}
}

// SetVolumeDefaultValue set the Volume default values from volume configuration
func (node *OscNode) SetVolumeDefaultValue() {
	if len(node.Volumes) == 0 {
		volume := OscVolume{
			Name:          DefaultVolumeName,
			Iops:          DefaultVolumeIops,
			Size:          DefaultVolumeSize,
			SubregionName: DefaultVolumeSubregionName,
			VolumeType:    DefaultVolumeType,
		}
		node.Volumes = append(node.Volumes, &volume)
	}
}

// SetDefaultValue set the Internet Service default values
func (igw *OscInternetService) SetDefaultValue() {
	if igw.Name == "" {
		igw.Name = DefaultInternetServiceName
	}
}

// SetDefaultValue set the vm default values
func (vm *OscVm) SetDefaultValue() {
	if vm.Name == "" {
		vm.Name = DefaultVmName
	}
	if vm.ImageId == "" {
		vm.ImageId = DefaultVmImageId
	}
	if vm.KeypairName == "" {
		vm.KeypairName = DefaultVmKeypairName
	}
	if vm.VmType == "" {
		vm.VmType = DefaultVmType
	}
	if vm.VolumeName == "" {
		vm.VolumeName = DefaultVolumeName
	}
	if vm.DeviceName == "" {
		vm.DeviceName = DefaultVmDeviceName
	}
	if vm.SubregionName == "" {
		vm.SubregionName = DefaultVmSubregionName
	}
	if vm.SubnetName == "" {
		vm.SubnetName = DefaultSubnetName
	}
	if vm.Role == "controlplane" && vm.LoadBalancerName == "" {
		vm.LoadBalancerName = DefaultLoadBalancerName
	}
	if len(vm.SecurityGroupNames) == 0 {
		securityGroup := OscSecurityGroupElement{
			Name: DefaultSecurityGroupName,
		}
		vm.SecurityGroupNames = []OscSecurityGroupElement{securityGroup}
	}
	if len(vm.PrivateIps) == 0 {
		privateIp := OscPrivateIpElement{
			Name:      DefaultVmPrivateIpName,
			PrivateIp: DefaultVmPrivateIp,
		}
		vm.PrivateIps = []OscPrivateIpElement{privateIp}
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
	if nat.Name == "" {
		nat.Name = DefaultNatServiceName
	}
	if nat.PublicIpName == "" {
		nat.PublicIpName = DefaultPublicIpName
	}
	if nat.SubnetName == "" {
		nat.SubnetName = DefaultSubnetName
	}
}

// SetRouteTableDefaultValue set the Route Table default values from network configuration
func (network *OscNetwork) SetRouteTableDefaultValue() {
	if len(network.RouteTables) == 0 {
		route := OscRoute{
			Name:        DefaultRouteName,
			TargetName:  DefaultTargetName,
			TargetType:  DefaultTargetType,
			Destination: DefaultDestination,
		}
		routeTable := OscRouteTable{
			Name:       DefaultRouteTableName,
			SubnetName: DefaultSubnetName,
			Routes:     []OscRoute{route},
		}
		network.RouteTables = append(network.RouteTables, &routeTable)
	}

}

// SetSecurityGroupDefaultValue set the security group default value

func (network *OscNetwork) SetSecurityGroupDefaultValue() {
	if len(network.SecurityGroups) == 0 {
		securityGroupRule := OscSecurityGroupRule{
			Name:          DefaultSecurityGroupRuleName,
			Flow:          DefaultFlow,
			IpProtocol:    DefaultIpProtocol,
			IpRange:       DefaultRuleIpRange,
			FromPortRange: DefaultFromPortRange,
			ToPortRange:   DefaultToPortRange,
		}
		securityGroup := OscSecurityGroup{
			Name:               DefaultSecurityGroupName,
			Description:        DefaultDescription,
			SecurityGroupRules: []OscSecurityGroupRule{securityGroupRule},
		}
		network.SecurityGroups = append(network.SecurityGroups, &securityGroup)
	}
}

// SetPublicIpDefaultDefaultValue set the Public Ip default values from publicip configuration
func (network *OscNetwork) SetPublicIpDefaultValue() {
	if len(network.PublicIps) == 0 {
		publicIp := OscPublicIp{
			Name: DefaultPublicIpName,
		}
		network.PublicIps = append(network.PublicIps, &publicIp)
	}
}

// SetSubnetDefaultValue set the Subnet default values from subnet configuration
func (network *OscNetwork) SetSubnetDefaultValue() {
	if len(network.Subnets) == 0 {
		subnet := OscSubnet{
			Name:          DefaultSubnetName,
			IpSubnetRange: DefaultIpSubnetRange,
		}
		network.Subnets = []*OscSubnet{
			&subnet,
		}
	}

}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	if lb.LoadBalancerName == "" {
		lb.LoadBalancerName = DefaultLoadBalancerName
	}
	if lb.LoadBalancerType == "" {
		lb.LoadBalancerType = DefaultLoadBalancerType
	}
	if lb.SubnetName == "" {
		lb.SubnetName = DefaultSubnetName
	}
	if lb.SecurityGroupName == "" {
		lb.SecurityGroupName = DefaultSecurityGroupName
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
