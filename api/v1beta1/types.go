package v1beta1

type OscNetwork struct {
    LoadBalancer OscLoadBalancer `json:"Loadbalancer,omitempty"`
    Net OscNet `json:"net,omitempty"`   
    Subnet OscSubnet `json:"subnet,omitempty"` 
    InternetService OscInternetService `json:"internetService,omitempty"`
    NatService OscNatService `json:"natService,omitempty"`
    RouteTables []OscRouteTable `json:"routeTables,omitempty"`
    PublicIps []OscPublicIp `json:"publicips,omitempty"`
}

type OscLoadBalancer struct {
    LoadBalancerName string `json:"loadbalancername,omitempty"`
    SubregionName string `json:"subregionname,omitempty"`
    Listener OscLoadBalancerListener `json:"listener,omitempty"`
    HealthCheck OscLoadBalancerHealthCheck `json:"healthCheck,omitempty"`
}

type OscLoadBalancerListener struct {
    BackendPort int32 `json:"backendport,omitempty"`
    BackendProtocol string `json:"backendprotocol,omitempty"`
    LoadBalancerPort int32 `json:"loadbalancerport,omitempty"`
    LoadBalancerProtocol string `json:"loadbalancerprotocol,omiempty"`
}

type OscLoadBalancerHealthCheck struct {
    CheckInterval int32 `json:"checkinterval,omitempty"`
    HealthyThreshold int32 `json:"healthythreshold,omitempty"`
    Port int32 `json:"port,omitempty"`
    Protocol string `json:"protocol,omitepty"`
    Timeout int32 `json:"timeout,omitempty"`
    UnhealthyThreshold int32 `json:"unhealthythreshold,omitempty"`
}

type OscNet struct {
    Name string `json:"name,omitempty"`
    IpRange string `json:"ipRange,omitempty"`
}

type OscInternetService struct {
    Name string `json:"name,omitempty"`
}

type OscSubnet struct {
    Name string `json:"name,omitempty"`
    IpSubnetRange string `json:"ipSubnetRange,omitempty"`
}

type OscNatService struct {
    Name string `json:"name,omitempty"` 
    PublicIpName string `json:"publicipname,omitempty"`
    SubnetName string `json:"subnetname,omitempty"`
}

type OscRouteTable struct {
    Name string `json:"name,omitempty"`
    Routes []OscRoute `json:"routes,omitempty"`
}

type OscPublicIp struct {
    Name string `json:"name,omitempty"`
}

type OscRoute struct {
    Name string `json:"name,omitempty"`
    TargetName string `json:"targetName,omitempty"` 
    TargetType string `json:"targetType,omitempty"`
    Destination string `json:"destination,omitempty"`
}

type OscResourceMapReference struct {
    ResourceMap map[string]string `json:"resourceMap,omitempty"`
}

type OscNetworkResource struct {
    LoadbalancerRef OscResourceMapReference `json:"LoadbalancerRef,omitempty"`
    NetRef OscResourceMapReference `json:"netref,omitempty"`
    SubnetRef OscResourceMapReference `json:"subnetref,omitempty"`
    InternetServiceRef OscResourceMapReference `json:"internetserviceref,omitempty"`
    RouteTablesRef OscResourceMapReference `json:"routetableref,omitempty"`
    LinkRouteTableRef OscResourceMapReference `json:"linkroutetableref,omitempty"`
    RouteRef OscResourceMapReference `json:"routeref,omitempty"`
    PublicIpRef OscResourceMapReference `json:"publicipref,omitempty"`
    NatServiceRef OscResourceMapReference `json:"natref,omitempty"`
}

var (
    DefaultLoadBalancerName string = "OscClusterApi-1"
    DefaultSubregionName string = "eu-west-2a"
    DefaultBackendPort int32 = 6443
    DefaultBackendProtocol string = "TCP"
    DefaultLoadBalancerPort int32 = 6443
    DefaultLoadBalancerProtocol string = "TCP"
    DefaultCheckInterval int32 = 30
    DefaultHealthyThreshold int32 = 10
    DefaultUnhealthyThreshold int32 = 2
    DefaultTimeout int32 = 5
    DefaultProtocol string = "TCP"
    DefaultPort int32 = 6443
    DefaultIpRange string = "172.19.95.128/25"
    DefaultIpSubnetRange string = "172.19.95.192/27"
    DefaultTargetName = "cluster-api-igw"
    DefaultTargetType = "igw"
    DefaultDestination = "0.0.0.0/0"
    DefaultRouteTableName = "cluster-api-routetable"
    DefaultRouteName = "cluster-api-route"
    DefaultPublicIpName = "cluster-api-publicip"
    DefaultNatServiceName = "cluster-api-natservice"
    DefaultSubnetName = "cluster-api-subnet"
    DefaultNetName = "cluster-api-net"
    DefaultInternetServiceName = "cluster-api-internetservice"
)

func (net *OscNet) SetDefaultValue() {
    if net.IpRange == "" {
        net.IpRange = DefaultIpRange
    }
    if net.Name == "" {
        net.Name = DefaultNetName
    } 
}

func (igw *OscInternetService) SetDefaultValue() {
    if igw.Name == "" {
        igw.Name = DefaultInternetServiceName   
    }
}

func (sub *OscSubnet) SetDefaultValue() {
    if sub.IpSubnetRange == "" {
        sub.IpSubnetRange = DefaultIpSubnetRange
    }
    if sub.Name == "" {
        sub.Name = DefaultSubnetName
    }
}

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

func (network *OscNetwork) SetRouteTableDefaultValue() {
    if len(network.RouteTables) == 0 {
        route := OscRoute{
            Name: DefaultRouteName,
            TargetName: DefaultTargetName,
            TargetType: DefaultTargetType,
            Destination: DefaultDestination,
        } 
        routetable := OscRouteTable{
            Name: DefaultRouteTableName,
            Routes: []OscRoute{route},
        }       
        var routetables []OscRouteTable = network.RouteTables
        routetables = append(routetables, routetable)
    }    

}

func (routetable *OscRouteTable) SetDefaultValue() {
    if len(routetable.Routes) == 0 {
        route := OscRoute{
            Name: DefaultRouteName,
            TargetName: DefaultTargetName,
            TargetType: DefaultTargetType,
            Destination: DefaultDestination,
        }
        var routes []OscRoute = routetable.Routes
        routes = append(routes,route)        
    }  
}

func (network *OscNetwork) SetPublicIpDefaultValue() {
    if len(network.PublicIps) == 0 {
        publicip := OscPublicIp {
            Name: DefaultPublicIpName,       
        }
        var publicips []OscPublicIp = network.PublicIps
        publicips = append(publicips,publicip)
    }
}

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
