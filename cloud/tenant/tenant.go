/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"context"

	"github.com/outscale/osc-sdk-go/v2"
)

type Tenant interface {
	Region() string
	ContextWithAuth(context.Context) context.Context
	Client() *osc.APIClient
}
