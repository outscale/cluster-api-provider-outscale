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
	_nethttp "net/http"
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_service/loadbalancer_mock.go -package mock_service -source ./load_balancer.go
type OscLoadBalancerInterface interface {
	ConfigureHealthCheck(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	GetLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error)
	DeleteLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer) error
	LinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error
	UnlinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error
	CheckLoadBalancerDeregisterVm(clockInsideLoop time.Duration, clockLoop time.Duration, spec *infrastructurev1beta1.OscLoadBalancer) error
	GetLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancerTag, error)
	CreateLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceTag) error
	DeleteLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error
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
	_, err := infrastructurev1beta1.ValidateLoadBalancerName(name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// ValidateProtocol check that the protocol string is a valide protocol
func ValidateProtocol(protocol string) (string, error) {
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var updateLoadBalancerResponse osc.UpdateLoadBalancerResponse
	updateLoadBalancerCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		updateLoadBalancerResponse, httpRes, err = oscApiClient.LoadBalancerApi.UpdateLoadBalancer(oscAuthClient).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", updateLoadBalancerRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, updateLoadBalancerCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	loadBalancer, ok := updateLoadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("Can not update loadbalancer")
	}
	return loadBalancer, nil
}

// LinkLoadBalancerBackendMachines link the loadBalancer with vm backend
func (s *Service) LinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error {
	linkLoadBalancerBackendMachinesRequest := osc.LinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	linkLoadBalancerBackendMachinesCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.LinkLoadBalancerBackendMachines(oscAuthClient).LinkLoadBalancerBackendMachinesRequest(linkLoadBalancerBackendMachinesRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", linkLoadBalancerBackendMachinesRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, linkLoadBalancerBackendMachinesCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// UnlinkLoadBalancerBackendMachines unlink the loadbalancer with vm backend
func (s *Service) UnlinkLoadBalancerBackendMachines(vmIds []string, loadBalancerName string) error {
	unlinkLoadBalancerBackendMachinesRequest := osc.UnlinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	unlinkLoadBalancerBackendMachinesCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.UnlinkLoadBalancerBackendMachines(oscAuthClient).UnlinkLoadBalancerBackendMachinesRequest(unlinkLoadBalancerBackendMachinesRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", unlinkLoadBalancerBackendMachinesRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, unlinkLoadBalancerBackendMachinesCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
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

	var readLoadBalancersResponse osc.ReadLoadBalancersResponse
	readLoadBalancerCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readLoadBalancersResponse, httpRes, err = oscApiClient.LoadBalancerApi.ReadLoadBalancers(oscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
		if err != nil {
			if httpRes != nil {
				requestStr := fmt.Sprintf("%v", readLoadBalancerRequest)
				if reconciler.KeepRetryWithError(
					requestStr,
					httpRes.StatusCode,
					reconciler.ThrottlingErrors) {
					return false, nil
				} else {
					return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
				}
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readLoadBalancerCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	var lb []osc.LoadBalancer
	loadBalancers, ok := readLoadBalancersResponse.GetLoadBalancersOk()
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

// GetLoadBalancerTag retrieve loadBalancer object from spec
func (s *Service) GetLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancerTag, error) {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return nil, err
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readLoadBalancerTagRequest := osc.ReadLoadBalancerTagsRequest{
		LoadBalancerNames: []string{loadBalancerName},
	}
	var readLoadBalancerTagsResponse osc.ReadLoadBalancerTagsResponse
	readLoadBalancerTagCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readLoadBalancerTagsResponse, httpRes, err = oscApiClient.LoadBalancerApi.ReadLoadBalancerTags(oscAuthClient).ReadLoadBalancerTagsRequest(readLoadBalancerTagRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readLoadBalancerTagRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, fmt.Errorf("%w failed to read Tag Name", err)
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readLoadBalancerTagCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	var tag []osc.LoadBalancerTag
	tags, ok := readLoadBalancerTagsResponse.GetTagsOk()
	if !ok {
		return nil, errors.New("Can not get tags")
	}
	if len(*tags) == 0 {
		return nil, nil
	} else {
		tag = append(tag, *tags...)
		return &tag[0], nil
	}
}

// CreateLoadBalancerTag create the load balancer tag
func (s *Service) CreateLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceTag) error {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return err
	}
	createLoadBalancerTagRequest := osc.CreateLoadBalancerTagsRequest{
		LoadBalancerNames: []string{loadBalancerName},
		Tags:              []osc.ResourceTag{loadBalancerTag},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	createLoadBalancerTagCallBack := func() (bool, error) {
		_, httpRes, err := oscApiClient.LoadBalancerApi.CreateLoadBalancerTags(oscAuthClient).CreateLoadBalancerTagsRequest(createLoadBalancerTagRequest).Execute()
		if err != nil {
			if httpRes != nil {
				fmt.Printf("Error with http result %s", httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", createLoadBalancerTagRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, fmt.Errorf("%w failed to add Tag", err)
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createLoadBalancerTagCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// CreateLoadBalancer create the load balancer
func (s *Service) CreateLoadBalancer(spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error) {
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

	var loadBalancerResponse osc.CreateLoadBalancerResponse
	createLoadBalancerCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		loadBalancerResponse, httpRes, err = oscApiClient.LoadBalancerApi.CreateLoadBalancer(oscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", loadBalancerRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createLoadBalancerCallBack)
	if waitErr != nil {
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
	deleteLoadBalancerCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.DeleteLoadBalancer(oscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteLoadBalancerRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteLoadBalancerCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// DeleteLoadBalancerTag delete the loadbalancerTag
func (s *Service) DeleteLoadBalancerTag(spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error {
	loadBalancerName, err := s.GetName(spec)
	if err != nil {
		return err
	}
	deleteLoadBalancerTagRequest := osc.DeleteLoadBalancerTagsRequest{
		LoadBalancerNames: []string{loadBalancerName},
		Tags:              []osc.ResourceLoadBalancerTag{loadBalancerTag},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteLoadBalancerTagCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.DeleteLoadBalancerTags(oscAuthClient).DeleteLoadBalancerTagsRequest(deleteLoadBalancerTagRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteLoadBalancerTagRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteLoadBalancerTagCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// CheckLoadBalancerDeregisterVm check vm is deregister as vm backend in loadBalancer
func (s *Service) CheckLoadBalancerDeregisterVm(clockInsideLoop time.Duration, clockLoop time.Duration, spec *infrastructurev1beta1.OscLoadBalancer) error {
	clock_time := clock.New()
	currentTimeout := clock_time.Now().Add(time.Second * clockLoop)
	var getLoadBalancerDeregisterVm = false
	for !getLoadBalancerDeregisterVm {
		time.Sleep(clockInsideLoop * time.Second)
		loadBalancer, err := s.GetLoadBalancer(spec)
		if err != nil {
			return err
		}
		loadBalancerBackendVmIds := loadBalancer.GetBackendVmIds()
		if len(loadBalancerBackendVmIds) == 0 {
			break
		}
		if clock_time.Now().After(currentTimeout) {
			return errors.New("vm is still register")
		}
	}
	return nil
}
