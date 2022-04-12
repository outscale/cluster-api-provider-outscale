package service

import (
	"fmt"
	"regexp"
	"strconv"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

// ValidateLoadBalancerName check that the loadBalancerName is a valide name of load balancer
func ValidateLoadBalancerName(loadBalancerName string) bool {
	isValidate := regexp.MustCompile(`^[0-9A-Za-z\s\-]{0,32}$`).MatchString
	return isValidate(loadBalancerName)
}

// ValidatePort check that the  port is a valide port
func ValidatePort(port int32) (int32, error) {
	isValidatePort := regexp.MustCompile(`^()([1-9]|[1-5]?[0-9]{2,4}|6[1-4][0-9]{3}|65[1-4][0-9]{2}|655[1-2][0-9]|6553[1-5])$`).MatchString
	if isValidatePort(strconv.Itoa(int(port))) {
		return port, nil
	} else {
		return port, errors.New("Invalid Port")
	}
}

func ValidateLoadBalancerType(loadBalancerType string) bool {
	switch {
	case loadBalancerType == "internet-facing" || loadBalancerType == "internal":
		return true
	default:
		return false
	}
}

// ValidateInterval check that the interval is a valide time of second
func (s *Service) ValidateInterval(interval int32) (int32, error) {
	isValidateInterval := regexp.MustCompile(`^([5-9]|[1-9][0-9]{1}|[1-5][0-9]{2}|600)$`).MatchString
	if isValidateInterval(strconv.Itoa(int(interval))) {
		return interval, nil
	} else {
		return interval, errors.New("Invalid Interval")
	}
}

// ValidateThreshold check that the threshold is a valide number of ping
func (s *Service) ValidateThreshold(threshold int32) (int32, error) {
	isValidateThreshold := regexp.MustCompile(`^([1-9]|10)$`).MatchString
	if isValidateThreshold(strconv.Itoa(int(threshold))) {
		return threshold, nil
	} else {
		return threshold, errors.New("Invalid Interval")
	}
}

// ValidateTimeout check that the timeoout is a valide maximum time of second
func (s *Service) ValidateTimeout(timeout int32) (int32, error) {
	isValidateTimeout := regexp.MustCompile(`^([2-9]|[1-5][0-9]|60)$`).MatchString
	if isValidateTimeout(strconv.Itoa(int(timeout))) {
		return timeout, nil
	} else {
		return timeout, errors.New("Invalid Timeout")
	}
}

// GetName return the name of the loadBalancer
func (s *Service) GetName(spec *infrastructurev1beta1.OscLoadBalancer) (string, error) {
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
	CheckInterval, err := s.ValidateInterval(spec.HealthCheck.CheckInterval)
	if err != nil {
		return nil, err
	}
	HealthyThreshold, err := s.ValidateThreshold(spec.HealthCheck.HealthyThreshold)
	if err != nil {
		return nil, err
	}
	Port, err := ValidatePort(spec.HealthCheck.Port)
	if err != nil {
		return nil, err
	}
	Protocol, err := s.ValidateProtocol(spec.HealthCheck.Protocol)
	if err != nil {
		return nil, err
	}
	Timeout, err := s.ValidateTimeout(spec.HealthCheck.Timeout)
	if err != nil {
		return nil, err
	}
	UnhealthyThreshold, err := s.ValidateThreshold(spec.HealthCheck.UnhealthyThreshold)
	if err != nil {
		return nil, err
	}
	healthCheck := osc.HealthCheck{
		CheckInterval:      CheckInterval,
		HealthyThreshold:   HealthyThreshold,
		Port:               Port,
		Protocol:           Protocol,
		Timeout:            Timeout,
		UnhealthyThreshold: UnhealthyThreshold,
	}
	updateLoadBalancerRequest := osc.UpdateLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
		HealthCheck:      &healthCheck,
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	updateLoadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.UpdateLoadBalancer(oscAuthClient).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return updateLoadBalancerResponse.LoadBalancer, nil
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
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readLoadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.ReadLoadBalancers(oscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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

// CreateLoadBalancer create the load balancer
func (s *Service) CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}

	loadBalancerType := spec.LoadBalancerType

	BackendPort, err := ValidatePort(spec.Listener.BackendPort)
	if err != nil {
		return nil, err
	}
	LoadBalancerPort, err := ValidatePort(spec.Listener.LoadBalancerPort)
	if err != nil {
		return nil, err
	}
	BackendProtocol, err := s.ValidateProtocol(spec.Listener.BackendProtocol)
	if err != nil {
		return nil, err
	}
	LoadBalancerProtocol, err := s.ValidateProtocol(spec.Listener.LoadBalancerProtocol)
	if err != nil {
		return nil, err
	}
	first_listener := osc.ListenerForCreation{
		BackendPort:          BackendPort,
		BackendProtocol:      &BackendProtocol,
		LoadBalancerPort:     LoadBalancerPort,
		LoadBalancerProtocol: LoadBalancerProtocol,
	}

	loadBalancerRequest := osc.CreateLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
		LoadBalancerType: &loadBalancerType,
		Listeners:        []osc.ListenerForCreation{first_listener},
		SecurityGroups:   &[]string{securityGroupId},
		Subnets:          &[]string{subnetId},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	loadBalancerResponse, httpRes, err := oscApiClient.LoadBalancerApi.CreateLoadBalancer(oscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return loadBalancerResponse.LoadBalancer, nil
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
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.LoadBalancerApi.DeleteLoadBalancer(oscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}
