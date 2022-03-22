package service

import(
    "regexp"
    infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
    "github.com/pkg/errors"
    osc "github.com/outscale/osc-sdk-go/v2"
    "fmt"
)

func ValidateLoadBalancerName(loadBalancerName string) (bool) {
   isValidate := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
   return isValidate(loadBalancerName)
}

func ValidateLoadBalancerRegionName(loadBalancerRegionName string) (bool) {
   isValidate := regexp.MustCompile(`^((?:[a-zA-Z]+-){2,3}[1-3-a-c]{2})$`).MatchString
   return isValidate(loadBalancerRegionName)
}

func (s *Service) GetName(spec infrastructurev1beta1.OscClusterSpec) (string, error) {
    var name string
    var clusterName string
    switch {
    case spec.LoadBalancerName != "":
        name = spec.LoadBalancerName
    default:
        clusterName = infrastructurev1beta1.OscReplaceName(s.scope.Name())
        name = clusterName + "-" + "apiserver" + "-" + s.scope.UID()
    }
    if ValidateLoadBalancerName(name) {
        return name, nil
    } else {
        return "", errors.New("Invalid Name")
    }
}

func (s *Service) GetRegionName(spec infrastructurev1beta1.OscClusterSpec) (string, error) {
    var name string
    switch {
    case spec.LoadBalancerRegion != "":
        name = spec.LoadBalancerRegion
    default:
        name = s.scope.Region()
    }
    if ValidateLoadBalancerRegionName(name) {
        return name, nil
    } else {
        return "", errors.New("Invalid Region Name")
    }
}

func (s *Service) GetLoadBalancer(spec infrastructurev1beta1.OscClusterSpec) (*osc.LoadBalancer, error) {
    loadBalancerName, err := s.GetName(spec)
    if err != nil {
        return nil, err
    } 
    filterLoadBalancer := osc.FiltersLoadBalancer{
	LoadBalancerNames: &[]string{loadBalancerName},
    }
    readLoadBalancerRequest := osc.ReadLoadBalancersRequest{Filters: &filterLoadBalancer}
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    readLoadBalancerResponse, httpRes, err := OscApiClient.LoadBalancerApi.ReadLoadBalancers(OscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()  
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    var lb []osc.LoadBalancer
    loadBalancers := *readLoadBalancerResponse.LoadBalancers 
    if len(loadBalancers) == 0 {
        return nil, nil
    } else {
        lb = append(lb, loadBalancers...)
        return &lb[0], nil
    }
} 

func (s *Service) CreateLoadBalancer(spec infrastructurev1beta1.OscClusterSpec) (*osc.LoadBalancer, error) {
    loadBalancerName, err := s.GetName(spec)   
    if err != nil {
        return nil, err
    }
    loadBalancerRegionName, err := s.GetRegionName(spec)
    if err != nil {
        return nil, err
    }
    fmt.Sprintf("LoadBalancer %s in region %s\n", loadBalancerName, loadBalancerRegionName)
    first_listener := osc.ListenerForCreation{
	BackendPort:          80,
	LoadBalancerPort:     80,
	LoadBalancerProtocol: "TCP",
    }
    first_tag := osc.ResourceTag{
        Key: "project",
        Value: "cluster-api",
    }
    loadBalancerRequest := osc.CreateLoadBalancerRequest{
	LoadBalancerName: loadBalancerName,
	Listeners:        []osc.ListenerForCreation{first_listener},
	SubregionNames:   &[]string{loadBalancerRegionName},
               
        Tags: &[]osc.ResourceTag{first_tag},
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    loadBalancerResponse, httpRes, err := OscApiClient.LoadBalancerApi.CreateLoadBalancer(OscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()    
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    return loadBalancerResponse.LoadBalancer, nil
}

func (s *Service) DeleteLoadBalancer(spec infrastructurev1beta1.OscClusterSpec) (error) {
    loadBalancerName, err := s.GetName(spec)
    if err != nil {
        return err
    }
    deleteLoadBalancerRequest := osc.DeleteLoadBalancerRequest{
        LoadBalancerName: loadBalancerName,
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    _, httpRes, err := OscApiClient.LoadBalancerApi.DeleteLoadBalancer(OscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return err
    }
    return nil
}


