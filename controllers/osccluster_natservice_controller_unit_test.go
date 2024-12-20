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
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

var (
	defaultNatServiceInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			NatService: infrastructurev1beta1.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
				},
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name: "test-publicip",
				},
			},
		},
	}

	defaultNatServiceReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			NatService: infrastructurev1beta1.OscNatService{Name: "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet-test",
				ResourceId:   "nat-test-natservice-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name:       "test-publicip",
					ResourceId: "eipalloc-test-publicip-uid",
				},
			},
		},
	}
)

// SetupWithNatServiceMock set natServiceMock with clusterScope and osccluster
func SetupWithNatServiceMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscNatServiceInterface *mock_net.MockOscNatServiceInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscNatServiceInterface = mock_net.NewMockOscNatServiceInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscNatServiceInterface, mockOscTagInterface
}

// TestGetNatResourceId has several tests to cover the code of the function getNatResourceId
func TestGetNatResourceId(t *testing.T) {
	natServiceTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		expNatServiceFound     bool
		expGetNatResourceIdErr error
	}{
		{
			name:                   "get natServiceId",
			spec:                   defaultNatServiceInitialize,
			expNatServiceFound:     true,
			expGetNatResourceIdErr: nil,
		},
		{
			name:                   "can not get natServiceId",
			spec:                   defaultNatServiceInitialize,
			expNatServiceFound:     false,
			expGetNatResourceIdErr: fmt.Errorf("test-natservice-uid does not exist"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope := Setup(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if nstc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			natResourceId, err := getNatResourceId(natServiceName, clusterScope)
			if err != nil {
				assert.Equal(t, nstc.expGetNatResourceIdErr, err, "GetNatResourceId() should return the same error")
			} else {
				assert.Nil(t, nstc.expGetNatResourceIdErr)

			}
			t.Logf("find natResourceId %s", natResourceId)
		})
	}
}

// TestCheckNatFormatParameters has several tests to cover the code of the function checkNatFormatParameters
func TestCheckNatFormatParameters(t *testing.T) {
	natServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expCheckNatFormatParametersErr error
	}{
		{
			name: "Check work without natService spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckNatFormatParametersErr: nil,
		},
		{
			name:                           "check natService format",
			spec:                           defaultNatServiceInitialize,
			expCheckNatFormatParametersErr: nil,
		},
		{
			name: "check Bad name natService",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice@test",
						PublicIpName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
		{
			name: "check Bad name publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIpName: "test-publicip@test",
						SubnetName:   "test-subnet",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
		{
			name: "Check BadName subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIpName: "test-publicip",
						SubnetName:   "test-subnet@test",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope := Setup(t, nstc.name, nstc.spec)
			natServiceName, err := checkNatFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, nstc.expCheckNatFormatParametersErr, err, "checkNatFormatParameters() should return the same error")
			} else {
				assert.Nil(t, nstc.expCheckNatFormatParametersErr)
			}
			t.Logf("find natServiceName %s\n", natServiceName)
		})
	}
}

// TestCheckNatSubnetOscAssociateResourceName has several tests to cover the code of the function checkNatSubnetOscAssociateResourceName
func TestCheckNatSubnetOscAssociateResourceName(t *testing.T) {
	natServiceTestCases := []struct {
		name                                         string
		spec                                         infrastructurev1beta1.OscClusterSpec
		expCheckNatSubnetOscAssociateResourceNameErr error
	}{
		{
			name: "check work without natService spec (with default values) ",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckNatSubnetOscAssociateResourceNameErr: fmt.Errorf("cluster-api-subnet-public-uid subnet does not exist in natService"),
		},
		{
			name: "check natService association with subnet",
			spec: defaultNatServiceInitialize,
			expCheckNatSubnetOscAssociateResourceNameErr: nil,
		},
		{
			name: "Check natService association with bad subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice@test",
						PublicIpName: "test-publicip",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: "eu-west-2a",
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatSubnetOscAssociateResourceNameErr: fmt.Errorf("cluster-api-subnet-public-uid subnet does not exist in natService"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope := Setup(t, nstc.name, nstc.spec)
			err := checkNatSubnetOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, nstc.expCheckNatSubnetOscAssociateResourceNameErr, err, "checkNatSubnetOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, nstc.expCheckNatSubnetOscAssociateResourceNameErr)
			}
		})
	}
}

// TestReconcileNatServiceCreate has several tests to cover the code of the function reconcileNatService
func TestReconcileNatServiceCreate(t *testing.T) {
	natServiceTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expPublicIpFound          bool
		expSubnetFound            bool
		expTagFound               bool
		expGetNatServiceErr       error
		expCreateNatServiceFound  bool
		expCreateNatServiceErr    error
		expReadTagErr             error
		expReconcileNatServiceErr error
	}{
		{
			name:                      "create natService (first time reconcile loop)",
			spec:                      defaultNatServiceInitialize,
			expTagFound:               false,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expGetNatServiceErr:       nil,
			expCreateNatServiceFound:  true,
			expCreateNatServiceErr:    nil,
			expReadTagErr:             nil,
			expReconcileNatServiceErr: nil,
		},
		{
			name:                      "failed to create natService",
			spec:                      defaultNatServiceInitialize,
			expTagFound:               false,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expGetNatServiceErr:       nil,
			expCreateNatServiceFound:  false,
			expCreateNatServiceErr:    fmt.Errorf("CreateNatService generic error"),
			expReconcileNatServiceErr: fmt.Errorf("CreateNatService generic error Can not create natService for Osccluster test-system/test-osc"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface, mockOscTagInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			natServiceRef.ResourceMap[natServiceName] = natServiceId
			tag := osc.Tag{
				ResourceId: &natServiceId,
			}
			if nstc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
					Return(&tag, nstc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
					Return(nil, nstc.expReadTagErr)
			}
			publicIpName := nstc.spec.Network.NatService.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if nstc.expPublicIpFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			subnetName := nstc.spec.Network.NatService.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if nstc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}
			clusterName := nstc.spec.Network.ClusterName + "-uid"
			natService := osc.CreateNatServiceResponse{
				NatService: &osc.NatService{
					NatServiceId: &natServiceId,
				},
			}

			if nstc.expCreateNatServiceFound {
				mockOscNatServiceInterface.
					EXPECT().
					CreateNatService(gomock.Eq(publicIpId), gomock.Eq(subnetId), gomock.Eq(natServiceName), gomock.Eq(clusterName)).
					Return(natService.NatService, nstc.expCreateNatServiceErr)
			} else {
				mockOscNatServiceInterface.
					EXPECT().
					CreateNatService(gomock.Eq(publicIpId), gomock.Eq(subnetId), gomock.Eq(natServiceName), gomock.Eq(clusterName)).
					Return(nil, nstc.expCreateNatServiceErr)
			}
			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileNatServiceErr.Error(), err.Error(), "reconcileNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileNatServiceErr)
			}
			t.Logf("Find reconcileNatService %v\n", reconcileNatService)
		})
	}
}

// TestReconcileNatServiceGet has several tests to cover the code of the function reconcileNatService
func TestReconcileNatServiceGet(t *testing.T) {
	natServiceTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expNatServiceFound        bool
		expPublicIpFound          bool
		expSubnetFound            bool
		expTagFound               bool
		expGetNatServiceErr       error
		expReadTagErr             error
		expReconcileNatServiceErr error
	}{
		{
			name:                      "check natService exist (second time reconcile loop)",
			spec:                      defaultNatServiceReconcile,
			expNatServiceFound:        true,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expTagFound:               false,
			expGetNatServiceErr:       nil,
			expReadTagErr:             nil,
			expReconcileNatServiceErr: nil,
		},
		{
			name:                      "failed to get natService",
			spec:                      defaultNatServiceReconcile,
			expNatServiceFound:        false,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expTagFound:               false,
			expGetNatServiceErr:       fmt.Errorf("GetSubnet generic error"),
			expReadTagErr:             nil,
			expReconcileNatServiceErr: fmt.Errorf("GetSubnet generic error"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface, mockOscTagInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if nstc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}
			tag := osc.Tag{
				ResourceId: &natServiceId,
			}
			if nstc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
					Return(&tag, nstc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
					Return(nil, nstc.expReadTagErr)
			}
			publicIpName := nstc.spec.Network.NatService.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if nstc.expPublicIpFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			subnetName := nstc.spec.Network.NatService.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if nstc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			natService := osc.CreateNatServiceResponse{
				NatService: &osc.NatService{
					NatServiceId: &natServiceId,
				},
			}
			readNatServices := osc.ReadNatServicesResponse{
				NatServices: &[]osc.NatService{
					*natService.NatService,
				},
			}
			readNatService := *readNatServices.NatServices

			if nstc.expNatServiceFound {
				mockOscNatServiceInterface.
					EXPECT().
					GetNatService(gomock.Eq(natServiceId)).
					Return(&readNatService[0], nstc.expGetNatServiceErr)
			} else {
				mockOscNatServiceInterface.
					EXPECT().
					GetNatService(gomock.Eq(natServiceId)).
					Return(nil, nstc.expGetNatServiceErr)
			}
			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileNatServiceErr, err, "reconcileNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileNatServiceErr)
			}
			t.Logf("Find reconcileNatService %v\n", reconcileNatService)
		})
	}
}

// TestReconcileNatServiceResourceId has several tests to cover the code of the function reconcileNatService
func TestReconcileNatServiceResourceId(t *testing.T) {
	natServiceTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expPublicIpFound          bool
		expSubnetFound            bool
		expTagFound               bool
		expReadTagErr             error
		expReconcileNatServiceErr error
	}{
		{
			name:                      "PublicIp does not exist",
			spec:                      defaultNatServiceInitialize,
			expPublicIpFound:          false,
			expSubnetFound:            true,
			expTagFound:               false,
			expReadTagErr:             nil,
			expReconcileNatServiceErr: fmt.Errorf("test-publicip-uid does not exist"),
		},
		{
			name:                      "Subnet does not exist",
			spec:                      defaultNatServiceInitialize,
			expPublicIpFound:          true,
			expSubnetFound:            false,
			expTagFound:               false,
			expReadTagErr:             nil,
			expReconcileNatServiceErr: fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                      "Failed to get tag",
			spec:                      defaultNatServiceReconcile,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expTagFound:               false,
			expReadTagErr:             fmt.Errorf("ReadTag generic error"),
			expReconcileNatServiceErr: fmt.Errorf("ReadTag generic error Can not get tag for OscCluster test-system/test-osc"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface, mockOscTagInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			publicIpName := nstc.spec.Network.NatService.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			tag := osc.Tag{
				ResourceId: &publicIpId,
			}
			if nstc.expPublicIpFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			subnetName := nstc.spec.Network.NatService.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if nstc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}
			natServiceName := nstc.spec.Network.NatService.Name + "-uid"

			if nstc.expPublicIpFound && nstc.expSubnetFound {
				if nstc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
						Return(&tag, nstc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(natServiceName)).
						Return(nil, nstc.expReadTagErr)
				}
			}
			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileNatServiceErr.Error(), err.Error(), "reconcileNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileNatServiceErr)
			}
			t.Logf("Find reconcileNatService %v\n", reconcileNatService)
		})
	}
}

// TestReconcileDeleteNatServiceDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteNatService
func TestReconcileDeleteNatServiceDeleteWithoutSpec(t *testing.T) {
	natServiceTestCases := []struct {
		name                            string
		spec                            infrastructurev1beta1.OscClusterSpec
		expGetNatServiceErr             error
		expDeleteNatServiceErr          error
		expReconcileDeleteNatServiceErr error
	}{
		{
			name:                            "delete natService without spec (with default values)",
			expGetNatServiceErr:             nil,
			expDeleteNatServiceErr:          nil,
			expReconcileDeleteNatServiceErr: nil,
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface, _ := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := "cluster-api-natservice-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			natServiceRef.ResourceMap[natServiceName] = natServiceId

			clusterScope.OscCluster.Spec.Network.NatService.ResourceId = natServiceId

			publicIpName := "cluster-api-publicip-nat-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			subnetName := "cluster-api-subnet-nat-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			natService := osc.CreateNatServiceResponse{
				NatService: &osc.NatService{
					NatServiceId: &natServiceId,
				},
			}
			readNatServices := osc.ReadNatServicesResponse{
				NatServices: &[]osc.NatService{
					*natService.NatService,
				},
			}
			readNatService := *readNatServices.NatServices

			mockOscNatServiceInterface.
				EXPECT().
				GetNatService(gomock.Eq(natServiceId)).
				Return(&readNatService[0], nstc.expGetNatServiceErr)

			mockOscNatServiceInterface.
				EXPECT().
				DeleteNatService(gomock.Eq(natServiceId)).
				Return(nstc.expDeleteNatServiceErr)

			reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope, mockOscNatServiceInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileDeleteNatServiceErr.Error(), err.Error(), "reconcileDeleteNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileDeleteNatServiceErr)
			}
			t.Logf("Find reconcileDeleteNatService %v\n", reconcileDeleteNatService)
		})
	}
}

// TestReconcileDeleteNatServiceDelete has several tests to cover the code of the function reconcileDeleteNatService
func TestReconcileDeleteNatServiceDelete(t *testing.T) {
	natServiceTestCases := []struct {
		name                            string
		spec                            infrastructurev1beta1.OscClusterSpec
		expSubnetFound                  bool
		expPublicIpFound                bool
		expNatServiceFound              bool
		expGetNatServiceErr             error
		expDeleteNatServiceErr          error
		expReconcileDeleteNatServiceErr error
	}{
		{
			name:                            "delete natService (first time reconcile loop)",
			spec:                            defaultNatServiceReconcile,
			expSubnetFound:                  true,
			expPublicIpFound:                true,
			expNatServiceFound:              true,
			expGetNatServiceErr:             nil,
			expDeleteNatServiceErr:          nil,
			expReconcileDeleteNatServiceErr: nil,
		},
		{
			name:                            "failed to delete natService",
			spec:                            defaultNatServiceReconcile,
			expSubnetFound:                  true,
			expPublicIpFound:                true,
			expNatServiceFound:              true,
			expGetNatServiceErr:             nil,
			expDeleteNatServiceErr:          fmt.Errorf("DeleteNatService generic error"),
			expReconcileDeleteNatServiceErr: fmt.Errorf("DeleteNatService generic error cannot delete natService for Osccluster test-system/test-osc"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface, _ := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if nstc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			natService := osc.CreateNatServiceResponse{
				NatService: &osc.NatService{
					NatServiceId: &natServiceId,
				},
			}
			readNatServices := osc.ReadNatServicesResponse{
				NatServices: &[]osc.NatService{
					*natService.NatService,
				},
			}
			readNatService := *readNatServices.NatServices
			if nstc.expNatServiceFound {
				mockOscNatServiceInterface.
					EXPECT().
					GetNatService(gomock.Eq(natServiceId)).
					Return(&readNatService[0], nstc.expGetNatServiceErr)
			} else {
				mockOscNatServiceInterface.
					EXPECT().
					GetNatService(gomock.Eq(natServiceId)).
					Return(nil, nstc.expGetNatServiceErr)
			}
			mockOscNatServiceInterface.
				EXPECT().
				DeleteNatService(gomock.Eq(natServiceId)).
				Return(nstc.expDeleteNatServiceErr)
			reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope, mockOscNatServiceInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileDeleteNatServiceErr.Error(), err.Error(), "reconcileDeleteNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileDeleteNatServiceErr)
			}
			t.Logf("Find reconcileDeleteNatService %v\n", reconcileDeleteNatService)
		})
	}
}
