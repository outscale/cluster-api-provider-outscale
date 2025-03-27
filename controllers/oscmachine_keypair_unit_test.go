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

package controllers

import (
	"errors"
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/require"
)

var (
	defaultKeyClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
		},
	}
)

// TestCheckKeyPairFormatParameters has several tests to cover the code of the function checkKeyPairFormatParameters
func TestCheckKeyPairFormatParameters(t *testing.T) {
	keypairTestCases := []struct {
		name                               string
		clusterSpec                        infrastructurev1beta1.OscClusterSpec
		machineSpec                        infrastructurev1beta1.OscMachineSpec
		expCheckKeyPairFormatParametersErr error
	}{
		{
			name:        "check keypair format",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-keypair",
					},
				},
			},
			expCheckKeyPairFormatParametersErr: nil,
		},
		{
			name:        "Check work without spec (with default values)",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expCheckKeyPairFormatParametersErr: nil,
		},
		{
			name:        "Check Bad name keypair",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "!test-keypair@Name",
					},
				},
			},
			expCheckKeyPairFormatParametersErr: errors.New("Invalid Tag Name"),
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, k.name, k.clusterSpec, k.machineSpec)
			err := checkKeypairFormatParameters(machineScope)
			if k.expCheckKeyPairFormatParametersErr != nil {
				require.EqualError(t, err, k.expCheckKeyPairFormatParametersErr.Error(), "checkKeyPairFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
