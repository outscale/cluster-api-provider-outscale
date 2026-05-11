/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

//go:generate ../../../bin/mockgen -destination mock_net/loadbalancer_mock.go -package mock_net -source ./load_balancer.go
type OscLoadBalancerInterface interface {
	ConfigureHealthCheck(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer) (*osc.LoadBalancer, error)
	GetLoadBalancer(ctx context.Context, loadBalancerName string) (*osc.LoadBalancer, error)
	CreateLoadBalancer(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer) error
	LinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error
	UnlinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error
	CreateLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, loadBalancerTag *osc.ResourceTag) error
	DeleteLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error
}

// ConfigureHealthCheck update loadBalancer to configure healthCheck
// Keep backoff: secondary call to CreateLoadBalancer
func (s *Service) ConfigureHealthCheck(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer) (*osc.LoadBalancer, error) {
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
	req := osc.UpdateLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
		HealthCheck:      &healthCheck,
	}

	resp, err := s.tenant.Client().UpdateLoadBalancer(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.LoadBalancer, nil
}

// LinkLoadBalancerBackendMachines link the loadBalancer with vm backend
func (s *Service) LinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error {
	linkLoadBalancerBackendMachinesRequest := osc.LinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	_, err := s.tenant.Client().LinkLoadBalancerBackendMachines(ctx, linkLoadBalancerBackendMachinesRequest)
	return err
}

// UnlinkLoadBalancerBackendMachines unlink the loadbalancer with vm backend
func (s *Service) UnlinkLoadBalancerBackendMachines(ctx context.Context, vmIds []string, loadBalancerName string) error {
	req := osc.UnlinkLoadBalancerBackendMachinesRequest{
		BackendVmIds:     &vmIds,
		LoadBalancerName: loadBalancerName,
	}
	_, err := s.tenant.Client().UnlinkLoadBalancerBackendMachines(ctx, req)
	return err
}

// GetLoadBalancer retrieve loadBalancer object from spec
func (s *Service) GetLoadBalancer(ctx context.Context, loadBalancerName string) (*osc.LoadBalancer, error) {
	req := osc.ReadLoadBalancersRequest{Filters: &osc.FiltersLoadBalancer{
		LoadBalancerNames: &[]string{loadBalancerName},
	}}

	resp, err := s.tenant.Client().ReadLoadBalancers(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.LoadBalancers) == 0:
		return nil, nil
	default:
		return &(*resp.LoadBalancers)[0], nil
	}
}

// CreateLoadBalancerTag create the load balancer tag
// Keep backoff for now, secondary call to CreateLoadBalancer.
func (s *Service) CreateLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, loadBalancerTag *osc.ResourceTag) error {
	req := osc.CreateLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceTag{*loadBalancerTag},
	}
	_, err := s.tenant.Client().CreateLoadBalancerTags(ctx, req)
	return err
}

// CreateLoadBalancer create the load balancer
func (s *Service) CreateLoadBalancer(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, subnetId string, securityGroupId string) (*osc.LoadBalancer, error) {
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

	req := osc.CreateLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
		LoadBalancerType: &loadBalancerType,
		Listeners:        []osc.ListenerForCreation{first_listener},
		SecurityGroups:   &[]string{securityGroupId},
		Subnets:          &[]string{subnetId},
	}

	resp, err := s.tenant.Client().CreateLoadBalancer(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.LoadBalancer, nil
}

// DeleteLoadBalancer delete the loadbalancer
func (s *Service) DeleteLoadBalancer(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer) error {
	req := osc.DeleteLoadBalancerRequest{
		LoadBalancerName: spec.LoadBalancerName,
	}
	_, err := s.tenant.Client().DeleteLoadBalancer(ctx, req)
	return err
}

// DeleteLoadBalancerTag delete the loadbalancerTag
func (s *Service) DeleteLoadBalancerTag(ctx context.Context, spec *infrastructurev1beta2.OscLoadBalancer, loadBalancerTag osc.ResourceLoadBalancerTag) error {
	req := osc.DeleteLoadBalancerTagsRequest{
		LoadBalancerNames: []string{spec.LoadBalancerName},
		Tags:              []osc.ResourceLoadBalancerTag{loadBalancerTag},
	}
	_, err := s.tenant.Client().DeleteLoadBalancerTags(ctx, req)
	return err
}
