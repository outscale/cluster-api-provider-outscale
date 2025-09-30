/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package utils

var (
	version = "dev"
)

// GetVersion retrieves the version of the provider
func GetVersion() string {
	return version
}
