/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta1

import (
	"fmt"
)

const (
	NameOutscaleProviderPrefix = "capo-"
	NameOutscaleProvider       = NameOutscaleProviderPrefix + "cluster-"
	APIServerRoleTagValue      = "apiserver"
	NodeRoleTagValue           = "node"
)

// ClusterTagKey add cluster tag key
func ClusterTagKey(name string) string {
	return fmt.Sprintf("%s%s", NameOutscaleProvider, name)
}
