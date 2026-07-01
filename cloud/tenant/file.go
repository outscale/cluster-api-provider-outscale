/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"fmt"

	"github.com/outscale/osc-sdk-go/v3/pkg/profile"
)

func FromFile(path, profileName string) (Tenant, error) {
	if profileName == "" {
		profileName = "default"
	}
	prof, err := profile.New(profile.FromFile(profileName, path))
	if err != nil {
		return nil, fmt.Errorf("from file: %w", err)
	}
	return FromProfile(prof)
}
