/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertsTagsToUserDataOutscaleSection(t *testing.T) {
	t.Run("Empty tags generate an empty user data section", func(t *testing.T) {
		assert.Equal(t, "", ConvertsTagsToUserDataOutscaleSection(map[string]string{}))
	})

	t.Run("Generating a user data section with tags", func(t *testing.T) {
		expected := `-----BEGIN OUTSCALE SECTION-----
filter_private_section=true
tags.key1=value1
-----END OUTSCALE SECTION-----
`
		assert.Equal(t, expected, ConvertsTagsToUserDataOutscaleSection(map[string]string{"key1": "value1"}))
	})

	t.Run("tags. prefixes are trimmed", func(t *testing.T) {
		expected := `-----BEGIN OUTSCALE SECTION-----
filter_private_section=true
tags.key1=value1
-----END OUTSCALE SECTION-----
`
		assert.Equal(t, expected, ConvertsTagsToUserDataOutscaleSection(map[string]string{"tags.key1": "value1"}))
	})
}

func TestMergeBootstrapData(t *testing.T) {
	tags := map[string]string{"key1": "value1"}
	outscaleSection := `-----BEGIN OUTSCALE SECTION-----
filter_private_section=true
tags.key1=value1
-----END OUTSCALE SECTION-----
`

	t.Run("cloud-config format prepends Outscale section", func(t *testing.T) {
		data := "#cloud-config\npackages:\n  - nginx\n"
		assert.Equal(t, outscaleSection+data, MergeBootstrapData(tags, data, "cloud-config"))
	})

	t.Run("empty format defaults to cloud-config behavior", func(t *testing.T) {
		data := "#cloud-config\npackages:\n  - nginx\n"
		assert.Equal(t, outscaleSection+data, MergeBootstrapData(tags, data, ""))
	})

	t.Run("ignition format returns bootstrap data as-is", func(t *testing.T) {
		data := `{"ignition":{"version":"3.1.0"}}`
		assert.Equal(t, data, MergeBootstrapData(tags, data, "ignition"))
	})

	t.Run("ignition format with empty tags returns bootstrap data as-is", func(t *testing.T) {
		data := `{"ignition":{"version":"3.1.0"}}`
		assert.Equal(t, data, MergeBootstrapData(map[string]string{}, data, "ignition"))
	})

	t.Run("empty tags with cloud-config returns bootstrap data as-is", func(t *testing.T) {
		data := "#cloud-config\npackages:\n  - nginx\n"
		assert.Equal(t, data, MergeBootstrapData(map[string]string{}, data, "cloud-config"))
	})
}
