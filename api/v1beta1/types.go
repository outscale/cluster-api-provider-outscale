package v1beta1

type OscNetwork struct {
    LoadBalancer OscLoadBalancer `json:"Loadbalancer,omitempty"`
    Net OscNet `json:"net,omitempty"`   
    Subnet OscSubnet `json:"subnet,omitempty"` 
    InternetService OscInternetService `json:"internetService,omitempty"`
    RouteTables []OscRouteTable `json:"routeTables,omitempty"`
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
    IpRange string `json:"ipRange,omitempty"`
}

type OscInternetService struct {}

type OscSubnet struct {
    IpSubnetRange string `json:"ipSubnetRange,omitempty"`
}

type OscRouteTable struct {
    Name string `json:"name,omitempty"`
    Routes []OscRoute `json:"routes,omitempty"`
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
)

func (net *OscNet) SetDefaultValue() {
    if net.IpRange == "" {
        net.IpRange = DefaultIpRange
    } 
}

func (sub *OscSubnet) SetDefaultValue() {
    if sub.IpSubnetRange == "" {
        sub.IpSubnetRange = DefaultIpSubnetRange
    }
}

func (network *OscNetwork) SetDefaultValue() {
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
