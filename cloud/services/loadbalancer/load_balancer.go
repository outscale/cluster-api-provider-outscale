/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package loadbalancer

import (
	"context"
	"errors"
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

//go:generate ../../../bin/mockgen -destination mock_loadbalancer/loadbalancer_mock.go -package mock_loadbalancer -source ./load_balancer.go
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

// ConfigureHealthCheck update loadBalancer to configure healthCheck
// Keep backoff: secondary call to CreateLoadBalancer
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

	var updateLoadBalancerResponse osc.UpdateLoadBalancerResponse
	updateLoadBalancerCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		updateLoadBalancerResponse, httpRes, err = s.tenant.Client().LoadBalancerApi.UpdateLoadBalancer(s.tenant.ContextWithAuth(ctx)).UpdateLoadBalancerRequest(updateLoadBalancerRequest).Execute()
		err = utils.LogAndExtractError(ctx, "UpdateLoadBalancer", updateLoadBalancerRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) || httpRes == nil {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, updateLoadBalancerCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	loadBalancer, ok := updateLoadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("cannot update loadbalancer")
	}
	return loadBalancer, nil
}

// LinkLoadBalancerBackendMachines link the loadBalancer with vm backend
func (s *Service) LinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error {
	linkLoadBalancerBackendMachinesRequest := osc.LinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}

	linkLoadBalancerBackendMachinesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = s.tenant.Client().LoadBalancerApi.LinkLoadBalancerBackendMachines(s.tenant.ContextWithAuth(ctx)).LinkLoadBalancerBackendMachinesRequest(linkLoadBalancerBackendMachinesRequest).Execute()
		err = utils.LogAndExtractError(ctx, "LinkLoadBalancerBackendMachines", linkLoadBalancerBackendMachinesRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) {
				klog.FromContext(ctx).V(4).Error(err, "retrying on error")
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
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

	_, httpRes, err := s.tenant.Client().LoadBalancerApi.UnlinkLoadBalancerBackendMachines(s.tenant.ContextWithAuth(ctx)).UnlinkLoadBalancerBackendMachinesRequest(unlinkLoadBalancerBackendMachinesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "UnlinkLoadBalancerBackendMachines", unlinkLoadBalancerBackendMachinesRequest, httpRes, err)
	return err
}

// GetLoadBalancer retrieve loadBalancer object from spec
func (s *Service) GetLoadBalancer(ctx context.Context, loadBalancerName string) (*osc.LoadBalancer, error) {
	filterLoadBalancer := osc.FiltersLoadBalancer{
		LoadBalancerNames: &[]string{loadBalancerName},
	}
	readLoadBalancerRequest := osc.ReadLoadBalancersRequest{Filters: &filterLoadBalancer}

	readLoadBalancersResponse, httpRes, err := s.tenant.Client().LoadBalancerApi.ReadLoadBalancers(s.tenant.ContextWithAuth(ctx)).ReadLoadBalancersRequest(readLoadBalancerRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadLoadBalancers", readLoadBalancerRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	var lb []osc.LoadBalancer
	loadBalancers, ok := readLoadBalancersResponse.GetLoadBalancersOk()
	if !ok {
		return nil, errors.New("cannot get loadbalancer")
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
	readLoadBalancerTagRequest := osc.ReadLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
	}
	readLoadBalancerTagsResponse, httpRes, err := s.tenant.Client().LoadBalancerApi.ReadLoadBalancerTags(s.tenant.ContextWithAuth(ctx)).ReadLoadBalancerTagsRequest(readLoadBalancerTagRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadLoadBalancerTags", readLoadBalancerTagRequest, httpRes, err)
	if err != nil {
		return nil, err
	}

	var tag []osc.LoadBalancerTag
	tags, ok := readLoadBalancerTagsResponse.GetTagsOk()
	if !ok {
		return nil, errors.New("cannot get tags")
	}
	if len(*tags) == 0 {
		return nil, nil
	} else {
		tag = append(tag, *tags...)
		return &tag[0], nil
	}
}

// CreateLoadBalancerTag create the load balancer tag
// Keep backoff for now, secondary call to CreateLoadBalancer.
func (s *Service) CreateLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag *osc.ResourceTag) error {
	createLoadBalancerTagRequest := osc.CreateLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceTag{*loadBalancerTag},
	}

	createLoadBalancerTagCallBack := func() (bool, error) {
		_, httpRes, err := s.tenant.Client().LoadBalancerApi.CreateLoadBalancerTags(s.tenant.ContextWithAuth(ctx)).CreateLoadBalancerTagsRequest(createLoadBalancerTagRequest).Execute()
		err = utils.LogAndExtractError(ctx, "CreateLoadBalancerTags", createLoadBalancerTagRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
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

	loadBalancerResponse, httpRes, err := s.tenant.Client().LoadBalancerApi.CreateLoadBalancer(s.tenant.ContextWithAuth(ctx)).CreateLoadBalancerRequest(loadBalancerRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateLoadBalancer", loadBalancerRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	loadBalancer, ok := loadBalancerResponse.GetLoadBalancerOk()
	if !ok {
		return nil, errors.New("cannot create loadbalancer")
	}
	return loadBalancer, nil
}

// DeleteLoadBalancer delete the loadbalancer
func (s *Service) DeleteLoadBalancer(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer) error {
	deleteLoadBalancerRequest := osc.DeleteLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
	}

	_, httpRes, err := s.tenant.Client().LoadBalancerApi.DeleteLoadBalancer(s.tenant.ContextWithAuth(ctx)).DeleteLoadBalancerRequest(deleteLoadBalancerRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteLoadBalancer", deleteLoadBalancerRequest, httpRes, err)
	return err
}

// DeleteLoadBalancerTag delete the loadbalancerTag
func (s *Service) DeleteLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta1.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error {
	deleteLoadBalancerTagRequest := osc.DeleteLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceLoadBalancerTag{loadBalancerTag},
	}

	_, httpRes, err := s.tenant.Client().LoadBalancerApi.DeleteLoadBalancerTags(s.tenant.ContextWithAuth(ctx)).DeleteLoadBalancerTagsRequest(deleteLoadBalancerTagRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteLoadBalancerTags", deleteLoadBalancerTagRequest, httpRes, err)
	return err
}
