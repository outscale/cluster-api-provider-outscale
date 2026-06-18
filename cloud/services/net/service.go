/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

//go:generate ../../../bin/mockgen -destination mock_net/net_mock.go -package mock_net -source ./service.go
type Servicer interface {
	InternetServiceInterface
	LoadBalancerInterface
	NatServiceInterface
	NetInterface
	NetAccessPointInterface
	NetPeeringInterface
	PublicIpInterface
	RouteTableInterface
	SubnetInterface
}

// Service is a collection of interfaces
type Service struct {
	tenant tenant.Tenant
	tags   tag.Servicer
}

// NewService return a service which is based on outscale api client
func NewService(t tenant.Tenant, tags tag.Servicer) *Service {
	return &Service{
		tenant: t,
		tags:   tags,
	}
}

var _ Servicer = (*Service)(nil)
