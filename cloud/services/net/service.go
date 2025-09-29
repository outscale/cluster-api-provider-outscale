/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
)

// Service is a collection of interfaces
type Service struct {
	tenant tenant.Tenant
}

// NewService return a service which is based on outscale api client
func NewService(t tenant.Tenant) *Service {
	return &Service{
		tenant: t,
	}
}
