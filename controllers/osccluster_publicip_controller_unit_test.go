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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
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
	defaultLinkVmInitialize = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Vm: infrastructurev1beta1.OscVm{
				PublicIpName: "test-publicip",
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
func SetupWithPublicIpMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscPublicIpInterface *mock_security.MockOscPublicIpInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscPublicIpInterface = mock_security.NewMockOscPublicIpInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscPublicIpInterface, mockOscTagInterface
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
			expGetPublicIpResourceIdErr: errors.New("test-publicip-uid does not exist"),
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
				if pitc.expGetPublicIpResourceIdErr != nil {
					require.EqualError(t, err, pitc.expGetPublicIpResourceIdErr.Error(), "getPublicIpResourceId() should return the same error")
				} else {
					require.NoError(t, err)
				}
				t.Logf("Find publicIpResourceId %s\n", publicIpResourceId)
			}
		})
	}
}

// TestLinkPublicIpResourceId has several tests to cover the code of the function getLinkPublicIpResourceId
func TestLinkPublicIpResourceId(t *testing.T) {
	linkPublicIpTestCases := []struct {
		name                            string
		clusterSpec                     infrastructurev1beta1.OscClusterSpec
		machineSpec                     infrastructurev1beta1.OscMachineSpec
		expLinkPublicIpFound            bool
		expGetLinkPublicIpResourceIdErr error
	}{
		{
			name:                            "get publicIpId",
			clusterSpec:                     defaultPublicIpInitialize,
			machineSpec:                     defaultLinkVmInitialize,
			expLinkPublicIpFound:            true,
			expGetLinkPublicIpResourceIdErr: nil,
		},
		{
			name:                            "can not get publicIpId",
			clusterSpec:                     defaultPublicIpInitialize,
			machineSpec:                     defaultLinkVmInitialize,
			expLinkPublicIpFound:            false,
			expGetLinkPublicIpResourceIdErr: errors.New("test-publicip does not exist"),
		},
	}
	for _, lpitc := range linkPublicIpTestCases {
		t.Run(lpitc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, lpitc.name, lpitc.clusterSpec, lpitc.machineSpec)
			publicIpsSpec := lpitc.clusterSpec.Network.PublicIps
			vmPublicIpName := lpitc.machineSpec.Node.Vm.PublicIpName
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				linkPublicIpId := "eipassoc-" + publicIpName
				linkPublicIpRef := machineScope.GetLinkPublicIpRef()
				linkPublicIpRef.ResourceMap = make(map[string]string)
				if lpitc.expLinkPublicIpFound {
					linkPublicIpRef.ResourceMap[vmPublicIpName] = linkPublicIpId
				}
				linkPublicIpResourceId, err := getLinkPublicIpResourceId(vmPublicIpName, machineScope)
				if lpitc.expGetLinkPublicIpResourceIdErr != nil {
					require.EqualError(t, err, lpitc.expGetLinkPublicIpResourceIdErr.Error(), "getLinkPublicIpResourceId() should return the same error")
				} else {
					require.NoError(t, err)
				}
				t.Logf("Find linkPublicIpResourceId %s\n", linkPublicIpResourceId)
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
			expCheckPublicIpFormatParametersErr: errors.New("Invalid Tag Name"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			publicIpName, err := checkPublicIpFormatParameters(clusterScope)
			if pitc.expCheckPublicIpFormatParametersErr != nil {
				require.EqualError(t, err, pitc.expCheckPublicIpFormatParametersErr.Error(), "checkPublicIpFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
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
			expCheckPublicIpOscAssociateResourceNameErr: errors.New("publicIp test-publicip-test-uid does not exist in natService"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			err := checkPublicIpOscAssociateResourceName(clusterScope)
			if pitc.expCheckPublicIpOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, pitc.expCheckPublicIpOscAssociateResourceNameErr.Error(), "checkPublicIpOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
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
			expCheckPublicIpOscDuplicateNameErr: errors.New("test-publicip-first already exist"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope := Setup(t, pitc.name, pitc.spec)
			err := checkPublicIpOscDuplicateName(clusterScope)
			if pitc.expCheckPublicIpOscDuplicateNameErr != nil {
				require.EqualError(t, err, pitc.expCheckPublicIpOscDuplicateNameErr.Error(), "checkPublicOscDuplicateName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestReconcilePublicIpGet has several tests to cover the code of the function reconcilePublicIp
func TestReconcilePublicIpGet(t *testing.T) {
	publicIpTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta1.OscClusterSpec
		notManagedByCapi        bool
		expSkip                 bool
		expPublicIpFound        bool
		expTagFound             bool
		expValidatePublicIpsErr error
		expReadTagErr           error
		expReconcilePublicIpErr error
	}{
		{
			name:                    "check publicIp exist (second time reconcile loop)",
			spec:                    defaultPublicIpReconcile,
			expPublicIpFound:        true,
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expReadTagErr:           nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "check two publicIp exist (second time reconcile loop)",
			spec:                    defaultMultiPublicIpReconcile,
			expPublicIpFound:        true,
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expReadTagErr:           nil,
			expReconcilePublicIpErr: nil,
		},
		{
			name:                    "failed to validate publicIp",
			spec:                    defaultPublicIpInitialize,
			expPublicIpFound:        false,
			expTagFound:             false,
			expValidatePublicIpsErr: errors.New("ValidatePublicIp generic error"),
			expReadTagErr:           nil,
			expReconcilePublicIpErr: errors.New("ValidatePublicIp generic error"),
		},
		{
			name:                    "failed to get tag",
			spec:                    defaultPublicIpReconcile,
			expPublicIpFound:        true,
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expReadTagErr:           errors.New("ReadTag generic error"),
			expReconcilePublicIpErr: errors.New("cannot get tag: ReadTag generic error"),
		},
		{
			name:             "skip reconcile because not managed by capi",
			spec:             defaultInternetServiceReconcile,
			notManagedByCapi: true,
			expSkip:          true,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface, mockOscTagInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var publicIpId string
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId = "eipalloc-" + publicIpName
				if !pitc.notManagedByCapi {
					publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
				}

				tag := osc.Tag{
					ResourceId: &publicIpId,
				}
				if !pitc.expSkip && pitc.expPublicIpFound {
					if pitc.expTagFound {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(publicIpName)).
							Return(&tag, pitc.expReadTagErr)
					} else {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(publicIpName)).
							Return(nil, pitc.expReadTagErr)
					}
				}

				publicIpIds = append(publicIpIds, publicIpId)
				if pitc.expPublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId
				}
			}
			if pitc.expValidatePublicIpsErr != nil {
				publicIpIds = []string{""}
			}
			if !pitc.expSkip {
				if pitc.expPublicIpFound {
					mockOscPublicIpInterface.
						EXPECT().
						ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
						Return(publicIpIds, pitc.expValidatePublicIpsErr)
				} else {
					mockOscPublicIpInterface.
						EXPECT().
						ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
						Return(nil, pitc.expValidatePublicIpsErr)
				}
			}
			reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope, mockOscPublicIpInterface, mockOscTagInterface)
			if pitc.expReconcilePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcilePublicIpErr.Error(), "reconcilePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
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
		notManagedByCapi        bool
		expPublicIpFound        bool
		expTagFound             bool
		expValidatePublicIpsErr error
		expCreatePublicIpFound  bool
		expCreatePublicIpErr    error
		expReadTagErr           error
		expReconcilePublicIpErr error
		expManagedByCapi        bool
	}{
		{
			name:                    "create publicIp (first time reconcile loop)",
			spec:                    defaultPublicIpInitialize,
			notManagedByCapi:        true,
			expPublicIpFound:        false,
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReadTagErr:           nil,
			expReconcilePublicIpErr: nil,
			expManagedByCapi:        true,
		},
		{
			name:                    "create two publicIp (first time reconcile loop)",
			spec:                    defaultMultiPublicIpInitialize,
			notManagedByCapi:        true,
			expPublicIpFound:        false,
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReadTagErr:           nil,
			expReconcilePublicIpErr: nil,
			expManagedByCapi:        true,
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
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  false,
			expCreatePublicIpErr:    errors.New("CreatePublicIp generic error"),
			expReadTagErr:           nil,
			expReconcilePublicIpErr: errors.New("cannot create publicIp: CreatePublicIp generic error"),
			expManagedByCapi:        true,
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
			expTagFound:             false,
			expValidatePublicIpsErr: nil,
			expCreatePublicIpFound:  true,
			expCreatePublicIpErr:    nil,
			expReadTagErr:           nil,
			expReconcilePublicIpErr: nil,
			expManagedByCapi:        true,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface, mockOscTagInterface := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for index, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				if !pitc.notManagedByCapi {
					publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
				}
				tag := osc.Tag{
					ResourceId: &publicIpId,
				}
				if pitc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(publicIpName)).
						Return(&tag, pitc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(publicIpName)).
						Return(nil, pitc.expReadTagErr)
				}
				publicIpIds = append(publicIpIds, publicIpId)

				publicIp := osc.CreatePublicIpResponse{
					PublicIp: &osc.PublicIp{
						PublicIpId: &publicIpId,
					},
				}
				if pitc.expCreatePublicIpFound {
					publicIpRef.ResourceMap[publicIpName] = publicIpId

					publicIpIds[index] = ""
					mockOscPublicIpInterface.
						EXPECT().
						CreatePublicIp(gomock.Any(), gomock.Eq(publicIpName)).
						Return(publicIp.PublicIp, pitc.expCreatePublicIpErr)
				} else {
					mockOscPublicIpInterface.
						EXPECT().
						CreatePublicIp(gomock.Any(), gomock.Eq(publicIpName)).
						Return(nil, pitc.expCreatePublicIpErr)
				}
			}
			if pitc.expCreatePublicIpErr != nil {
				publicIpIds = []string{""}
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpsErr)
			} else {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpsErr)
			}
			reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope, mockOscPublicIpInterface, mockOscTagInterface)
			if pitc.expReconcilePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcilePublicIpErr.Error(), "reconcilePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
			}
			resourceMapValues := make([]string, 0, len(publicIpRef.ResourceMap))
			for _, value := range publicIpRef.ResourceMap {
				resourceMapValues = append(resourceMapValues, value)
			}
			if pitc.expManagedByCapi {
				require.Contains(t, resourceMapValues, infrastructurev1beta1.ManagedByValueCapi)
			} else {
				require.NotContains(t, resourceMapValues, infrastructurev1beta1.ManagedByValueCapi)
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
		notManagedByCapi              bool
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
			clusterScope, ctx, mockOscPublicIpInterface, _ := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			publicIpName := "cluster-api-publicip-nat-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if !pitc.notManagedByCapi {
				publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
			}
			publicIpIds = append(publicIpIds, publicIpId)
			mockOscPublicIpInterface.
				EXPECT().
				ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
				Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			mockOscPublicIpInterface.
				EXPECT().
				CheckPublicIpUnlink(gomock.Any(), gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
				Return(pitc.expCheckPublicIpUnlinkErr)
			mockOscPublicIpInterface.
				EXPECT().
				DeletePublicIp(gomock.Any(), gomock.Eq(publicIpId)).
				Return(pitc.expDeletePublicIpErr)
			networkSpec := clusterScope.GetNetwork()
			networkSpec.SetPublicIpDefaultValue()
			clusterScope.OscCluster.Spec.Network.PublicIps[0].ResourceId = publicIpId
			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)

			if pitc.expReconcileDeletePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcileDeletePublicIpErr.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
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
		notManagedByCapi              bool
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
			spec:                          defaultMultiPublicIpReconcile,
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
			expDeletePublicIpErr:          errors.New("DeletePublicIp generic error"),
			expReconcileDeletePublicIpErr: errors.New("cannot delete publicIp eipalloc-test-publicip-uid: DeletePublicIp generic error"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface, _ := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				if !pitc.notManagedByCapi {
					publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
				}
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIpInterface.
					EXPECT().
					CheckPublicIpUnlink(gomock.Any(), gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIpUnlinkErr)
				mockOscPublicIpInterface.
					EXPECT().
					DeletePublicIp(gomock.Any(), gomock.Eq(publicIpId)).
					Return(pitc.expDeletePublicIpErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpIdsErr)
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)

			if pitc.expReconcileDeletePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcileDeletePublicIpErr.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
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
		notManagedByCapi              bool
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
			expCheckPublicIpUnlinkErr:     errors.New("CheckPublicIpUnlink generic error"),
			expReconcileDeletePublicIpErr: errors.New("cannot check publicIp eipalloc-test-publicip-uid: CheckPublicIpUnlink generic error"),
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface, _ := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				if !pitc.notManagedByCapi {
					publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
				}
				publicIpIds = append(publicIpIds, publicIpId)
				mockOscPublicIpInterface.
					EXPECT().
					CheckPublicIpUnlink(gomock.Any(), gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(publicIpId)).
					Return(pitc.expCheckPublicIpUnlinkErr)
			}
			if pitc.expPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
			} else {
				if len(publicIpIds) == 0 {
					publicIpIds = []string{""}
				}
				mockOscPublicIpInterface.
					EXPECT().
					ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
					Return(nil, pitc.expValidatePublicIpIdsErr)
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)
			if pitc.expReconcileDeletePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcileDeletePublicIpErr.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
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
		notManagedByCapi              bool
		expSkip                       bool
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
			expValidatePublicIpIdsErr:     errors.New("ValidatePublicIp generic error"),
			expReconcileDeletePublicIpErr: errors.New("cannot validate publicips: ValidatePublicIp generic error"),
		},
		{
			name:                          "remove finalizer (user delete publicIp without cluster-api)",
			spec:                          defaultPublicIpReconcile,
			expPublicIpFound:              false,
			expValidatePublicIpIdsErr:     nil,
			expReconcileDeletePublicIpErr: nil,
		},
		{
			name:             "skip reconcile delete because not managed by capi",
			spec:             defaultInternetServiceReconcile,
			notManagedByCapi: true,
			expSkip:          true,
		},
	}
	for _, pitc := range publicIpTestCases {
		t.Run(pitc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscPublicIpInterface, _ := SetupWithPublicIpMock(t, pitc.name, pitc.spec)
			publicIpsSpec := pitc.spec.Network.PublicIps
			var publicIpIds []string
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			for _, publicIpSpec := range publicIpsSpec {
				publicIpName := publicIpSpec.Name + "-uid"
				publicIpId := "eipalloc-" + publicIpName
				if !pitc.notManagedByCapi {
					publicIpRef.ResourceMap[v1beta1.ManagedByKey(publicIpId)] = v1beta1.ManagedByValueCapi
				}
				publicIpIds = append(publicIpIds, publicIpId)
			}
			if !pitc.expSkip {
				if pitc.expPublicIpFound {
					mockOscPublicIpInterface.
						EXPECT().
						ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
						Return(publicIpIds, pitc.expValidatePublicIpIdsErr)
				} else {
					if len(publicIpIds) != 0 {
						mockOscPublicIpInterface.
							EXPECT().
							ValidatePublicIpIds(gomock.Any(), gomock.Eq(publicIpIds)).
							Return(nil, pitc.expValidatePublicIpIdsErr)
					}
				}
			}

			reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, mockOscPublicIpInterface)
			if pitc.expReconcileDeletePublicIpErr != nil {
				require.EqualError(t, err, pitc.expReconcileDeletePublicIpErr.Error(), "reconcileDeletePublicIp() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeletePublicIp %v\n", reconcileDeletePublicIp)
		})
	}
}
