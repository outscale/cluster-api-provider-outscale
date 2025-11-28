/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import "github.com/outscale/cluster-api-provider-outscale/cloud/utils"

func userAgent() string {
	return "cluster-api-provider-outscale/" + utils.GetVersion()
}
