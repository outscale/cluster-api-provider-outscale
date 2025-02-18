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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	base64 "encoding/base64"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
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
	defaultKeyPairInitialize = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			KeyPair: infrastructurev1beta1.OscKeypair{
				Name:      "test-keypair",
				PublicKey: generateSSHPublicKey(),
			},
		},
	}
)

func generateSSHPublicKey() string {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(ssh.MarshalAuthorizedKey(publicKey))
}

// SetupWithKeyPairMock set keyPairMock with clusterScope and osccluster
func SetupWithKeyPairMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscKeyPairInterface *mock_security.MockOscKeyPairInterface) {
	clusterScope, machineScope = SetupMachine(t, name, spec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscKeyPairInterface = mock_security.NewMockOscKeyPairInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscKeyPairInterface
}

// TestGetKeyPairResourceId has several tests to cover the code of the function getKeyPairResourceId
func TestGetKeyPairResourceId(t *testing.T) {
	keyPairTestCases := []struct {
		name                       string
		spec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                infrastructurev1beta1.OscMachineSpec
		expKeyPairFound            bool
		expGetKeyPairResourceIdErr error
	}{
		{
			name:                       "get keyPairID",
			spec:                       defaultKeyClusterInitialize,
			machineSpec:                defaultKeyPairInitialize,
			expKeyPairFound:            true,
			expGetKeyPairResourceIdErr: nil,
		},
		{
			name: "can not get keyPairID",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:            false,
			expGetKeyPairResourceIdErr: errors.New(" does not exist"),
		},
	}

	for _, k := range keyPairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope, _, _ := SetupWithKeyPairMock(t, k.name, k.spec, k.machineSpec)
			keyPairSpec := k.machineSpec.Node.KeyPair
			keyPairName := keyPairSpec.Name
			keyPairRef := machineScope.GetKeypairRef()
			if keyPairName != "" {
				keyPairRef.ResourceMap = make(map[string]string)
				keyPairRef.ResourceMap[keyPairName] = keyPairName
			}
			keyPairResourceID, err := getKeyPairResourceId(keyPairName, machineScope)
			if k.expGetKeyPairResourceIdErr != nil {
				require.EqualError(t, err, k.expGetKeyPairResourceIdErr.Error(), "get should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find keyPairResourceID %s\n", keyPairResourceID)
		})
	}
}

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
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "test-keypair",
						PublicKey: generateSSHPublicKey(),
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
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "!test-keypair@Name",
						PublicKey: generateSSHPublicKey(),
					},
				},
			},
			expCheckKeyPairFormatParametersErr: errors.New("Invalid Tag Name"),
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, k.name, k.clusterSpec, k.machineSpec)
			keyPairName, err := checkKeypairFormatParameters(machineScope)
			if k.expCheckKeyPairFormatParametersErr != nil {
				require.EqualError(t, err, k.expCheckKeyPairFormatParametersErr.Error(), "checkKeyPairFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find keyPairName %s\n", keyPairName)
		})
	}
}

func TestCheckKeypairSameName(t *testing.T) {
	keypairTestCases := []struct {
		name                       string
		clusterSpec                infrastructurev1beta1.OscClusterSpec
		machineSpec                infrastructurev1beta1.OscMachineSpec
		expCheckKeypairSameNameErr error
	}{
		{
			name:        "check the same keypair name",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name: "test-keypair",
					},
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-keypair",
					},
				},
			},
			expCheckKeypairSameNameErr: nil,
		},
		{
			name:        "check not have the same keypair name from keypair section",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name: "test-bad-keypair",
					},
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-keypair",
					},
				},
			},
			expCheckKeypairSameNameErr: errors.New("test-bad-keypair is not the same in vm and keypair section"),
		},
		{
			name:        "check not have the same keypair name from vm section",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name: "test-keypair",
					},
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-bad-keypair",
					},
				},
			},
			expCheckKeypairSameNameErr: errors.New("test-keypair is not the same in vm and keypair section"),
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, k.name, k.clusterSpec, k.machineSpec)
			err := checkKeypairSameName(machineScope)
			if k.expCheckKeypairSameNameErr != nil {
				require.EqualError(t, err, k.expCheckKeypairSameNameErr.Error(), "checkKeypairSameName() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Got the same keypair name %s in both vm and keypair section \n", k.machineSpec.Node.Vm.KeypairName)
		})
	}
}

// TestReconcileKeyPairGet has several tests to cover the code of the function reconcileKeyPair
func TestReconcileKeyPairGet(t *testing.T) {
	keypairTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		machineSpec            infrastructurev1beta1.OscMachineSpec
		expKeyPairFound        bool
		expValidateKeyPairs    bool
		expGetKeyPairErr       error
		expReconcileKeyPairErr error
	}{
		{
			name: "check keypair exist",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "test-keypairValid",
						PublicKey: "00",
					},
				},
			},
			expKeyPairFound:     true,
			expValidateKeyPairs: true,
		},
		{
			name: "failed to get keypair",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:        false,
			expValidateKeyPairs:    false,
			expGetKeyPairErr:       errors.New("GetKeyPair generic error"),
			expReconcileKeyPairErr: errors.New("cannot get keypair: GetKeyPair generic error"),
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscKeyPairInterface := SetupWithKeyPairMock(t, k.name, k.spec, k.machineSpec)
			keyPairSpec := k.machineSpec.Node.KeyPair
			keyPairName := keyPairSpec.Name
			keyPairRef := machineScope.GetKeypairRef()
			keyPairRef.ResourceMap = make(map[string]string)
			keyPairRef.ResourceMap[keyPairName] = keyPairSpec.ResourceId
			key := osc.ReadKeypairsResponse{
				Keypairs: &[]osc.Keypair{
					{
						KeypairName: &keyPairName,
					},
				},
			}
			// keyPairCreated := osc.CreateKeypairResponse{
			// 	Keypair: &osc.KeypairCreated{
			// 		KeypairName: &keyPairName,
			// 	},
			// }
			keyPairSpec.ResourceId = keyPairName
			mockOscKeyPairInterface.
				EXPECT().
				GetKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
				Return(&(*key.Keypairs)[0], k.expGetKeyPairErr)

			// SA4022: the address of a variable cannot be nil
			// if &(*key.Keypairs)[0] == nil {
			// 	mockOscKeyPairInterface.
			// 		EXPECT().
			// 		CreateKeyPair(gomock.Eq(keyPairName)).
			// 		Return(keyPairCreated.Keypair, k.expReconcileKeyPairErr)
			// }

			reconcileKeyPair, err := reconcileKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if k.expReconcileKeyPairErr != nil {
				require.EqualError(t, err, k.expReconcileKeyPairErr.Error(), "reconcileKeyPair() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileKeyPair)
		})
	}
}

// TestReconcileKeyPairCreate has several tests to cover the code of the function reconcileKeyPair
func TestReconcileKeyPairCreate(t *testing.T) {
	keypairTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		machineSpec            infrastructurev1beta1.OscMachineSpec
		expKeyPairFound        bool
		expValidateKeyPairs    bool
		expGetKeyPairErr       error
		expCreateKeyPairFound  bool
		expCreateKeyPairErr    error
		expReconcileKeyPairErr error
	}{
		{
			name: "failed to create keypair ",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expValidateKeyPairs:    false,
			expCreateKeyPairFound:  false,
			expCreateKeyPairErr:    errors.New("CreateKeyPair failed"),
			expReconcileKeyPairErr: errors.New("cannot create keypair: CreateKeyPair failed"),
		},

		{
			name: "create keypair (first time reconcile loop)",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:        false,
			expValidateKeyPairs:    false,
			expGetKeyPairErr:       errors.New("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
			expCreateKeyPairFound:  true,
			expCreateKeyPairErr:    nil,
			expReconcileKeyPairErr: nil,
		},

		{
			name: "user delete keypair without cluster-api",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:        false,
			expValidateKeyPairs:    false,
			expGetKeyPairErr:       errors.New("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
			expCreateKeyPairFound:  false,
			expCreateKeyPairErr:    nil,
			expReconcileKeyPairErr: nil,
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscKeyPairInterface := SetupWithKeyPairMock(t, k.name, k.spec, k.machineSpec)
			keyPairSpec := k.machineSpec.Node.KeyPair
			keyPairName := keyPairSpec.Name

			keyPairCreated := osc.CreateKeypairResponse{
				Keypair: &osc.KeypairCreated{
					KeypairName: &keyPairName,
				},
			}

			if k.expCreateKeyPairFound {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
					Return(nil, nil)
				mockOscKeyPairInterface.
					EXPECT().
					CreateKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
					Return(keyPairCreated.Keypair, k.expCreateKeyPairErr) // keypair to becreated
			} else {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
					Return(nil, nil)
				mockOscKeyPairInterface.
					EXPECT().
					CreateKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
					Return(nil, k.expCreateKeyPairErr)
			}

			reconcileKeyPair, err := reconcileKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if k.expReconcileKeyPairErr != nil {
				require.EqualError(t, err, k.expReconcileKeyPairErr.Error(), "reconcileKeyPair() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileKeyPair)
		})
	}
}

// TestReconcileDeleteKeyPair tests key pair deletion during delete reconciliation.
func TestReconcileDeleteKeyPair(t *testing.T) {
	keypairTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		machineSpec                  infrastructurev1beta1.OscMachineSpec
		expReconcileDeleteKeyPairErr error
		expGetKeyPair                bool
		expGetKeyPairErr             error
		expDeleteKeyPair             bool
		expDeleteKeyPairErr          error
	}{
		{
			name:        "failed to delete keypair removed outside cluster api",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:          "test-keypair",
						DeleteKeypair: true,
					},
				},
			},
			expGetKeyPair:                true,
			expGetKeyPairErr:             errors.New("GetKeyPair generic error"),
			expReconcileDeleteKeyPairErr: errors.New("cannot get keypair: GetKeyPair generic error"),
		},
		{
			name:        "no keypair",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{},
				},
			},
		},
		{
			name:        "failed to delete keypair",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:          "test-keypair",
						DeleteKeypair: true,
					},
				},
			},

			expGetKeyPair:                true,
			expDeleteKeyPair:             true,
			expDeleteKeyPairErr:          errors.New("Can not delete keypair"),
			expReconcileDeleteKeyPairErr: errors.New("cannot delete keypair: Can not delete keypair"),
		},
		{
			name:        "delete keypair",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:          "test-keypair",
						PublicKey:     "00",
						DeleteKeypair: true,
					},
				},
			},

			expGetKeyPair:    true,
			expDeleteKeyPair: true,
		},
		// FIXME: this cannot happen - if keypair is not found, `nil, error` is returned, not `nil, nil`
		// {
		// 	name:        "can not find keypair",
		// 	clusterSpec: defaultKeyClusterInitialize,
		// 	machineSpec: infrastructurev1beta1.OscMachineSpec{
		// 		Node: infrastructurev1beta1.OscNode{
		// 			KeyPair: infrastructurev1beta1.OscKeypair{
		// 				Name:          "test-keypair",
		// 				PublicKey:     "00",
		// 				DeleteKeypair: true,
		// 			},
		// 		},
		// 	},

		// 	expGetKeyPair: true,
		// },
		{
			name:        "keep keypair",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name: "test-keypair",
					},
				},
			},
		},
	}

	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscKeyPairInterface := SetupWithKeyPairMock(t, k.name, k.clusterSpec, k.machineSpec)
			keyPairSpec := k.machineSpec.Node.KeyPair
			keyPairName := keyPairSpec.Name

			keypair := &osc.Keypair{
				KeypairName: &keyPairName,
			}
			if k.expGetKeyPair {
				if k.expGetKeyPairErr == nil {
					mockOscKeyPairInterface.
						EXPECT().
						GetKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
						Return(keypair, nil)
				} else {
					mockOscKeyPairInterface.
						EXPECT().
						GetKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
						Return(nil, k.expGetKeyPairErr)
				}
			}
			if k.expDeleteKeyPair {
				mockOscKeyPairInterface.
					EXPECT().
					DeleteKeyPair(gomock.Any(), gomock.Eq(keyPairName)).
					Return(k.expDeleteKeyPairErr)
			}

			reconcileDeleteKeyPair, err := reconcileDeleteKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if k.expReconcileDeleteKeyPairErr != nil {
				require.EqualError(t, err, k.expReconcileDeleteKeyPairErr.Error(), "reconcileKeyPair() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileDeleteKeyPair)
		})
	}
}
