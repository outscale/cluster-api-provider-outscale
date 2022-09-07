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

package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_service/loadbalancer_mock.go -package mock_service -source ./load_balancer.go
type OscLoadBalancerInterface interface {
	ConfigureHealthCheck(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	GetLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetID string, securityGroupID string) (*osc.LoadBalancer, error)
	DeleteLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) error
	LinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error
	UnlinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error
	CheckLoadBalancerDeregisterVM(clockInsideLoop time.Duration, clockLoop time.Duration, spec *infrastructurev1beta1.OscLoadBalancer) error
}

// GetName return the name of the loadBalancer.
func (s *Service) GetName(spec *infrastructurev1beta1.OscLoadBalancer) (string, error) {
	var name string
	var clusterName string
	if spec.LoadBalancerName != "" {
		name = spec.LoadBalancerName
	} else {
		clusterName = infrastructurev1beta1.OscReplaceName(s.scope.GetName())
		name = clusterName + "-" + "apiserver" + "-" + s.scope.GetUID()
	}
	_, err := infrastructurev1beta1.ValidateLoadBalancerName(name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ValidateProtocol check that the protocol string is a valide protocol.
func ValidateProtocol(protocol string) (string, error) {
	switch {
	case protocol == "HTTP" || protocol == "TCP":
		return protocol, nil
	case protocol == "SSL" || protocol == "HTTPS":
		return protocol, errors.New("ssl certificat is required")
	default:
		return protocol, errors.New("invalid protocol")
	}
}

// ConfigureHealthCheck update loadBalancer to configure healthCheck.
func (s *Service) ConfigureHealthCheck(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}
	checkInterval := spec.HealthCheck.CheckInterval
	healthyThreshold := spec.HealthCheck.HealthyThreshold
	port := spec.HealthCheck.Port
	protocol := spec.HealthCheck.Protocol
	timeout := spec.HealthCheck.Timeout
	unhealthyThreshold := spec.HealthCheck.UnhealthyThreshold
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
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	updateLoadBalancerResponse, httpRes, err := oscAPIClient.LoadBalancerApi.UpdateLoadBalancer(oscAuthClient).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	loadBalancer, ok := updateLoadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("can not update loadbalancer")
	}
	return loadBalancer, nil
}

// LinkLoadBalancerBackendMachines link the loadBalancer with vm backend.
func (s *Service) LinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error {
	linkLoadBalancerBackendMachinesRequest := osc.LinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.LoadBalancerApi.LinkLoadBalancerBackendMachines(oscAuthClient).LinkLoadBalancerBackendMachinesRequest(linkLoadBalancerBackendMachinesRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// UnlinkLoadBalancerBackendMachines unlink the loadbalancer with vm backend.
func (s *Service) UnlinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error {
	unlinkLoadBalancerBackendMachinesRequest := osc.UnlinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.LoadBalancerApi.UnlinkLoadBalancerBackendMachines(oscAuthClient).UnlinkLoadBalancerBackendMachinesRequest(unlinkLoadBalancerBackendMachinesRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetLoadBalancer retrieve loadBalancer object from spec.
func (s *Service) GetLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}
	filterLoadBalancer := osc.FiltersLoadBalancer{
		LoadBalancerNames: &[]string{loadBalancerName},
	}
	readLoadBalancerRequest := osc.ReadLoadBalancersRequest{Filters: &filterLoadBalancer}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readLoadBalancerResponse, httpRes, err := oscAPIClient.LoadBalancerApi.ReadLoadBalancers(oscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var lb []osc.LoadBalancer
	loadBalancers, ok := readLoadBalancerResponse.GetLoadBalancersOk()
	if !ok {
		return nil, errors.New("can not get loadbalancer")
	}
	if len(*loadBalancers) == 0 {
		return nil, nil
	}
	lb = append(lb, *loadBalancers...)
	return &lb[0], nil
}

// CreateLoadBalancer create the load balancer.
func (s *Service) CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetID string, securityGroupID string) (*osc.LoadBalancer, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}

	loadBalancerType := spec.LoadBalancerType
	backendPort := spec.Listener.BackendPort
	loadBalancerPort := spec.Listener.LoadBalancerPort
	backendProtocol := spec.Listener.BackendProtocol
	loadBalancerProtocol := spec.Listener.LoadBalancerProtocol

	if err != nil {
		return nil, err
	}
	firstlistener := osc.ListenerForCreation{
		BackendPort:          backendPort,
		BackendProtocol:      &backendProtocol,
		LoadBalancerPort:     loadBalancerPort,
		LoadBalancerProtocol: loadBalancerProtocol,
	}

	loadBalancerRequest := osc.CreateLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
		LoadBalancerType: &loadBalancerType,
		Listeners:        []osc.ListenerForCreation{firstlistener},
		SecurityGroups:   &[]string{securityGroupID},
		Subnets:          &[]string{subnetID},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	loadBalancerResponse, httpRes, err := oscAPIClient.LoadBalancerApi.CreateLoadBalancer(oscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	loadBalancer, ok := loadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("can not create loadbalancer")
	}
	return loadBalancer, nil
}

// DeleteLoadBalancer delete the loadbalancer.
func (s *Service) DeleteLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) error {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return err
	}
	deleteLoadBalancerRequest := osc.DeleteLoadBalancerRequest{
		LoadBalancerName: loadBalancerName,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.LoadBalancerApi.DeleteLoadBalancer(oscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// CheckLoadBalancerDeregisterVM check vm is deregister as vm backend in loadBalancer.
func (s *Service) CheckLoadBalancerDeregisterVM(clockInsideLoop time.Duration, clockLoop time.Duration, spec *infrastructurev1beta1.OscLoadBalancer) error {
	clocktime := clock.New()
	currentTimeout := clocktime.Now().Add(time.Second * clockLoop)
	var getLoadBalancerDeregisterVM = false
	for !getLoadBalancerDeregisterVM {
		loadBalancer, err := s.GetLoadBalancer(spec)
		if err != nil {
			return err
		}
		loadBalancerBackendVMIds := loadBalancer.GetBackendVmIds()
		if len(loadBalancerBackendVMIds) == 0 {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)
		if clocktime.Now().After(currentTimeout) {
			return errors.New("vm is still register")
		}
	}
	return nil
}
