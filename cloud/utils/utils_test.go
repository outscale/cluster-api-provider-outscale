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
