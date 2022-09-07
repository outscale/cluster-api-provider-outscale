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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

var (
	defaultPublicIpInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IPRange: "10.0.0.0/16",
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name: "test-publicip",
				},
			},
		},
	}
	defaultMultiPublicIpInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IPRange: "10.0.0.0/16",
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name: "test-publicip-first",
				},
				{
					Name: "test-publicip-second",
				},
			},
		},
	}

	defaultPublicIpReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IPRange:    "10.0.0.0/16",
				ResourceID: "vpc-test-net-uid",
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name:       "test-publicip",
					ResourceID: "eipalloc-test-publicip-uid",
				},
			},
		},
	}

	defaultMultiPublicIpReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IPRange:    "10.0.0.0/16",
				ResourceID: "vpc-test-net-uid",
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name:       "test-publicip-first",
					ResourceID: "eipalloc-test-publicip-first-uid",
				},
				{
					Name:       "test-publicip-second",
					ResourceID: "eipalloc-test-publicip-second-uid",
				},
			},
		},
	}
)

// SetupWithPublicIpMock set publicIpMock with clusterScope and osccluster
func SetupWithPublicIpMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscPublicIPInterface *mock_security.MockOscPublicIPInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscPublicIPInterface = mock_security.NewMockOscPublicIPInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscPublicIPInterface
}

// TestGetPublicIPResourceID has several tests to cover the code of the function getPublicIPResourceId
func TestGetPublicIPResourceID(t *testing.T) {
	publicIpTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expPublicIpFound            bool
		expGetPublicIPResourceIDErr error
	}{
		{
			name:                        "get publicIpId",
			spec:                        defaultPublicIpInitialize,
			expPublicIpFound:            true,
			expGetPublicIPResourceIDErr: nil,
		},
		{
			name:                        "can not get publicIpId",
			spec:                        defaultPublicIpInitialize,
			expPublicIpFound:            false,
			expGetPublicIPResourceIDErr: fmt.Errorf("test-publicip-uid does not exist"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpRef := clusterScope.GetPublicIPRef()
				publicIpRef.ResourceMap = make(map[string]string)
				if pitc.expPublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId
				}
				publicIpResourceID, err := getPublicIPResourceID(publicIpName, clusterScope)
				if err != nil {
					assert.Equal(t, pitc.expGetPublicIPResourceIDErr, err, "getPublicIPResourceId() should return the same error")
				} else {
					assert.Nil(t, pitc.expGetPublicIPResourceIDErr)
				}
				t.Logf("Find publicIpResourceID %s\n", publicIpResourceID)
			}
		})
	}
}

// TestCheckPublicIpFormatParameters has several tests to cover the code of the function checkPublicIPFormatParameters
func TestCheckPublicIpFormatParameters(t *testing.T) {
	publicIpTestCases := []struct {
		name                                string
		spec                                infrastructurev1beta1.OscClusterSpec
		expCheckPublicIpFormatParametersErr error
	}{
		{
			name: "check work without publicIp spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckPublicIpFormatParametersErr: nil,
		},
		{
			name:                                "check publicIp format",
			spec:                                defaultPublicIpInitialize,
			expCheckPublicIpFormatParametersErr: nil,
		},
		{
			name: "check Bad Name publicip",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip@test",
						},
					},
				},
			},
			expCheckPublicIpFormatParametersErr: fmt.Errorf("invalid Tag Name"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			publicIpName, err := checkPublicIPFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, pitc.expCheckPublicIpFormatParametersErr, err, "checkPublicIPFormatParameters() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpFormatParametersErr)
			}
			t.Logf("find publicIpName %s\n", publicIpName)
		})
	}
}

// TestCheckPublicIpOscAssociateResourceName has several tests to cover the code of the function checkPublicIPOscAssociateResourceName
func TestCheckPublicIpOscAssociateResourceName(t *testing.T) {
	publicIpTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expCheckPublicIpOscAssociateResourceNameErr error
	}{
		{
			name: "check natservice association with publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIPName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckPublicIpOscAssociateResourceNameErr: nil,
		},
		{
			name: "check natService association with bad publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIPName: "test-publicip-test",
						SubnetName:   "test-subnet",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckPublicIpOscAssociateResourceNameErr: fmt.Errorf("publicIp test-publicip-test-uid does not exist in natService "),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			err := checkPublicIPOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, pitc.expCheckPublicIpOscAssociateResourceNameErr, err, "checkPublicIPOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckPublicIpOscDuplicateName has several tests to cover the code of the function checkPublicIPOscDuplicateName
func TestCheckPublicIpOscDuplicateName(t *testing.T) {
	publicIpTestCases := []struct {
		name                                string
		spec                                infrastructurev1beta1.OscClusterSpec
		expCheckPublicIpOscDuplicateNameErr error
	}{
		{
			name:                                "get distinct name",
			spec:                                defaultMultiPublicIpInitialize,
			expCheckPublicIpOscDuplicateNameErr: nil,
		},
		{
			name: "get duplicate Name",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIPName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip-first",
						},
						{
							Name: "test-publicip-first",
						},
					},
				},
			},
			expCheckPublicIpOscDuplicateNameErr: fmt.Errorf("test-publicip-first already exist"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			duplicateResourcePublicIpErr := checkPublicIPOscDuplicateName(clusterScope)
			if duplicateResourcePublicIpErr != nil {
				assert.Equal(t, pitc.expCheckPublicIpOscDuplicateNameErr, duplicateResourcePublicIpErr, "checkPublicOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpOscDuplicateNameErr)
			}
		})
	}

}

// TestReconcilePublicIpGet has several tests to cover the code of the function reconcilePublicIP
func TestReconcilePublicIpGet(t *testing.T) {
	publicIpTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta1.OscClusterSpec
		expPublicIpFound        bool
		expValidatePublicIPSErr error
		expReconcilePublicIpErr error
	}{
		{
			name:                    "check publicIp exist (second time reconcile loop)",
			spec:                    defaultPublicIpReconcile,
			expPublicIpFound:        true,
			expValidatePublicIPSErr: nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "check two publicIp exist (second time reconcile loop)",
			spec:                    defaultMultiPublicIpReconcile,
			expPublicIpFound:        true,
			expValidatePublicIPSErr: nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "failed to validate publicIp",
			spec:                    defaultPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIPSErr: fmt.Errorf("ValidatePublicIp generic error"),
			expReconcilePublicIpErr: fmt.Errorf("ValidatePublicIp generic error"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			var publicIpIds []string
			var publicIpId string
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId = "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				if pitc.expPublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId
				}

			}
			if pitc.expValidatePublicIPSErr != nil {
				publicIpIds = []string{""}
			}
			if pitc.expPublicIpFound {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIPSErr)
			} else {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIPSErr)
			}
			reconcilePublicIP, err := reconcilePublicIP(ctx, clusterScope, mockOscPublicIPInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcilePublicIpErr, err, "reconcilePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcilePublicIpErr)
			}
			t.Logf("Find reconcilePublicIP %v\n", reconcilePublicIP)
		})
	}
}

// TestReconcilePublicIpCreate has several tests to cover the code of the function reconcilePublicIP
func TestReconcilePublicIpCreate(t *testing.T) {
	publicIpTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta1.OscClusterSpec
		expPublicIpFound        bool
		expValidatePublicIPSErr error
		expCreatePublicIPFound  bool
		expCreatePublicIPErr    error
		expReconcilePublicIpErr error
	}{
		{
			name:                    "create publicIp (first time reconcile loop)",
			spec:                    defaultPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIPSErr: nil,
			expCreatePublicIPFound:  true,
			expCreatePublicIPErr:    nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "create two publicIp (first time reconcile loop)",
			spec:                    defaultMultiPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIPSErr: nil,
			expCreatePublicIPFound:  true,
			expCreatePublicIPErr:    nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name: "failed to create publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expPublicIpFound:        false,
			expValidatePublicIPSErr: nil,
			expCreatePublicIPFound:  false,
			expCreatePublicIPErr:    fmt.Errorf("CreatePublicIp generic error"),
			expReconcilePublicIpErr: fmt.Errorf("CreatePublicIp generic error Can not create publicIp for Osccluster test-system/test-osc"),
		},
		{
			name: "user delete publicIp without cluster-api",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expPublicIpFound:        false,
			expValidatePublicIPSErr: nil,
			expCreatePublicIPFound:  true,
			expCreatePublicIPErr:    nil,
			expReconcilePublicIpErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			var publicIpIds []string
			for index, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)

				publicIp := osc.CreatePublicIpResponse{
					PublicIp: &osc.PublicIp{
						PublicIpId: &publicIpId,
					},
				}
				publicIpRef := clusterScope.GetPublicIPRef()
				publicIpRef.ResourceMap = make(map[string]string)
				if pitc.expCreatePublicIPFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId

					publicIpIds[index] = ""
					mockOscPublicIPInterface.
						EXPECT().
						CreatePublicIP(gomock.Eq(publicIpName)).
						Return(publicIp.PublicIp, pitc.expCreatePublicIPErr)
				} else {
					mockOscPublicIPInterface.
						EXPECT().
						CreatePublicIP(gomock.Eq(publicIpName)).
						Return(nil, pitc.expCreatePublicIPErr)
				}

			}
			if pitc.expCreatePublicIPErr != nil {
				publicIpIds = []string{""}
			}
			if pitc.expPublicIpFound {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIPSErr)
			} else {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIPSErr)
			}
			reconcilePublicIP, err := reconcilePublicIP(ctx, clusterScope, mockOscPublicIPInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcilePublicIpErr.Error(), err.Error(), "reconcilePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcilePublicIpErr)
			}
			t.Logf("Find reconcilePublicIP %v\n", reconcilePublicIP)
		})
	}
}

// TestReconcileDeletePublicIPDeleteWithoutSpec has several tests to cover the code of the function reconcileDeletePublicIP
func TestReconcileDeletePublicIPDeleteWithoutSpec(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expValidatePublicIPIdsErr     error
		expCheckPublicIPUnlinkErr     error
		expDeletePublicIPErr          error
		expReconcileDeletePublicIPErr error
	}{
		{
			name:                          "delete publicIp without spec (with default values)",
			expValidatePublicIPIdsErr:     nil,
			expDeletePublicIPErr:          nil,
			expCheckPublicIPUnlinkErr:     nil,
			expReconcileDeletePublicIPErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			publicIpName := "cluster-api-publicip-nat-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpIds = append(publicIpIds, publicIpId)
			mockOscPublicIPInterface.
				EXPECT().
				ValidatePublicIPIds(gomock.Eq(publicIpIds)).
				Return(publicIpIds, pitc.expValidatePublicIPIdsErr)
			mockOscPublicIPInterface.
				EXPECT().
				CheckPublicIPUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
				Return(pitc.expCheckPublicIPUnlinkErr)
			mockOscPublicIPInterface.
				EXPECT().
				DeletePublicIP(gomock.Eq(publicIpId)).
				Return(pitc.expDeletePublicIPErr)
			networkSpec := clusterScope.GetNetwork()
			networkSpec.SetPublicIPDefaultValue()
			clusterScope.OscCluster.Spec.Network.PublicIPS[0].ResourceID = publicIpId
			reconcileDeletePublicIP, err := reconcileDeletePublicIP(ctx, clusterScope, mockOscPublicIPInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIPErr.Error(), err.Error(), "reconcileDeletePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIPErr)
			}
			t.Logf("Find reconcileDeletePublicIP %v\n", reconcileDeletePublicIP)
		})
	}
}

// TestReconcileDeletePublicIPDelete has several tests to cover the code of the function reconcileDeletePublicIP
func TestReconcileDeletePublicIPDelete(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIPIdsErr     error
		expCheckPublicIPUnlinkErr     error
		expDeletePublicIPErr          error
		expReconcileDeletePublicIPErr error
	}{
		{
			name:                          "delete publicIp (first time reconcile loop)",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIPIdsErr:     nil,
			expCheckPublicIPUnlinkErr:     nil,
			expDeletePublicIPErr:          nil,
			expReconcileDeletePublicIPErr: nil,
		},
		{
			name:                          "delete two publicIp (first time reconcile loop)",
			spec:                          defaultMultiPublicIpInitialize,
			expPublicIpFound:              true,
			expValidatePublicIPIdsErr:     nil,
			expCheckPublicIPUnlinkErr:     nil,
			expDeletePublicIPErr:          nil,
			expReconcileDeletePublicIPErr: nil,
		},
		{
			name:                          "failed to delete publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIPIdsErr:     nil,
			expCheckPublicIPUnlinkErr:     nil,
			expDeletePublicIPErr:          fmt.Errorf("DeletePublicIp generic error"),
			expReconcileDeletePublicIPErr: fmt.Errorf("DeletePublicIp generic error Can not delete publicIp for Osccluster test-system/test-osc"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIPInterface.
					EXPECT().
					CheckPublicIPUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIPUnlinkErr)
				mockOscPublicIPInterface.
					EXPECT().
					DeletePublicIP(gomock.Eq(publicIpId)).
					Return(pitc.expDeletePublicIPErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIPIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIPIdsErr)
			}

			reconcileDeletePublicIP, err := reconcileDeletePublicIP(ctx, clusterScope, mockOscPublicIPInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIPErr.Error(), err.Error(), "reconcileDeletePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIPErr)
			}
			t.Logf("Find reconcileDeletePublicIP %v\n", reconcileDeletePublicIP)
		})
	}
}

// TestReconcileDeletePublicIPCheck has one test to cover the code of the function reconcileDeletePublicIP
func TestReconcileDeletePublicIPCheck(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIPIdsErr     error
		expCheckPublicIPUnlinkErr     error
		expReconcileDeletePublicIPErr error
	}{
		{
			name:                          "failed to delete publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIPIdsErr:     nil,
			expCheckPublicIPUnlinkErr:     fmt.Errorf("CheckPublicIpUnlink generic error"),
			expReconcileDeletePublicIPErr: fmt.Errorf("CheckPublicIpUnlink generic error Can not delete publicIp eipalloc-test-publicip-uid for Osccluster test-system/test-osc"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIPInterface.
					EXPECT().
					CheckPublicIPUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIPUnlinkErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIPIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIPIdsErr)
			}

			reconcileDeletePublicIP, err := reconcileDeletePublicIP(ctx, clusterScope, mockOscPublicIPInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIPErr.Error(), err.Error(), "reconcileDeletePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIPErr)
			}
			t.Logf("Find reconcileDeletePublicIP %v\n", reconcileDeletePublicIP)
		})
	}
}

// TestReconcileDeletePublicIPGet has several tests to cover the code of the function reconcileDeletePublicIP
func TestReconcileDeletePublicIPGet(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIPIdsErr     error
		expReconcileDeletePublicIPErr error
	}{
		{
			name: "check work without publicIp spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expPublicIpFound:              false,
			expValidatePublicIPIdsErr:     nil,
			expReconcileDeletePublicIPErr: nil,
		},
		{
			name:                          "failed to validate publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              false,
			expValidatePublicIPIdsErr:     fmt.Errorf("ValidatePublicIp generic error"),
			expReconcileDeletePublicIPErr: fmt.Errorf("ValidatePublicIp generic error"),
		},
		{
			name:                          "remove finalizer (user delete publicIp without cluster-api)",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              false,
			expValidatePublicIPIdsErr:     nil,
			expReconcileDeletePublicIPErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIPInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIPS
			var publicIpIds []string
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIPIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIPInterface.
					EXPECT().
					ValidatePublicIPIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIPIdsErr)
			}

			reconcileDeletePublicIP, err := reconcileDeletePublicIP(ctx, clusterScope, mockOscPublicIPInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIPErr, err, "reconcileDeletePublicIP() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIPErr)
			}
			t.Logf("Find reconcileDeletePublicIP %v\n", reconcileDeletePublicIP)
		})
	}
}
