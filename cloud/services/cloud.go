/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package services

import (
	"fmt"
	"sync"

	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/loadbalancer"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

type Servicer interface {
	DefaultTenant() (tenant.Tenant, error)

	Net(t tenant.Tenant) net.OscNetInterface
	Subnet(t tenant.Tenant) net.OscSubnetInterface
	SecurityGroup(t tenant.Tenant) security.OscSecurityGroupInterface

	InternetService(t tenant.Tenant) net.OscInternetServiceInterface
	RouteTable(t tenant.Tenant) security.OscRouteTableInterface
	NatService(t tenant.Tenant) net.OscNatServiceInterface
	PublicIp(t tenant.Tenant) security.OscPublicIpInterface
	LoadBalancer(t tenant.Tenant) loadbalancer.OscLoadBalancerInterface

	VM(t tenant.Tenant) compute.OscVmInterface
	Image(t tenant.Tenant) compute.OscImageInterface

	Tag(t tenant.Tenant) tag.OscTagInterface
}

type Services struct {
	once          sync.Once
	defaultTenant tenant.Tenant
}

func NewServices() (*Services, error) {
	return &Services{}, nil
}

func (s *Services) DefaultTenant() (tenant.Tenant, error) {
	var err error
	s.once.Do(func() {
		s.defaultTenant, err = tenant.TenantFromEnv()
	})
	if err != nil {
		return nil, fmt.Errorf("tenant from env: %w", err)
	}
	return s.defaultTenant, err
}

// getNetSvc retrieve netSvc
func (s *Services) Net(t tenant.Tenant) net.OscNetInterface {
	return net.NewService(t)
}

// getSubnetSvc retrieve subnetSvc
func (s *Services) Subnet(t tenant.Tenant) net.OscSubnetInterface {
	return net.NewService(t)
}

// getInternetServiceSvc retrieve internetServiceSvc
func (s *Services) InternetService(t tenant.Tenant) net.OscInternetServiceInterface {
	return net.NewService(t)
}

// getRouteTableSvc retrieve routeTableSvc
func (s *Services) RouteTable(t tenant.Tenant) security.OscRouteTableInterface {
	return security.NewService(t)
}

// getSecurityGroupSvc retrieve securityGroupSvc
func (s *Services) SecurityGroup(t tenant.Tenant) security.OscSecurityGroupInterface {
	return security.NewService(t)
}

// getNatServiceSvc retrieve natServiceSvc
func (s *Services) NatService(t tenant.Tenant) net.OscNatServiceInterface {
	return net.NewService(t)
}

// getVmSvc retrieve vmSvc
func (s *Services) VM(t tenant.Tenant) compute.OscVmInterface {
	return compute.NewService(t)
}

// getImageSvc retrieve imageSvc
func (s *Services) Image(t tenant.Tenant) compute.OscImageInterface {
	return compute.NewService(t)
}

// getPublicIpSvc retrieve publicIpSvc
func (s *Services) PublicIp(t tenant.Tenant) security.OscPublicIpInterface {
	return security.NewService(t)
}

// getLoadBalancerSvc retrieve loadBalancerSvc
func (s *Services) LoadBalancer(t tenant.Tenant) loadbalancer.OscLoadBalancerInterface {
	return loadbalancer.NewService(t)
}

// getTagSvc retrieve tagSvc
func (s *Services) Tag(t tenant.Tenant) tag.OscTagInterface {
	return tag.NewService(t)
}

var _ Servicer = (*Services)(nil)
