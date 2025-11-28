/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package utils

import "strings"

var (
	version = "dev"
)

// GetVersion retrieves the version of the provider
func GetVersion() string {
	if !strings.HasPrefix(version, "v") {
		return "dev"
	}
	return version
}
