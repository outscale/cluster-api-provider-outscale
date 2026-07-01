/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package services

import (
	"fmt"
	"sync"

	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

type Servicer interface {
	DefaultTenant() (tenant.Tenant, error)

	Net(t tenant.Tenant) net.Servicer
	Compute(t tenant.Tenant) compute.Servicer
	Tag(t tenant.Tenant) tag.Servicer
}

type Services struct {
	mu            sync.Mutex
	defaultTenant tenant.Tenant
}

func NewServices() (*Services, error) {
	return &Services{}, nil
}

func (s *Services) DefaultTenant() (tenant.Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.defaultTenant == nil {
		var err error
		s.defaultTenant, err = tenant.Default()
		if err != nil {
			return nil, fmt.Errorf("default tenant: %w", err)
		}
	}
	return s.defaultTenant, nil
}

// Net returns the Net service
func (s *Services) Net(t tenant.Tenant) net.Servicer {
	return net.NewService(t, tag.NewService(t))
}

// VM returns a VM service
func (s *Services) Compute(t tenant.Tenant) compute.Servicer {
	return compute.NewService(t, tag.NewService(t))
}

// Tag returns a tag service
func (s *Services) Tag(t tenant.Tenant) tag.Servicer {
	return tag.NewService(t)
}

var _ Servicer = (*Services)(nil)
