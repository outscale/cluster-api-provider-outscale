/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package v1beta1_test

import (
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestValidateSubregion(t *testing.T) {
	var tcs = []struct {
		subregion string
		valid     bool
	}{
		{subregion: "eu-west-2a", valid: true},
		{subregion: "eu-west-2b", valid: true},

		{subregion: "cloudgouv-eu-west-1a", valid: true},
		{subregion: "cloudgouv-eu-west-1b", valid: true},
		{subregion: "cloudgouv-eu-west-1c", valid: true},

		{subregion: "us-east-2a", valid: true},
		{subregion: "us-east-2b", valid: true},

		{subregion: "us-west-1a", valid: true},
		{subregion: "us-west-1b", valid: true},

		{subregion: "ap-northeast-1a", valid: true},
		{subregion: "ap-northeast-1b", valid: true},
	}
	for _, tc := range tcs {
		ferr := v1beta1.ValidateSubregion(nil, tc.subregion)
		if tc.valid {
			require.Nil(t, ferr, tc.subregion)
		} else {
			require.NotNil(t, ferr, tc.subregion)
		}
	}
}

func TestValidateVmType(t *testing.T) {
	var tcs = []struct {
		vmType string
		valid  bool
	}{
		{vmType: "tinav0.c1r1p2", valid: false},
		{vmType: "tinav1.c1r1p2", valid: false},
		{vmType: "tinav2.c1r1p2", valid: false},
		{vmType: "tinav3.c1r1p2", valid: true},
		{vmType: "tinav7.c1r2p2", valid: true},
		{vmType: "tinav10.c1r2p2", valid: true},

		{vmType: "tinav7.c0r1p2", valid: false},
		{vmType: "tinav7.c10r1p2", valid: true},

		{vmType: "tinav7.c1r0p2", valid: false},
		{vmType: "tinav7.c1r10p2", valid: true},

		{vmType: "tinav7.c1r1p0", valid: false},
		{vmType: "tinav7.c1r2p1", valid: true},
		{vmType: "tinav7.c1r2p3", valid: true},
		{vmType: "tinav7.c1r1p4", valid: false},

		{vmType: "inference7-l40.small", valid: false},
		{vmType: "inference7-l40.medium", valid: true},
		{vmType: "inference7-l40.large", valid: true},
		{vmType: "inference7-l40.xlarge", valid: false},

		{vmType: "inference7-h100.small", valid: false},
		{vmType: "inference7-h100.medium", valid: true},
		{vmType: "inference7-h100.large", valid: true},
		{vmType: "inference7-h100.xlarge", valid: true},
		{vmType: "inference7-h100.2xlarge", valid: true},

		{vmType: "inference7-h200.small", valid: false},
		{vmType: "inference7-h200.medium", valid: false},
		{vmType: "inference7-h200.large", valid: false},
		{vmType: "inference7-h200.2xsmall", valid: true},
		{vmType: "inference7-h200.2xmedium", valid: true},
		{vmType: "inference7-h200.2xlarge", valid: true},
		{vmType: "inference7-h200.4xlarge", valid: true},
		{vmType: "inference7-h200.4xlargeA", valid: true},
	}
	for _, tc := range tcs {
		ferr := v1beta1.ValidateVmType(nil, tc.vmType)
		if tc.valid {
			require.Nil(t, ferr, tc.vmType)
		} else {
			require.NotNil(t, ferr, tc.vmType)
		}
	}
}
