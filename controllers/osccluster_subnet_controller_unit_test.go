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

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net/mock_net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
)

var (
	defaultSubnetInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
				},
			},
		},
	}
	defaultSubnetReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
		},
	}
	defaultSubnetReconcileWithSkipReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-uid",
					SkipReconcile: true,
				},
			},
		},
	}

	defaultMultiSubnetInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet-first",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
					SubregionName: "eu-west-2b",
				},
			},
		},
	}
	defaultMultiSubnetInitializeWithSkipReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet-first",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					SkipReconcile: true,
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
					SubregionName: "eu-west-2b",
					SkipReconcile: true,
				},
			},
		},
	}
	defaultMultiSubnetReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet-first",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-first-uid",
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
					SubregionName: "eu-west-2b",
					ResourceId:    "subnet-test-subnet-second-uid",
				},
			},
		},
	}
	defaultMultiSubnetReconcileWithSkipReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet-first",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-first-uid",
					SkipReconcile: true,
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
					SubregionName: "eu-west-2b",
					ResourceId:    "subnet-test-subnet-second-uid",
					SkipReconcile: true,
				},
			},
		},
	}
)

// SetupWithSubnetMock set subnetMock with clusterScope and osccluster
func SetupWithSubnetMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscSubnetInterface *mock_net.MockOscSubnetInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscSubnetInterface = mock_net.NewMockOscSubnetInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscSubnetInterface, mockOscTagInterface
}

// TestGetSubnetResourceId has several tests to cover the code of the function getSubnetResourceId
func TestGetSubnetResourceId(t *testing.T) {
	subnetTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expSubnetFound            bool
		expGetSubnetResourceIdErr error
	}{
		{
			name:                      "get SubnetId",
			spec:                      defaultSubnetInitialize,
			expSubnetFound:            true,
			expGetSubnetResourceIdErr: nil,
		},
		{
			name:                      "failed to get Subnet",
			spec:                      defaultSubnetInitialize,
			expSubnetFound:            false,
			expGetSubnetResourceIdErr: errors.New("test-subnet-uid does not exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope := Setup(t, stc.name, stc.spec)
			subnetsSpec := stc.spec.Network.Subnets
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				if stc.expSubnetFound {
					subnetRef := clusterScope.GetSubnetRef()
					subnetRef.ResourceMap = make(map[string]string)
					subnetRef.ResourceMap[subnetName] = subnetId
				}
				subnetResourceId, err := getSubnetResourceId(subnetName, clusterScope)
				if stc.expGetSubnetResourceIdErr != nil {
					require.EqualError(t, err, stc.expGetSubnetResourceIdErr.Error(), "getSubnetResourceId() should return the same error")
				} else {
					require.NoError(t, err)
				}
				t.Logf("Find subnetResourceId %s\n", subnetResourceId)
			}
		})
	}
}

// TestCheckSubnetOscDuplicateName has several tests to cover the code of the func checkSubnetOscDuplicateName
func TestCheckSubnetOscDuplicateName(t *testing.T) {
	subnetTestCases := []struct {
		name                              string
		spec                              infrastructurev1beta1.OscClusterSpec
		expCheckSubnetOscDuplicateNameErr error
	}{

		{
			name:                              "get separate Name",
			spec:                              defaultMultiSubnetInitialize,
			expCheckSubnetOscDuplicateNameErr: nil,
		},
		{
			name: "get duplicate Name",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet-first",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
						},
						{
							Name:          "test-subnet-first",
							IpSubnetRange: "10.0.1.0/24",
							SubregionName: "eu-west-2b",
						},
					},
				},
			},
			expCheckSubnetOscDuplicateNameErr: errors.New("test-subnet-first already exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope := Setup(t, stc.name, stc.spec)
			err := checkSubnetOscDuplicateName(clusterScope)
			if stc.expCheckSubnetOscDuplicateNameErr != nil {
				require.EqualError(t, err, stc.expCheckSubnetOscDuplicateNameErr.Error(), "checkSubnetOscDupicateName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckSubnetFormatParameters has several tests to cover the code of the function checkSubnetFormatParameters

func TestCheckSubnetFormatParameters(t *testing.T) {
	subnetTestCases := []struct {
		name                              string
		spec                              infrastructurev1beta1.OscClusterSpec
		expCheckSubnetFormatParametersErr error
	}{
		{
			name: "Check work without spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckSubnetFormatParametersErr: nil,
		},
		{
			name:                              "check Subnet format",
			spec:                              defaultSubnetInitialize,
			expCheckSubnetFormatParametersErr: nil,
		},
		{
			name: "check Bad Name subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet@test",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: errors.New("Invalid Tag Name"),
		},
		{
			name: "check Bad Ip Range Prefix subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/36",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: errors.New("invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "check Bad Ip Range Ip subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.256/16",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: errors.New("invalid CIDR address: 10.0.0.256/16"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope := Setup(t, stc.name, stc.spec)
			subnetName, err := checkSubnetFormatParameters(clusterScope)
			if stc.expCheckSubnetFormatParametersErr != nil {
				require.EqualError(t, err, stc.expCheckSubnetFormatParametersErr.Error(), "checkSubnetFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find subnetName %s\n", subnetName)
		})
	}
}

// TestReconcileSubnetCreate has several tests to cover the code of the function reconcileSubnet
func TestReconcileSubnetCreate(t *testing.T) {
	subnetTestCases := []struct {
		name                  string
		spec                  infrastructurev1beta1.OscClusterSpec
		expSubnetFound        bool
		expNetFound           bool
		expTagFound           bool
		expCreateSubnetFound  bool
		expCreateSubnetErr    error
		expGetSubnetIdsErr    error
		expReconcileSubnetErr error
		expReadTagErr         error
	}{
		{
			name:                  "create Subnet (first time reconcile loop)",
			spec:                  defaultSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           false,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
			expReadTagErr:         nil,
		},
		{
			name:                  "create two Subnets (first time reconcile loop)",
			spec:                  defaultMultiSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           false,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
			expReadTagErr:         nil,
		},
		{
			name:                  "create two Subnets with skip reconcile (first time reconcile loop)",
			spec:                  defaultMultiSubnetInitializeWithSkipReconcile,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           false,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
			expReadTagErr:         nil,
		},
		{
			name:                  "failed to create subnet",
			spec:                  defaultSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           false,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    errors.New("CreateSubnet generic error"),
			expGetSubnetIdsErr:    nil,
			expReadTagErr:         nil,
			expReconcileSubnetErr: errors.New("cannot create subnet: CreateSubnet generic error"),
		},
		{
			name:                  "user delete subnet without cluster-api",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           false,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReadTagErr:         nil,
			expReconcileSubnetErr: nil,
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, mockOscTagInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			clusterName := stc.spec.Network.ClusterName + "-uid"

			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			subnetsSpec := stc.spec.Network.Subnets
			var subnetIds []string
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				tag := osc.Tag{
					ResourceId: &subnetId,
				}
				if stc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(subnetName)).
						Return(&tag, stc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(subnetName)).
						Return(nil, stc.expReadTagErr)
				}
				subnetIds = append(subnetIds, subnetId)
				subnet := osc.CreateSubnetResponse{
					Subnet: &osc.Subnet{
						SubnetId: &subnetId,
					},
				}

				subnetRef := clusterScope.GetSubnetRef()
				subnetRef.ResourceMap = make(map[string]string)
				if stc.expCreateSubnetFound {
					subnetRef.ResourceMap[subnetName] = subnetId
					mockOscSubnetInterface.
						EXPECT().
						CreateSubnet(gomock.Any(), gomock.Eq(subnetSpec), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(subnetName)).
						Return(subnet.Subnet, stc.expCreateSubnetErr)
				} else {
					mockOscSubnetInterface.
						EXPECT().
						CreateSubnet(gomock.Any(), gomock.Eq(subnetSpec), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(subnetName)).
						Return(nil, stc.expCreateSubnetErr)
				}
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface, mockOscTagInterface)
			if stc.expReconcileSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileSubnetErr.Error(), "reconcileSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileSubnet  %v\n", reconcileSubnet)
		})
	}
}

// TestReconcileSubnetGet has several tests to cover the code of the function reconcileSubnet
func TestReconcileSubnetGet(t *testing.T) {
	subnetTestCases := []struct {
		name                  string
		spec                  infrastructurev1beta1.OscClusterSpec
		expSkipReconcile      bool
		expSubnetFound        bool
		expNetFound           bool
		expTagFound           bool
		expGetSubnetIdsErr    error
		expReadTagErr         error
		expReconcileSubnetErr error
	}{
		{
			name:                  "check Subnet exist (second time reconcile loop)",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        true,
			expNetFound:           true,
			expTagFound:           true,
			expGetSubnetIdsErr:    nil,
			expReadTagErr:         nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:                  "check two subnets exist (second time reconcile loop)",
			spec:                  defaultMultiSubnetReconcile,
			expSubnetFound:        true,
			expNetFound:           true,
			expTagFound:           true,
			expGetSubnetIdsErr:    nil,
			expReadTagErr:         nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:               "skip reconciliation loop",
			spec:               defaultMultiSubnetReconcileWithSkipReconcile,
			expSkipReconcile:   true,
			expNetFound:        true,
			expGetSubnetIdsErr: nil,
		},
		{
			name:                  "failed to get Subnet",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        false,
			expNetFound:           true,
			expTagFound:           true,
			expGetSubnetIdsErr:    errors.New("GetSubnet generic error"),
			expReadTagErr:         nil,
			expReconcileSubnetErr: errors.New("GetSubnet generic error"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, mockOscTagInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			subnetsSpec := stc.spec.Network.Subnets
			var subnetIds []string
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				tag := osc.Tag{
					ResourceId: &subnetId,
				}
				if !stc.expSkipReconcile {
					if stc.expSubnetFound {
						if stc.expTagFound {
							mockOscTagInterface.
								EXPECT().
								ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(subnetName)).
								Return(&tag, stc.expReadTagErr)
						} else {
							mockOscTagInterface.
								EXPECT().
								ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(subnetName)).
								Return(nil, stc.expReadTagErr)
						}
					}
					subnetIds = append(subnetIds, subnetId)
				}
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface, mockOscTagInterface)
			if stc.expReconcileSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileSubnetErr.Error(), "ReconcileSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileSubnet  %v\n", reconcileSubnet)
		})
	}
}

// TestReconcileSubnetResourceId has several tests to cover the code of the function reconcileSubnet
func TestReconcileSubnetResourceId(t *testing.T) {
	subnetTestCases := []struct {
		name                  string
		spec                  infrastructurev1beta1.OscClusterSpec
		expTagFound           bool
		expNetFound           bool
		expReadTagErr         error
		expGetSubnetIdsErr    error
		expReconcileSubnetErr error
	}{
		{
			name:                  "Net does not exist",
			spec:                  defaultSubnetReconcile,
			expTagFound:           false,
			expNetFound:           false,
			expReadTagErr:         nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: errors.New("test-net-uid does not exist"),
		},
		{
			name:                  "failed to get tag",
			spec:                  defaultSubnetReconcile,
			expTagFound:           true,
			expNetFound:           true,
			expReadTagErr:         errors.New("ReadTag generic error"),
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: errors.New("cannot get tag: ReadTag generic error"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, mockOscTagInterface := SetupWithSubnetMock(t, stc.name, stc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			subnetsSpec := stc.spec.Network.Subnets
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			var subnetIds []string
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				subnetIds = append(subnetIds, subnetId)
				var subnetIds []string
				tag := osc.Tag{
					ResourceId: &subnetId,
				}
				if stc.expTagFound {
					mockOscSubnetInterface.
						EXPECT().
						GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
						Return(subnetIds, stc.expGetSubnetIdsErr)

					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(subnetName)).
						Return(&tag, stc.expReadTagErr)
				}
			}

			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface, mockOscTagInterface)
			if stc.expReconcileSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileSubnetErr.Error(), "reconcileSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileSubnet  %v\n", reconcileSubnet)
		})
	}
}

// TestReconcileDeleteSubnetGet has several tests to cover the code of the function reconcileDeleteSubnet
func TestReconcileDeleteSubnetGet(t *testing.T) {
	subnetTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expSkipReconcile            bool
		expSubnetFound              bool
		expNetFound                 bool
		expGetSubnetIdsErr          error
		expReconcileDeleteSubnetErr error
	}{
		{
			name:                        "Failed to get subnet",
			spec:                        defaultSubnetReconcile,
			expSubnetFound:              false,
			expNetFound:                 true,
			expGetSubnetIdsErr:          errors.New("GetSubnet generic error"),
			expReconcileDeleteSubnetErr: errors.New("GetSubnet generic error"),
		},
		{
			name:             "skip deletion reconciliation loop",
			spec:             defaultSubnetReconcileWithSkipReconcile,
			expSkipReconcile: true,
		},
		{
			name:                        "Remove finalizer (User delete Subnets without cluster-api)",
			spec:                        defaultSubnetReconcile,
			expSubnetFound:              false,
			expNetFound:                 true,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, _ := SetupWithSubnetMock(t, stc.name, stc.spec)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			subnetsSpec := stc.spec.Network.Subnets
			var subnetIds []string
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				subnetIds = append(subnetIds, subnetId)
			}
			if !stc.expSkipReconcile {
				if stc.expSubnetFound {
					mockOscSubnetInterface.
						EXPECT().
						GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
						Return(subnetIds, stc.expGetSubnetIdsErr)
				} else {
					mockOscSubnetInterface.
						EXPECT().
						GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
						Return(nil, stc.expGetSubnetIdsErr)
				}
			}

			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if stc.expReconcileDeleteSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileDeleteSubnetErr.Error(), "reconcileDeleteSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}

// TestReconcileDeleteSubnetDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteSubnet
func TestReconcileDeleteSubnetDeleteWithoutSpec(t *testing.T) {
	subnetTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expDeleteSubnetErr          error
		expGetSubnetIdsErr          error
		expReconcileDeleteSubnetErr error
	}{
		{
			name:                        "delete Net without spec (with default values)",
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, _ := SetupWithSubnetMock(t, stc.name, stc.spec)
			netName := "cluster-api-net-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			clusterScope.OscCluster.Spec.Network.Net.ResourceId = netId

			var subnetIds []string
			subnetName := "cluster-api-subnet-uid"
			subnetId := "subnet-" + subnetName
			subnetIds = append(subnetIds, subnetId)
			mockOscSubnetInterface.
				EXPECT().
				GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
				Return(subnetIds, stc.expGetSubnetIdsErr)
			mockOscSubnetInterface.
				EXPECT().
				DeleteSubnet(gomock.Any(), gomock.Eq(subnetId)).
				Return(stc.expDeleteSubnetErr)
			networkSpec := clusterScope.GetNetwork()
			networkSpec.SetSubnetDefaultValue()
			clusterScope.OscCluster.Spec.Network.Subnets[0].ResourceId = subnetId
			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if stc.expReconcileDeleteSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileDeleteSubnetErr.Error(), "reconcileDeleteSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}

// TestReconcileDeleteSubnetDelete has several tests to cover the code of the function reconcileDeleteSubnet
func TestReconcileDeleteSubnetDelete(t *testing.T) {
	subnetTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expSubnetFound              bool
		expNetFound                 bool
		expDeleteSubnetErr          error
		expGetSubnetIdsErr          error
		expReconcileDeleteSubnetErr error
	}{
		{
			name:                        "delete Net (first time reconcile loop)",
			spec:                        defaultSubnetReconcile,
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
		{
			name:                        "delete two Net (first time reconcile loop)",
			spec:                        defaultMultiSubnetReconcile,
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
		{
			name:                        "failed to delete Subnet",
			spec:                        defaultSubnetReconcile,
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          errors.New("DeleteSubnet generic error"),
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: errors.New("cannot delete subnet: DeleteSubnet generic error"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, _ := SetupWithSubnetMock(t, stc.name, stc.spec)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			subnetsSpec := stc.spec.Network.Subnets
			var subnetIds []string
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				subnetIds = append(subnetIds, subnetId)
				mockOscSubnetInterface.
					EXPECT().
					DeleteSubnet(gomock.Any(), gomock.Eq(subnetId)).
					Return(stc.expDeleteSubnetErr)
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}

			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if stc.expReconcileDeleteSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileDeleteSubnetErr.Error(), "reconcileDeleteSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}

// TestReconcileDeleteSubnet_NoNetKnown tests that reconciliation suceeds if no net is known
func TestReconcileDeleteSubnet_NoNetKnown(t *testing.T) {
	subnetTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expReconcileDeleteSubnetErr error
	}{
		{
			name: "Net does not exist",
			spec: defaultSubnetReconcile,
		},
		{
			name: "check failed without net and subnet spec (retrieve default values cluster-api)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface, _ := SetupWithSubnetMock(t, stc.name, stc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if stc.expReconcileDeleteSubnetErr != nil {
				require.EqualError(t, err, stc.expReconcileDeleteSubnetErr.Error(), "reconcileDeleteSubnet() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}
