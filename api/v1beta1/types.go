package v1beta1

type OscNetwork struct {
        // The Load Balancer configuration
        // +optional
	LoadBalancer    OscLoadBalancer    `json:"loadBalancer,omitempty"`
        // The Net configuration
        // +optional
	Net             OscNet             `json:"net,omitempty"`
        // The Subnet configuration
        // +optional
	Subnets         []*OscSubnet       `json:"subnets,omitempty"`
        // The Internet Service configuration
        // +optional
	InternetService OscInternetService `json:"internetService,omitempty"`
        // The Nat Service configuration
        // +optional
	NatService      OscNatService      `json:"natService,omitempty"`
        // The Route Table configuration
        // +optional
	RouteTables     []*OscRouteTable   `json:"routeTables,omitempty"`
        // The Public Ip configuration
        // +optional
	PublicIps       []*OscPublicIp     `json:"publicIps,omitempty"`
}

type OscLoadBalancer struct {
        // The Load Balancer unique name
        // +optional
	LoadBalancerName string                     `json:"loadbalancername,omitempty"`
        // The SubRegion Name where the Load Balancer will be created
        // +optional 
	SubregionName    string                     `json:"subregionname,omitempty"`
        // The listener configuration of the Load Balancer
        // +optional
	Listener         OscLoadBalancerListener    `json:"listener,omitempty"`
        // The healthCheck configuration  of the Load Balancer
        // +optional
	HealthCheck      OscLoadBalancerHealthCheck `json:"healthCheck,omitempty"`
}

type OscLoadBalancerListener struct {
        // The port on which the backend vm will listen
        // +optional
	BackendPort          int32  `json:"backendport,omitempty"`
        // The protocol ('HTTP'|'TCP') to route the traffic to the backend vm 
        // +optional
	BackendProtocol      string `json:"backendprotocol,omitempty"`
        // The port on which the loadbalancer will listen
        // +optional
	LoadBalancerPort     int32  `json:"loadbalancerport,omitempty"`
        // the routing protocol ('HTTP'|'TCP')
        // +optional
	LoadBalancerProtocol string `json:"loadbalancerprotocol,omiempty"`
}

type OscLoadBalancerHealthCheck struct {
        // the time in second between two pings
        // +optional
	CheckInterval      int32  `json:"checkinterval,omitempty"`
        // the consecutive number of pings which are sucessful to consider the vm healthy
        // +optional
	HealthyThreshold   int32  `json:"healthythreshold,omitempty"`
        // the HealthCheck port number
        // +optional
	Port               int32  `json:"port,omitempty"`
        // The HealthCheck protocol ('HTTP'|'TCP')
        // +optional
	Protocol           string `json:"protocol,omitepty"`
        // the Timeout to consider VM unhealthy
        // +optional
	Timeout            int32  `json:"timeout,omitempty"`
        // the consecutive number of pings which are failed to consider the vm unhealthy
        // +optional
	UnhealthyThreshold int32  `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
        // Name Tag associated with the Net
        // +optional
	Name       string `json:"name,omitempty"`
        // Net Ip range with CIDR notation
        // +optional
	IpRange    string `json:"ipRange,omitempty"`
        // The Net Id response
        // +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscInternetService struct {
        // The tag name associated with the Subnet
        // +optional
	Name       string `json:"name,omitempty"`
        // the Internet Service response
        // +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscSubnet struct {
        // The tag name associated with the Subnet
        // +optional
	Name          string `json:"name,omitempty"`
        // Subnet Ip range with CIDR notation
        // +optional
	IpSubnetRange string `json:"ipSubnetRange,omitempty"`
        // The Subnet Id response
        // +optional
	ResourceId    string `json:"resourceId,omitempty"`
}

type OscNatService struct {
        // The tag name associated with the Nat Service
        // +optional
	Name         string `json:"name,omitempty"`
        // The Public Ip tag name associated wtih a Public Ip
        // +optional
	PublicIpName string `json:"publicipname,omitempty"`
        // The subnet tag name associated with a Subnet
        // +optional
	SubnetName   string `json:"subnetname,omitempty"`
        // The Nat Service Id response
        // +optional
	ResourceId   string `json:"resourceId,omitempty"`
}

type OscRouteTable struct {
        // The tag name associated with the Route Table
        // +optional
	Name       string     `json:"name,omitempty"`
        // The subnet tag name associated with a Subnet
        // +optional
	SubnetName string     `json:"subnetname,omitempty"`
        // The Route configuration
        // +optional
	Routes     []OscRoute `json:"routes,omitempty"`
        // The Route Table Id response
        // +optional
	ResourceId string     `json:"resourceId,omitempty"`
}

type OscPublicIp struct {
        // The tag name associated with the Public Ip
        // +optional
	Name       string `json:"name,omitempty"`
        // The Public Ip Id response
        // +optional
	ResourceId string `json:"resourceId,omitempty"`
}

type OscRoute struct {
        // The tag name associated with the Route
        // +optional
	Name        string `json:"name,omitempty"`
        // The tag name associated with the target resource type 
        // +optional
	TargetName  string `json:"targetName,omitempty"`
        // The target resource type which can be Internet Service (gateway) or Nat Service (nat-service)
        // +optional
	TargetType  string `json:"targetType,omitempty"`
        // the destination match Ip range with CIDR notation
        // +optional
	Destination string `json:"destination,omitempty"`
        // The Route Id response
        // +optional
	ResourceId  string `json:"resourceId,omitempty"`
}
// Map between resourceId and resourceName (tag Name with cluster UID)
type OscResourceMapReference struct {
	ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

type OscNetworkResource struct {
        // Map between LoadbalancerId  and LoadbalancerName (Load Balancer tag Name with cluster UID)
	LoadbalancerRef    OscResourceMapReference `json:"LoadbalancerRef,omitempty"`
        // Map between NetId  and NetName (Net tag Name with cluster UID)
	NetRef             OscResourceMapReference `json:"netref,omitempty"`
        // Map between SubnetId  and SubnetName (Subnet tag Name with cluster UID)
	SubnetRef          OscResourceMapReference `json:"subnetref,omitempty"`
        // Map between InternetServiceId  and InternetServiceName (Internet Service tag Name with cluster UID)
	InternetServiceRef OscResourceMapReference `json:"internetserviceref,omitempty"`
        // Map between RouteTablesId  and RouteTablesName (Route Tables tag Name with cluster UID)
	RouteTablesRef     OscResourceMapReference `json:"routetableref,omitempty"`
	LinkRouteTableRef  OscResourceMapReference `json:"linkroutetableref,omitempty"`
        // Map between RouteId  and RouteName (Route tag Name with cluster UID)
	RouteRef           OscResourceMapReference `json:"routeref,omitempty"`
        // Map between PublicIpId  and PublicIpName (Public IP tag Name with cluster UID)
	PublicIpRef        OscResourceMapReference `json:"publicipref,omitempty"`
        // Map between NatServiceId  and NatServiceName (Nat Service tag Name with cluster UID)
	NatServiceRef      OscResourceMapReference `json:"natref,omitempty"`
}

var (
	DefaultLoadBalancerName     string = "OscClusterApi-1"
	DefaultSubregionName        string = "eu-west-2a"
	DefaultBackendPort          int32  = 6443
	DefaultBackendProtocol      string = "TCP"
	DefaultLoadBalancerPort     int32  = 6443
	DefaultLoadBalancerProtocol string = "TCP"
	DefaultCheckInterval        int32  = 30
	DefaultHealthyThreshold     int32  = 10
	DefaultUnhealthyThreshold   int32  = 2
	DefaultTimeout              int32  = 5
	DefaultProtocol             string = "TCP"
	DefaultPort                 int32  = 6443
	DefaultIpRange              string = "172.19.95.128/25"
	DefaultIpSubnetRange        string = "172.19.95.192/27"
	DefaultTargetName           string = "cluster-api-internetservice"
	DefaultTargetType           string = "gateway"
	DefaultDestination          string = "0.0.0.0/0"
	DefaultRouteTableName       string = "cluster-api-routetable"
	DefaultRouteName            string = "cluster-api-route"
	DefaultPublicIpName         string = "cluster-api-publicip"
	DefaultNatServiceName       string = "cluster-api-natservice"
	DefaultSubnetName           string = "cluster-api-subnet"
	DefaultNetName              string = "cluster-api-net"
	DefaultInternetServiceName  string = "cluster-api-internetservice"
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

// SetDefaultValue set the Internet Service default values
func (igw *OscInternetService) SetDefaultValue() {
	if igw.Name == "" {
		igw.Name = DefaultInternetServiceName
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
		routetable := OscRouteTable{
			Name:       DefaultRouteTableName,
			SubnetName: DefaultSubnetName,
			Routes:     []OscRoute{route},
		}
		var routetables []*OscRouteTable = network.RouteTables
		routetables = append(routetables, &routetable)
		network.RouteTables = routetables
	}

}

// SetDefaultValue set the Route Table default values from routetable configuration
func (routetable *OscRouteTable) SetDefaultValue() {
	if len(routetable.Routes) == 0 {
		route := OscRoute{
			Name:        DefaultRouteName,
			TargetName:  DefaultTargetName,
			TargetType:  DefaultTargetType,
			Destination: DefaultDestination,
		}
		var routes []OscRoute = routetable.Routes
		routes = append(routes, route)
	}
}

// SetPublicIpDefaultDefaultValue set the Public Ip default values from publicip configuration
func (network *OscNetwork) SetPublicIpDefaultValue() {
	if len(network.PublicIps) == 0 {
		publicip := OscPublicIp{
			Name: DefaultPublicIpName,
		}
		var publicips []*OscPublicIp = network.PublicIps
		publicips = append(publicips, &publicip)
		network.PublicIps = publicips
	}
}

// SetSubnetDefaultValue set the Sunet default values from subnet configuration
func (network *OscNetwork) SetSubnetDefaultValue() {
	if len(network.Subnets) == 0 {
		subnet := OscSubnet{
			Name:          DefaultSubnetName,
			IpSubnetRange: DefaultIpSubnetRange,
		}
		var subnets []*OscSubnet = network.Subnets
		subnets = append(subnets, &subnet)
		network.Subnets = subnets
	}

}

// SetDefaultValue set the LoadBalancer Service default values
func (lb *OscLoadBalancer) SetDefaultValue() {
	if lb.LoadBalancerName == "" {
		lb.LoadBalancerName = DefaultLoadBalancerName
	}
	if lb.SubregionName == "" {
		lb.SubregionName = DefaultSubregionName
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
