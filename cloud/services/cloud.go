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
	NetPeering(t tenant.Tenant) net.OscNetPeeringInterface
	NetAccessPoint(t tenant.Tenant) net.OscNetAccessPointInterface
	Subnet(t tenant.Tenant) net.OscSubnetInterface
	SecurityGroup(t tenant.Tenant) security.OscSecurityGroupInterface

	InternetService(t tenant.Tenant) net.OscInternetServiceInterface
	RouteTable(t tenant.Tenant) security.OscRouteTableInterface
	NatService(t tenant.Tenant) net.OscNatServiceInterface
	PublicIp(t tenant.Tenant) security.OscPublicIpInterface
	LoadBalancer(t tenant.Tenant) loadbalancer.OscLoadBalancerInterface

	VM(t tenant.Tenant) compute.OscVmInterface
	Image(t tenant.Tenant) compute.OscImageInterface
	FlexibleGPU(t tenant.Tenant) compute.OscFGPUInterface

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

// Net returns the Net service
func (s *Services) Net(t tenant.Tenant) net.OscNetInterface {
	return net.NewService(t)
}

// NetPeering returns the NetPeering service
func (s *Services) NetPeering(t tenant.Tenant) net.OscNetPeeringInterface {
	return net.NewService(t)
}

// NetAccessPoint returns the NetAccessPoint service
func (s *Services) NetAccessPoint(t tenant.Tenant) net.OscNetAccessPointInterface {
	return net.NewService(t)
}

// Subnet returns the Subnet interface
func (s *Services) Subnet(t tenant.Tenant) net.OscSubnetInterface {
	return net.NewService(t)
}

// InternetService returns an internetService service
func (s *Services) InternetService(t tenant.Tenant) net.OscInternetServiceInterface {
	return net.NewService(t)
}

// RouteTable returns a routeTable service
func (s *Services) RouteTable(t tenant.Tenant) security.OscRouteTableInterface {
	return security.NewService(t)
}

// SecurityGroup returns a securityGroup service
func (s *Services) SecurityGroup(t tenant.Tenant) security.OscSecurityGroupInterface {
	return security.NewService(t)
}

// NatService returns a natService service
func (s *Services) NatService(t tenant.Tenant) net.OscNatServiceInterface {
	return net.NewService(t)
}

// VM returns a VM service
func (s *Services) VM(t tenant.Tenant) compute.OscVmInterface {
	return compute.NewService(t)
}

// Image returns an image service
func (s *Services) Image(t tenant.Tenant) compute.OscImageInterface {
	return compute.NewService(t)
}

// FlexibleGpu returns a fGPU service
func (s *Services) FlexibleGPU(t tenant.Tenant) compute.OscFGPUInterface {
	return compute.NewService(t)
}

// PublicIp returns a public IP service
func (s *Services) PublicIp(t tenant.Tenant) security.OscPublicIpInterface {
	return security.NewService(t)
}

// LoadBalancer returns a loadBalancer service
func (s *Services) LoadBalancer(t tenant.Tenant) loadbalancer.OscLoadBalancerInterface {
	return loadbalancer.NewService(t)
}

// Tag service returns a tag service
func (s *Services) Tag(t tenant.Tenant) tag.OscTagInterface {
	return tag.NewService(t)
}

var _ Servicer = (*Services)(nil)
