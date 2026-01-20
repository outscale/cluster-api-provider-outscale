/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package utils

import (
	"fmt"
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/osc-sdk-go/v2"
)

func ConvertsTagsToUserDataOutscaleSection(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	b := &strings.Builder{}
	_, _ = fmt.Fprintln(b, "-----BEGIN OUTSCALE SECTION-----")
	_, _ = fmt.Fprintln(b, "filter_private_section=true")
	for key, value := range tags {
		key = strings.TrimPrefix(key, "tags.")
		_, _ = fmt.Fprintf(b, "tags.%s=%s\n", key, value)
	}
	_, _ = fmt.Fprintln(b, "-----END OUTSCALE SECTION-----")
	return b.String()
}

func RoleTags(roles []infrastructurev1beta1.OscRole) []osc.ResourceTag {
	rs := make([]osc.ResourceTag, 0, len(roles))
	for _, role := range roles {
		rs = append(rs, osc.ResourceTag{
			Key: "OscK8sRole/" + string(role),
		})
		// CCM 0.4 compatibility.
		switch role {
		case infrastructurev1beta1.RoleService:
			rs = append(rs, osc.ResourceTag{
				Key: "kubernetes.io/role/elb",
			})
		case infrastructurev1beta1.RoleInternalService:
			rs = append(rs, osc.ResourceTag{
				Key: "kubernetes.io/role/internal-elb",
			})
		}
	}
	return rs
}
