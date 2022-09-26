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

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	osc "github.com/outscale/osc-sdk-go/v2"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
)

var (
	defaultPublicIpInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
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
				IpRange: "10.0.0.0/16",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
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
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name:       "test-publicip",
					ResourceId: "eipalloc-test-publicip-uid",
				},
			},
		},
	}

	defaultMultiPublicIpReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name:       "test-publicip-first",
					ResourceId: "eipalloc-test-publicip-first-uid",
				},
				{
					Name:       "test-publicip-second",
					ResourceId: "eipalloc-test-publicip-second-uid",
				},
			},
		},
	}
)

// SetupWithPublicIpMock set publicIpMock with clusterScope and osccluster
func SetupWithPublicIpMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscPublicIpInterface *mock_security.MockOscPublicIpInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscPublicIpInterface = mock_security.NewMockOscPublicIpInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscPublicIpInterface
}

// TestGetPublicIpResourceId has several tests to cover the code of the function getPublicIpResourceId
func TestGetPublicIpResourceId(t *testing.T) {
	publicIpTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expPublicIpFound            bool
		expGetPublicIpResourceIdErr error
	}{
		{
			name:                        "get publicIpId",
			spec:                        defaultPublicIpInitialize,
			expPublicIpFound:            true,
			expGetPublicIpResourceIdErr: nil,
		},
		{
			name:                        "can not get publicIpId",
			spec:                        defaultPublicIpInitialize,
			expPublicIpFound:            false,
			expGetPublicIpResourceIdErr: fmt.Errorf("test-publicip-uid does not exist"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpRef := clusterScope.GetPublicIpRef()
				publicIpRef.ResourceMap = make(map[string]string)
				if pitc.expPublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId
				}
				publicIpResourceId, err := getPublicIpResourceId(publicIpName, clusterScope)
				if err != nil {
					assert.Equal(t, pitc.expGetPublicIpResourceIdErr, err, "getPublicIpResourceId() should return the same error")
				} else {
					assert.Nil(t, pitc.expGetPublicIpResourceIdErr)
				}
				t.Logf("Find publicIpResourceId %s\n", publicIpResourceId)
			}
		})
	}
}

// TestCheckPublicIpFormatParameters has several tests to cover the code of the function checkPublicIpFormatParameters
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
						IpRange: "10.0.0.0/16",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip@test",
						},
					},
				},
			},
			expCheckPublicIpFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			publicIpName, err := checkPublicIpFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, pitc.expCheckPublicIpFormatParametersErr, err, "checkPublicIpFormatParameters() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpFormatParametersErr)
			}
			t.Logf("find publicIpName %s\n", publicIpName)
		})
	}
}

// TestCheckPublicIpOscAssociateResourceName has several tests to cover the code of the function checkPublicIpOscAssociateResourceName
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
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIpName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
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
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIpName: "test-publicip-test",
						SubnetName:   "test-subnet",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
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
			err := checkPublicIpOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, pitc.expCheckPublicIpOscAssociateResourceNameErr, err, "checkPublicIpOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckPublicIpOscDuplicateName has several tests to cover the code of the function checkPublicIpOscDuplicateName
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
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIpName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
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
			duplicateResourcePublicIpErr := checkPublicIpOscDuplicateName(clusterScope)
			if duplicateResourcePublicIpErr != nil {
				assert.Equal(t, pitc.expCheckPublicIpOscDuplicateNameErr, duplicateResourcePublicIpErr, "checkPublicOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, pitc.expCheckPublicIpOscDuplicateNameErr)
			}
		})
	}

}

// TestReconcilePublicIpGet has several tests to cover the code of the function reconcilePublicIp
func TestReconcilePublicIpGet(t *testing.T) {
	publicIpTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta1.OscClusterSpec
		expPublicIpFound        bool
		expValidatePublicIpsErr error
		expReconcilePublicIpErr error
	}{
		{
			name:                    "check publicIp exist (second time reconcile loop)",
			spec:                    defaultPublicIpReconcile,
			expPublicIpFound:        true,
			expValidatePublicIpsErr: nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "check two publicIp exist (second time reconcile loop)",
			spec:                    defaultMultiPublicIpReconcile,
			expPublicIpFound:        true,
			expValidatePublicIpsErr: nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "failed to validate publicIp",
			spec:                    defaultPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIpsErr: fmt.Errorf("ValidatePublicIp generic error"),
			expReconcilePublicIpErr: fmt.Errorf("ValidatePublicIp generic error"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var publicIpId string
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId = "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				if pitc.expPublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId
				}

			}
			if pitc.expValidatePublicIpsErr != nil {
				publicIpIds = []string{""}
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpsErr)
			} else {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpsErr)
			}
			reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope, mockOscPublicIpInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcilePublicIpErr, err, "reconcilePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcilePublicIpErr)
			}
			t.Logf("Find reconcilePublicIp %v\n", reconcilePublicIp)
		})
	}
}

// TestReconcilePublicIpCreate has several tests to cover the code of the function reconcilePublicIp
func TestReconcilePublicIpCreate(t *testing.T) {
	publicIpTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta1.OscClusterSpec
		expPublicIpFound        bool
		expValidatePublicIpsErr error
		expCreatePublicIpFound  bool
		expCreatePublicIpErr    error
		expReconcilePublicIpErr error
	}{
		{
			name:                    "create publicIp (first time reconcile loop)",
			spec:                    defaultPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "create two publicIp (first time reconcile loop)",
			spec:                    defaultMultiPublicIpInitialize,
			expPublicIpFound:        false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name: "failed to create publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expPublicIpFound:        false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  false,
			expCreatePublicIpErr:    fmt.Errorf("CreatePublicIp generic error"),
			expReconcilePublicIpErr: fmt.Errorf("CreatePublicIp generic error Can not create publicIp for Osccluster test-system/test-osc"),
		},
		{
			name: "user delete publicIp without cluster-api",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expPublicIpFound:        false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReconcilePublicIpErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
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
				publicIpRef := clusterScope.GetPublicIpRef()
				publicIpRef.ResourceMap = make(map[string]string)
				if pitc.expCreatePublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId

					publicIpIds[index] = ""
					mockOscPublicIpInterface.
						EXPECT().
						CreatePublicIp(gomock.Eq(publicIpName)).
						Return(publicIp.PublicIp, pitc.expCreatePublicIpErr)
				} else {
					mockOscPublicIpInterface.
						EXPECT().
						CreatePublicIp(gomock.Eq(publicIpName)).
						Return(nil, pitc.expCreatePublicIpErr)
				}

			}
			if pitc.expCreatePublicIpErr != nil {
				publicIpIds = []string{""}
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpsErr)
			} else {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpsErr)
			}
			reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope, mockOscPublicIpInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcilePublicIpErr.Error(), err.Error(), "reconcilePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcilePublicIpErr)
			}
			t.Logf("Find reconcilePublicIp %v\n", reconcilePublicIp)
		})
	}
}

// TestReconcileDeletePublicIpDeleteWithoutSpec has several tests to cover the code of the function reconcileDeletePublicIp
func TestReconcileDeletePublicIpDeleteWithoutSpec(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expValidatePublicIpIdsErr     error
		expCheckPublicIpUnlinkErr     error
		expDeletePublicIpErr          error
		expReconcileDeletePublicIpErr error
	}{
		{
			name:                          "delete publicIp without spec (with default values)",
			expValidatePublicIpIdsErr:     nil,
			expDeletePublicIpErr:          nil,
			expCheckPublicIpUnlinkErr:     nil,
			expReconcileDeletePublicIpErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			publicIpName := "cluster-api-publicip-nat-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpIds = append(publicIpIds, publicIpId)
			mockOscPublicIpInterface.
				EXPECT().
				ValidatePublicIpIds(gomock.Eq(publicIpIds)).
				Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			mockOscPublicIpInterface.
				EXPECT().
				CheckPublicIpUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
				Return(pitc.expCheckPublicIpUnlinkErr)
			mockOscPublicIpInterface.
				EXPECT().
				DeletePublicIp(gomock.Eq(publicIpId)).
				Return(pitc.expDeletePublicIpErr)
			networkSpec := clusterScope.GetNetwork()
			networkSpec.SetPublicIpDefaultValue()
			clusterScope.OscCluster.Spec.Network.PublicIps[0].ResourceId = publicIpId
			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIpErr.Error(), err.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIpErr)
			}
			t.Logf("Find reconcileDeletePublicIp %v\n", reconcileDeletePublicIp)
		})
	}
}

// TestReconcileDeletePublicIpDelete has several tests to cover the code of the function reconcileDeletePublicIp
func TestReconcileDeletePublicIpDelete(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIpIdsErr     error
		expCheckPublicIpUnlinkErr     error
		expDeletePublicIpErr          error
		expReconcileDeletePublicIpErr error
	}{
		{
			name:                          "delete publicIp (first time reconcile loop)",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIpIdsErr:     nil,
			expCheckPublicIpUnlinkErr:     nil,
			expDeletePublicIpErr:          nil,
			expReconcileDeletePublicIpErr: nil,
		},
		{
			name:                          "delete two publicIp (first time reconcile loop)",
			spec:                          defaultMultiPublicIpInitialize,
			expPublicIpFound:              true,
			expValidatePublicIpIdsErr:     nil,
			expCheckPublicIpUnlinkErr:     nil,
			expDeletePublicIpErr:          nil,
			expReconcileDeletePublicIpErr: nil,
		},
		{
			name:                          "failed to delete publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIpIdsErr:     nil,
			expCheckPublicIpUnlinkErr:     nil,
			expDeletePublicIpErr:          fmt.Errorf("DeletePublicIp generic error"),
			expReconcileDeletePublicIpErr: fmt.Errorf("DeletePublicIp generic error Can not delete publicIp for Osccluster test-system/test-osc"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIpInterface.
					EXPECT().
					CheckPublicIpUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIpUnlinkErr)
				mockOscPublicIpInterface.
					EXPECT().
					DeletePublicIp(gomock.Eq(publicIpId)).
					Return(pitc.expDeletePublicIpErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpIdsErr)
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIpErr.Error(), err.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIpErr)
			}
			t.Logf("Find reconcileDeletePublicIp %v\n", reconcileDeletePublicIp)
		})
	}
}

// TestReconcileDeletePublicIpCheck has one test to cover the code of the function reconcileDeletePublicIp
func TestReconcileDeletePublicIpCheck(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIpIdsErr     error
		expCheckPublicIpUnlinkErr     error
		expReconcileDeletePublicIpErr error
	}{
		{
			name:                          "failed to delete publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              true,
			expValidatePublicIpIdsErr:     nil,
			expCheckPublicIpUnlinkErr:     fmt.Errorf("CheckPublicIpUnlink generic error"),
			expReconcileDeletePublicIpErr: fmt.Errorf("CheckPublicIpUnlink generic error Can not delete publicIp eipalloc-test-publicip-uid for Osccluster test-system/test-osc"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIpInterface.
					EXPECT().
					CheckPublicIpUnlink(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIpUnlinkErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpIdsErr)
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)

			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIpErr.Error(), err.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIpErr)
			}
			t.Logf("Find reconcileDeletePublicIp %v\n", reconcileDeletePublicIp)
		})
	}
}

// TestReconcileDeletePublicIpGet has several tests to cover the code of the function reconcileDeletePublicIp
func TestReconcileDeletePublicIpGet(t *testing.T) {
	publicIpTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expPublicIpFound              bool
		expValidatePublicIpIdsErr     error
		expReconcileDeletePublicIpErr error
	}{
		{
			name: "check work without publicIp spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expPublicIpFound:              false,
			expValidatePublicIpIdsErr:     nil,
			expReconcileDeletePublicIpErr: nil,
		},
		{
			name:                          "failed to validate publicIp",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              false,
			expValidatePublicIpIdsErr:     fmt.Errorf("ValidatePublicIp generic error"),
			expReconcileDeletePublicIpErr: fmt.Errorf("ValidatePublicIp generic error"),
		},
		{
			name:                          "remove finalizer (user delete publicIp without cluster-api)",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              false,
			expValidatePublicIpIdsErr:     nil,
			expReconcileDeletePublicIpErr: nil,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				publicIpIds = append(publicIpIds, publicIpId)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpIdsErr)
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)
			if err != nil {
				assert.Equal(t, pitc.expReconcileDeletePublicIpErr, err, "reconcileDeletePublicIp() should return the same error")
			} else {
				assert.Nil(t, pitc.expReconcileDeletePublicIpErr)
			}
			t.Logf("Find reconcileDeletePublicIp %v\n", reconcileDeletePublicIp)
		})
	}
}
