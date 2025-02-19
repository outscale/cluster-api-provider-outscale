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
		_, err := v1beta1.ValidateSubregionName(tc.subregion)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
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
	}
	for _, tc := range tcs {
		_, err := v1beta1.ValidateVmType(tc.vmType)
		if tc.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
