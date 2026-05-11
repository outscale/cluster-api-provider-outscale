/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type Tenant interface {
	Region() string
	Client() *osc.Client
}
