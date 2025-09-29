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
	assert.Equal(t, "", ConvertsTagsToUserDataOutscaleSection(map[string]string{}))

	expected := `-----BEGIN OUTSCALE SECTION-----
filter_private_section=true
tags.key1=value1
-----END OUTSCALE SECTION-----
`
	assert.Equal(t, expected, ConvertsTagsToUserDataOutscaleSection(map[string]string{"key1": "value1"}))
}
