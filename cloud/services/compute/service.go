/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

//go:generate ../../../bin/mockgen -destination mock_compute/compute_mock.go -package mock_compute -source ./service.go
type Servicer interface {
	FGPUInterface
	ImageInterface
	SecurityGroupInterface
	VmInterface
}

type Service struct {
	tenant tenant.Tenant
	tags   tag.Servicer
}

func NewService(t tenant.Tenant, tags tag.Servicer) *Service {
	return &Service{
		tenant: t,
		tags:   tags,
	}
}

var _ Servicer = (*Service)(nil)
