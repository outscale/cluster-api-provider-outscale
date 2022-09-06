package controllers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	base64 "encoding/base64"

	"fmt"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"

	"testing"
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
	defaultKeyClusterReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
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

	defaultKeyPairReconcile = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			KeyPair: infrastructurev1beta1.OscKeypair{
				Name:       "test-keypair",
				PublicKey:  generateSSHPublicKey(),
				ResourceId: "test-keypair-uid",
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
			expGetKeyPairResourceIdErr: fmt.Errorf(" does not exist"),
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
			if err != nil {
				assert.Equal(t, k.expGetKeyPairResourceIdErr, err, "get should return the same error")
			} else {
				assert.Nil(t, k.expGetKeyPairResourceIdErr)
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
			expCheckKeyPairFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
		{
			name:        "Check empty Public key ",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "test-keypairEmpty",
						PublicKey: "",
					},
				},
			},
			expCheckKeyPairFormatParametersErr: fmt.Errorf("keypair Public Ip is empty"),
		},
		{
			name:        "Check bad Public key ",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "test-keypairWrong",
						PublicKey: "!@@",
					},
				},
			},
			expCheckKeyPairFormatParametersErr: fmt.Errorf("Invalid keypairType"),
		},
	}
	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, k.name, k.clusterSpec, k.machineSpec)
			keyPairName, err := checkKeypairFormatParameters(machineScope)
			if err != nil {
				assert.Equal(t, k.expCheckKeyPairFormatParametersErr, err, "checkKeyPairFormatParameters() should return the same error")
			} else {
				assert.Nil(t, k.expCheckKeyPairFormatParametersErr)
			}
			t.Logf("find keyPairName %s\n", keyPairName)
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
			expKeyPairFound:        true,
			expValidateKeyPairs:    true,
			expReconcileKeyPairErr: nil,
		},
		{
			name: "failed to get keypair",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:        false,
			expValidateKeyPairs:    false,
			expReconcileKeyPairErr: fmt.Errorf("GetKeyPair generic error"),
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
			keyPairSpec.ResourceId = keyPairName

			if k.expKeyPairFound {
				if len(*key.Keypairs) != 0 {
					mockOscKeyPairInterface.
						EXPECT().
						GetKeyPair(gomock.Eq(keyPairName)).
						Return(&(*key.Keypairs)[0], k.expReconcileKeyPairErr)

					mockOscKeyPairInterface.
						EXPECT().
						CreateKeyPair(gomock.Eq(keyPairName)).
						Return(nil, k.expReconcileKeyPairErr)
				}
			} else {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Eq(keyPairName)).
					Return(nil, k.expReconcileKeyPairErr)
			}

			reconcileKeyPair, err := reconcileKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if err != nil {
				assert.Equal(t, k.expReconcileKeyPairErr.Error(), err.Error(), "reconcileKeyPair() should return the same error")
			} else {
				assert.Nil(t, k.expReconcileKeyPairErr)
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
			expCreateKeyPairErr:    fmt.Errorf("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
			expReconcileKeyPairErr: fmt.Errorf("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
		},

		{
			name: "create keypair (first time reconcile loop)",
			spec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expKeyPairFound:        false,
			expValidateKeyPairs:    false,
			expGetKeyPairErr:       fmt.Errorf("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
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
			expGetKeyPairErr:       fmt.Errorf("CreateKeyPair failed Can not create keypair for OscCluster test-system/test-osc"),
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
					GetKeyPair(gomock.Eq(keyPairName)).
					Return(nil, nil)
				mockOscKeyPairInterface.
					EXPECT().
					CreateKeyPair(gomock.Eq(keyPairName)).
					Return(keyPairCreated.Keypair, k.expReconcileKeyPairErr) // keypair to becreated
			} else {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Eq(keyPairName)).
					Return(nil, nil)
				mockOscKeyPairInterface.
					EXPECT().
					CreateKeyPair(gomock.Eq(keyPairName)).
					Return(nil, k.expReconcileKeyPairErr)
			}

			reconcileKeyPair, err := reconcileKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if err != nil {
				assert.Equal(t, k.expReconcileKeyPairErr.Error(), err.Error(), "reconcileKeyPair() should return the same error")
			} else {
				assert.Nil(t, k.expReconcileKeyPairErr)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileKeyPair)
		})
	}
}

// TestReconcileDeleteKeyPairGet has several tests to cover the code of the function reconcileDeleteKeyPair
func TestReconcileDeleteKeyPairGet(t *testing.T) {
	keypairTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		machineSpec                  infrastructurev1beta1.OscMachineSpec
		expReconcileDeleteKeyPairErr error
		expKeyPairFound              bool
		expKeyPairDelete             bool
	}{
		{
			name:        "failed to delete keypair removed outside cluster api ",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},

			expKeyPairFound:              false,
			expReconcileDeleteKeyPairErr: fmt.Errorf("Can not delete keypair for OscCluster test-system/test-osc"),
			expKeyPairDelete:             false,
		},
		{
			name:        "failed to delete keypair ",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},

			expKeyPairFound:              true,
			expReconcileDeleteKeyPairErr: fmt.Errorf("Can not delete keypair for OscCluster test-system/test-osc"),
			expKeyPairDelete:             false,
		},
		{
			name:        "delete keypair ",
			clusterSpec: defaultKeyClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:      "test-keypair",
						PublicKey: "00",
					},
				},
			},

			expKeyPairFound:              true,
			expReconcileDeleteKeyPairErr: nil,
			expKeyPairDelete:             true,
		},
	}

	for _, k := range keypairTestCases {
		t.Run(k.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscKeyPairInterface := SetupWithKeyPairMock(t, k.name, k.clusterSpec, k.machineSpec)
			keyPairSpec := k.machineSpec.Node.KeyPair
			keyPairName := keyPairSpec.Name

			key := osc.ReadKeypairsResponse{
				Keypairs: &[]osc.Keypair{
					{
						KeypairName: &keyPairName,
					},
				},
			}
			if k.expKeyPairFound {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Eq(keyPairName)).
					Return(&(*key.Keypairs)[0], nil)
				if k.expKeyPairDelete {
					mockOscKeyPairInterface.
						EXPECT().
						DeleteKeyPair(gomock.Eq(keyPairName)).
						Return(k.expReconcileDeleteKeyPairErr)
				} else {
					mockOscKeyPairInterface.
						EXPECT().
						DeleteKeyPair(gomock.Eq(keyPairName)).
						Return(k.expReconcileDeleteKeyPairErr)
				}
			} else {
				mockOscKeyPairInterface.
					EXPECT().
					GetKeyPair(gomock.Eq(keyPairName)).
					Return(nil, k.expReconcileDeleteKeyPairErr)
			}

			reconcileDeleteKeyPair, err := reconcileDeleteKeypair(ctx, machineScope, mockOscKeyPairInterface)
			if err != nil {
				assert.Equal(t, k.expReconcileDeleteKeyPairErr.Error(), err.Error(), "reconcileKeyPair() should return the same error")
			} else {
				assert.Nil(t, k.expReconcileDeleteKeyPairErr)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileDeleteKeyPair)
		})
	}
}
