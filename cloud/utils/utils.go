/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
		_, _ = fmt.Fprintf(b, "tags.%s=%s\n", key, value)
	}
	_, _ = fmt.Fprintln(b, "-----END OUTSCALE SECTION-----")
	return b.String()
}

func RoleTags(roles []infrastructurev1beta1.OscRole) []osc.ResourceTag {
	rs := make([]osc.ResourceTag, 0, len(roles))
	for i := range roles {
		rs = append(rs, osc.ResourceTag{
			Key:   "OscK8sRole/" + string(roles[i]),
			Value: "true",
		})
	}
	return rs
}
