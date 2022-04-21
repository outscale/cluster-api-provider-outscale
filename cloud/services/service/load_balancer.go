package service

import (
	"fmt"
	"regexp"

	"errors"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
)

const (
	minPort      = 0
	maxPort      = 65536
	minInterval  = 4
	maxInterval  = 601
	minThreshold = 0
	maxThreshold = 11
	minTimeout   = 1
	maxTimeout   = 61
)

//go:generate ../../../bin/mockgen -destination mock_service/loadbalancer_mock.go -package mock_service -source ./load_balancer.go
type OscLoadBalancerInterface interface {
	ConfigureHealthCheck(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	GetLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error)
	DeleteLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) error
}

// ValidateLoadBalancerName check that the loadBalancerName is a valide name of load balancer
func ValidateLoadBalancerName(loadBalancerName string) bool {
	isValidate := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
	return isValidate(loadBalancerName)
}

// ValidatePort check that the  port is a valide port
func ValidatePort(port int32) (int32, error) {
	if port > minPort && port < maxPort {
		return port, nil
	} else {
		return port, errors.New("Invalid Port")
	}
}

func ValidateLoadBalancerType(loadBalancerType string) bool {
	if loadBalancerType == "internet-facing" || loadBalancerType == "internal" {
		return true
	} else {
		return false
	}
}

// ValidateInterval check that the interval is a valide time of second
func (s *Service) ValidateInterval(interval int32) (int32, error) {
	if interval > minInterval && interval < maxInterval {
		return interval, nil
	} else {
		return interval, errors.New("Invalid Interval")
	}
}

// ValidateThreshold check that the threshold is a valide number of ping
func (s *Service) ValidateThreshold(threshold int32) (int32, error) {
	if threshold > minThreshold && threshold < maxThreshold {
		return threshold, nil
	} else {
		return threshold, errors.New("Invalid threshold")
	}
}

// ValidateTimeout check that the timeoout is a valide maximum time of second
func (s *Service) ValidateTimeout(timeout int32) (int32, error) {
	if timeout > minTimeout && timeout < maxTimeout {
		return timeout, nil
	} else {
		return timeout, errors.New("Invalid Timeout")
	}
}

// GetName return the name of the loadBalancer
func (s *Service) GetName(spec *infrastructurev1beta1.OscLoadBalancer) (string, error) {
	var name string
	var clusterName string
	if spec.LoadBalancerName != "" {
		name = spec.LoadBalancerName
	} else {
		clusterName = infrastructurev1beta1.OscReplaceName(s.scope.GetName())
		name = clusterName + "-" + "apiserver" + "-" + s.scope.GetUID()
	}
	if !ValidateLoadBalancerName(name) {
		return "", errors.New("Invalid Name")
	}
	return name, nil
}

// ValidateProtocol check that the protocol string is a valide protocol
func (s *Service) ValidateProtocol(protocol string) (string, error) {
	switch {
	case protocol == "HTTP" || protocol == "TCP":
		return protocol, nil
	case protocol == "SSL" || protocol == "HTTPS":
		return protocol, errors.New("Ssl certificat is required")
	default:
		return protocol, errors.New("Invalid protocol")
	}
}

// ConfigureHealthCheck update loadBalancer to configure healthCheck
func (s *Service) ConfigureHealthCheck(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}
	checkInterval, err := s.ValidateInterval(spec.HealthCheck.CheckInterval)
	if err != nil {
		return nil, err
	}
	healthyThreshold, err := s.ValidateThreshold(spec.HealthCheck.HealthyThreshold)
	if err != nil {
		return nil, err
	}
	port, err := ValidatePort(spec.HealthCheck.Port)
	if err != nil {
		return nil, err
	}
	protocol, err := s.ValidateProtocol(spec.HealthCheck.Protocol)
	if err != nil {
		return nil, err
	}
	timeout, err := s.ValidateTimeout(spec.HealthCheck.Timeout)
	if err != nil {
		return nil, err
	}
	unhealthyThreshold, err := s.ValidateThreshold(spec.HealthCheck.UnhealthyThreshold)
	if err != nil {
		return nil, err
	}
	healthCheck := osc.HealthCheck{
		CheckInterval:      checkInterval,
		HealthyThreshold:   healthyThreshold,
		Port:               port,
		Protocol:           protocol,
		Timeout:            timeout,
		UnhealthyThreshold: unhealthyThreshold,
	}
	updateLoadBalancerRequest := osc.UpdateLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
		HealthCheck:      &healthCheck,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	updateLoadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.UpdateLoadBalancer(oscAuthClient).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	loadBalancer, ok := updateLoadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("Can not update loadbalancer")
	}
	return loadBalancer, nil
}

// GetLoadBalancer retrieve loadBalancer object from spec
func (s *Service) GetLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}
	filterLoadBalancer := osc.FiltersLoadBalancer{
		LoadBalancerNames: &[]string{loadBalancerName},
	}
	readLoadBalancerRequest := osc.ReadLoadBalancersRequest{Filters: &filterLoadBalancer}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readLoadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.ReadLoadBalancers(oscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var lb []osc.LoadBalancer
	loadBalancers, ok := readLoadBalancerResponse.GetLoadBalancersOk()
	if !ok {
		return nil, errors.New("Can not get loadbalancer")
	}
	if len(*loadBalancers) == 0 {
		return nil, nil
	} else {
		lb = append(lb, *loadBalancers...)
		return &lb[0], nil
	}
}

// CreateLoadBalancer create the load balancer
func (s *Service) CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}

	loadBalancerType := spec.LoadBalancerType

	backendPort, err := ValidatePort(spec.Listener.BackendPort)
	if err != nil {
		return nil, err
	}
	loadBalancerPort, err := ValidatePort(spec.Listener.LoadBalancerPort)
	if err != nil {
		return nil, err
	}
	backendProtocol, err := s.ValidateProtocol(spec.Listener.BackendProtocol)
	if err != nil {
		return nil, err
	}
	loadBalancerProtocol, err := s.ValidateProtocol(spec.Listener.LoadBalancerProtocol)
	if err != nil {
		return nil, err
	}
	first_listener := osc.ListenerForCreation{
		BackendPort:          backendPort,
		BackendProtocol:      &backendProtocol,
		LoadBalancerPort:     loadBalancerPort,
		LoadBalancerProtocol: loadBalancerProtocol,
	}

	loadBalancerRequest := osc.CreateLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
		LoadBalancerType: &loadBalancerType,
		Listeners:        []osc.ListenerForCreation{first_listener},
		SecurityGroups:   &[]string{securityGroupId},
		Subnets:          &[]string{subnetId},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	loadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.CreateLoadBalancer(oscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	loadBalancer, ok := loadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("Can not create loadbalancer")
	}
	return loadBalancer, nil
}

// DeleteLoadBalancer delete the loadbalancer
func (s *Service) DeleteLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) error {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return err
	}
	deleteLoadBalancerRequest := osc.DeleteLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.LoadBalancerApi.DeleteLoadBalancer(oscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}
