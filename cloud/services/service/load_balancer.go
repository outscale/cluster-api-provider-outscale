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
	"context"
	"errors"
	"fmt"
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_service/loadbalancer_mock.go -package mock_service -source ./load_balancer.go
type OscLoadBalancerInterface interface {
	ConfigureHealthCheck(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error)
	GetLoadBalancer(ctx context.Context, loadBalancerName string) (*osc.LoadBalancer, error)
	CreateLoadBalancer(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) error
	LinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error
	UnlinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error
	GetLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancerTag, error)
	CreateLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag *osc.ResourceTag) error
	DeleteLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error
}

// ValidateProtocol check that the protocol string is a valid protocol
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
func (s *Service) ConfigureHealthCheck(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancer, error) {
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
		LoadBalancerName: spec.LoadBalancerName,
		HealthCheck:      &healthCheck,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var updateLoadBalancerResponse osc.UpdateLoadBalancerResponse
	updateLoadBalancerCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		updateLoadBalancerResponse, httpRes, err = oscApiClient.LoadBalancerApi.UpdateLoadBalancer(oscAuthClient).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
		utils.LogAPICall(ctx, "UpdateLoadBalancer", updateLoadBalancerRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) LinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error {
	linkLoadBalancerBackendMachinesRequest := osc.LinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	linkLoadBalancerBackendMachinesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.LinkLoadBalancerBackendMachines(oscAuthClient).LinkLoadBalancerBackendMachinesRequest(linkLoadBalancerBackendMachinesRequest).Execute()
		utils.LogAPICall(ctx, "LinkLoadBalancerBackendMachines", linkLoadBalancerBackendMachinesRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) UnlinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error {
	unlinkLoadBalancerBackendMachinesRequest := osc.UnlinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	unlinkLoadBalancerBackendMachinesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.UnlinkLoadBalancerBackendMachines(oscAuthClient).UnlinkLoadBalancerBackendMachinesRequest(unlinkLoadBalancerBackendMachinesRequest).Execute()
		utils.LogAPICall(ctx, "UnlinkLoadBalancerBackendMachines", unlinkLoadBalancerBackendMachinesRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) GetLoadBalancer(ctx context.Context, loadBalancerName string) (*osc.LoadBalancer, error) {
	filterLoadBalancer := osc.FiltersLoadBalancer{
		LoadBalancerNames: &[]string{loadBalancerName},
	}
	readLoadBalancerRequest := osc.ReadLoadBalancersRequest{Filters: &filterLoadBalancer}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var readLoadBalancersResponse osc.ReadLoadBalancersResponse
	readLoadBalancerCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readLoadBalancersResponse, httpRes, err = oscApiClient.LoadBalancerApi.ReadLoadBalancers(oscAuthClient).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
		utils.LogAPICall(ctx, "ReadLoadBalancers", readLoadBalancerRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				requestStr := fmt.Sprintf("%v", readLoadBalancerRequest)
				if reconciler.KeepRetryWithError(
					requestStr,
					httpRes.StatusCode,
					reconciler.ThrottlingErrors) {
					return false, nil
				} else {
					return false, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) GetLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) (*osc.LoadBalancerTag, error) {
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readLoadBalancerTagRequest := osc.ReadLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
	}
	var readLoadBalancerTagsResponse osc.ReadLoadBalancerTagsResponse
	readLoadBalancerTagCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readLoadBalancerTagsResponse, httpRes, err = oscApiClient.LoadBalancerApi.ReadLoadBalancerTags(oscAuthClient).ReadLoadBalancerTagsRequest(readLoadBalancerTagRequest).Execute()
		utils.LogAPICall(ctx, "ReadLoadBalancerTags", readLoadBalancerTagRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}
			requestStr := fmt.Sprintf("%v", readLoadBalancerTagRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, fmt.Errorf("failed to read Tag Name: %w", err)
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
func (s *Service) CreateLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag *osc.ResourceTag) error {
	createLoadBalancerTagRequest := osc.CreateLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceTag{*loadBalancerTag},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	createLoadBalancerTagCallBack := func() (bool, error) {
		_, httpRes, err := oscApiClient.LoadBalancerApi.CreateLoadBalancerTags(oscAuthClient).CreateLoadBalancerTagsRequest(createLoadBalancerTagRequest).Execute()
		utils.LogAPICall(ctx, "CreateLoadBalancerTags", createLoadBalancerTagRequest, httpRes, err)
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
			return false, fmt.Errorf("failed to add Tag: %w", err)
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
func (s *Service) CreateLoadBalancer(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error) {
	loadBalancerType := spec.LoadBalancerType
	backendPort := spec.Listener.BackendPort
	loadBalancerPort := spec.Listener.LoadBalancerPort
	backendProtocol := spec.Listener.BackendProtocol
	loadBalancerProtocol := spec.Listener.LoadBalancerProtocol

	first_listener := osc.ListenerForCreation{
		BackendPort:          backendPort,
		BackendProtocol:      &backendProtocol,
		LoadBalancerPort:     loadBalancerPort,
		LoadBalancerProtocol: loadBalancerProtocol,
	}

	loadBalancerRequest := osc.CreateLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
		LoadBalancerType: &loadBalancerType,
		Listeners:        []osc.ListenerForCreation{first_listener},
		SecurityGroups:   &[]string{securityGroupId},
		Subnets:          &[]string{subnetId},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var loadBalancerResponse osc.CreateLoadBalancerResponse
	createLoadBalancerCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		loadBalancerResponse, httpRes, err = oscApiClient.LoadBalancerApi.CreateLoadBalancer(oscAuthClient).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
		utils.LogAPICall(ctx, "CreateLoadBalancer", loadBalancerRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
		return nil, waitErr
	}
	loadBalancer, ok := loadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("Can not create loadbalancer")
	}
	return loadBalancer, nil
}

// DeleteLoadBalancer delete the loadbalancer
func (s *Service) DeleteLoadBalancer(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) error {
	deleteLoadBalancerRequest := osc.DeleteLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteLoadBalancerCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.DeleteLoadBalancer(oscAuthClient).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
		utils.LogAPICall(ctx, "DeleteLoadBalancer", deleteLoadBalancerRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) DeleteLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error {
	deleteLoadBalancerTagRequest := osc.DeleteLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceLoadBalancerTag{loadBalancerTag},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteLoadBalancerTagCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.LoadBalancerApi.DeleteLoadBalancerTags(oscAuthClient).DeleteLoadBalancerTagsRequest(deleteLoadBalancerTagRequest).Execute()
		utils.LogAPICall(ctx, "DeleteLoadBalancerTags", deleteLoadBalancerTagRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
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
