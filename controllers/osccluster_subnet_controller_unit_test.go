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

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"
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
				ResourceId: "vpc-test-net",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					ResourceId:    "subnet-test-subnet-uid",
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
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
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
					ResourceId:    "subnet-test-subnet-first-uid",
				},
				{
					Name:          "test-subnet-second",
					IpSubnetRange: "10.0.1.0/24",
					ResourceId:    "subnet-test-subnet-second-uid",
				},
			},
		},
	}
)

// SetupWithSubnetMock set subnetMock with clusterScope and osccluster
func SetupWithSubnetMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscSubnetInterface *mock_net.MockOscSubnetInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscSubnetInterface = mock_net.NewMockOscSubnetInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscSubnetInterface
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
			expGetSubnetResourceIdErr: fmt.Errorf("test-subnet-uid does not exist"),
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
				if err != nil {
					assert.Equal(t, stc.expGetSubnetResourceIdErr, err, "getSubnetResourceId() should return the same error")
				} else {
					assert.Nil(t, stc.expGetSubnetResourceIdErr)

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
						},
						{
							Name:          "test-subnet-first",
							IpSubnetRange: "10.0.1.0/24",
						},
					},
				},
			},
			expCheckSubnetOscDuplicateNameErr: fmt.Errorf("test-subnet-first already exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope := Setup(t, stc.name, stc.spec)
			duplicateResourceSubnetErr := checkSubnetOscDuplicateName(clusterScope)
			if duplicateResourceSubnetErr != nil {
				assert.Equal(t, stc.expCheckSubnetOscDuplicateNameErr, duplicateResourceSubnetErr, "checkSubnetOscDupicateName() should return the same error")
			} else {
				assert.Nil(t, stc.expCheckSubnetOscDuplicateNameErr)
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
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
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
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.0/36"),
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
						},
					},
				},
			},
			expCheckSubnetFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.256/16"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope := Setup(t, stc.name, stc.spec)
			subnetName, err := checkSubnetFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, stc.expCheckSubnetFormatParametersErr.Error(), err.Error(), "checkSubnetFormatParameters() should return the same error")
			} else {
				assert.Nil(t, stc.expCheckSubnetFormatParametersErr)
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
		expCreateSubnetFound  bool
		expCreateSubnetErr    error
		expGetSubnetIdsErr    error
		expReconcileSubnetErr error
	}{
		{
			name:                  "create Subnet (first time reconcile loop)",
			spec:                  defaultSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:                  "create two Subnets (first time reconcile loop)",
			spec:                  defaultMultiSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:                  "failed to create subnet",
			spec:                  defaultSubnetInitialize,
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    fmt.Errorf("CreateSubnet generic error"),
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: fmt.Errorf("CreateSubnet generic error Can not create subnet for Osccluster test-system/test-osc"),
		},
		{
			name:                  "user delete subnet without cluster-api",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
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
						CreateSubnet(gomock.Eq(subnetSpec), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(subnetName)).
						Return(subnet.Subnet, stc.expCreateSubnetErr)
				} else {
					mockOscSubnetInterface.
						EXPECT().
						CreateSubnet(gomock.Eq(subnetSpec), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(subnetName)).
						Return(nil, stc.expCreateSubnetErr)
				}
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileSubnetErr.Error(), err.Error(), "reconcileSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileSubnetErr)
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
		expSubnetFound        bool
		expNetFound           bool
		expGetSubnetIdsErr    error
		expReconcileSubnetErr error
	}{
		{
			name:                  "check Subnet exist (second time reconcile loop)",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        true,
			expNetFound:           true,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:                  "check two subnets exist (second time reconcile loop)",
			spec:                  defaultMultiSubnetReconcile,
			expSubnetFound:        true,
			expNetFound:           true,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name:                  "failed to get Subnet",
			spec:                  defaultSubnetReconcile,
			expSubnetFound:        false,
			expNetFound:           true,
			expGetSubnetIdsErr:    fmt.Errorf("GetSubnet generic error"),
			expReconcileSubnetErr: fmt.Errorf("GetSubnet generic error"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
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
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileSubnetErr, err, "ReconcileSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileSubnetErr)
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
		expReconcileSubnetErr error
	}{
		{
			name:                  "Net does not exist",
			spec:                  defaultSubnetReconcile,
			expReconcileSubnetErr: fmt.Errorf("test-net-uid does not exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileSubnetErr, err, "reconcileSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileSubnetErr)
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
			expGetSubnetIdsErr:          fmt.Errorf("GetSubnet generic error"),
			expReconcileDeleteSubnetErr: fmt.Errorf("GetSubnet generic error"),
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
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
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
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}

			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileDeleteSubnetErr, err, "reconcileDeleteSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileDeleteSubnetErr)
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
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
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
				GetSubnetIdsFromNetIds(gomock.Eq(netId)).
				Return(subnetIds, stc.expGetSubnetIdsErr)
			mockOscSubnetInterface.
				EXPECT().
				DeleteSubnet(gomock.Eq(subnetId)).
				Return(stc.expDeleteSubnetErr)
			networkSpec := clusterScope.GetNetwork()
			networkSpec.SetSubnetDefaultValue()
			clusterScope.OscCluster.Spec.Network.Subnets[0].ResourceId = subnetId
			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileDeleteSubnetErr, err, "reconcileDeleteSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileDeleteSubnetErr)
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
			expDeleteSubnetErr:          fmt.Errorf("DeleteSubnet generic error"),
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: fmt.Errorf("DeleteSubnet generic error Can not delete subnet for Osccluster test-system/test-osc"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)
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
					DeleteSubnet(gomock.Eq(subnetId)).
					Return(stc.expDeleteSubnetErr)
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(subnetIds, stc.expGetSubnetIdsErr)
			} else {
				mockOscSubnetInterface.
					EXPECT().
					GetSubnetIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, stc.expGetSubnetIdsErr)
			}

			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileDeleteSubnetErr.Error(), err.Error(), "reconcileDeleteSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileDeleteSubnetErr)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}

// TestReconcileDeleteSubnetResourceId has several tests to cover the code of the function reconcileDeleteSubnet
func TestReconcileDeleteSubnetResourceId(t *testing.T) {
	subnetTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expReconcileDeleteSubnetErr error
	}{
		{
			name:                        "Net does not exist",
			spec:                        defaultSubnetReconcile,
			expReconcileDeleteSubnetErr: fmt.Errorf("test-net-uid does not exist"),
		},
		{
			name: "check failed without net and subnet spec (retrieve default values cluster-api)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expReconcileDeleteSubnetErr: fmt.Errorf("cluster-api-net-uid does not exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSubnetInterface := SetupWithSubnetMock(t, stc.name, stc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				assert.Equal(t, stc.expReconcileDeleteSubnetErr, err, "reconcileDeleteSubnet() should return the same error")
			} else {
				assert.Nil(t, stc.expReconcileDeleteSubnetErr)
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}
