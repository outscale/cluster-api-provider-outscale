/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"fmt"

	"github.com/outscale/osc-sdk-go/v3/pkg/profile"
)

func Default() (Tenant, error) {
	prof, err := profile.New()
	if err != nil {
		return nil, fmt.Errorf("tenant from env: %w", err)
	}
	return FromProfile(prof)
}
