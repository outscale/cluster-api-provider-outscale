/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

type Service struct {
	tenant tenant.Tenant
}

func NewService(t tenant.Tenant) *Service {
	return &Service{
		tenant: t,
	}
}
