/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package security

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
