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
