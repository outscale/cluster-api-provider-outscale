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

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

var (
	defaultNatServiceInitialize = infrastructurev1beta1.OscClusterSpec{
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
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IPSubnetRange: "10.0.0.0/24",
				},
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
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
				IPRange:    "10.0.0.0/16",
				ResourceID: "vpc-test-net-uid",
			},
			NatService: infrastructurev1beta1.OscNatService{Name: "test-natservice",
				PublicIPName: "test-publicip",
				SubnetName:   "test-subnet-test",
				ResourceID:   "nat-test-natservice-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IPSubnetRange: "10.0.0.0/24",
					ResourceID:    "subnet-test-subnet-uid",
				},
			},
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name:       "test-publicip",
					ResourceID: "eipalloc-test-publicip-uid",
				},
			},
		},
	}
)

// SetupWithNatServiceMock set natServiceMock with clusterScope and osccluster
func SetupWithNatServiceMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscNatServiceInterface *mock_net.MockOscNatServiceInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscNatServiceInterface = mock_net.NewMockOscNatServiceInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscNatServiceInterface
}

// TestGetNatResourceID has several tests to cover the code of the function getNatResourceID
func TestGetNatResourceID(t *testing.T) {
	natServiceTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		expNatServiceFound     bool
		expGetNatResourceIDErr error
	}{
		{
			name:                   "get natServiceId",
			spec:                   defaultNatServiceInitialize,
			expNatServiceFound:     true,
			expGetNatResourceIDErr: nil,
		},
		{
			name:                   "can not get natServiceId",
			spec:                   defaultNatServiceInitialize,
			expNatServiceFound:     false,
			expGetNatResourceIDErr: fmt.Errorf("test-natservice-uid does not exist"),
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

			natResourceID, err := getNatResourceID(natServiceName, clusterScope)
			if err != nil {
				assert.Equal(t, nstc.expGetNatResourceIDErr, err, "GetNatResourceId() should return the same error")
			} else {
				assert.Nil(t, nstc.expGetNatResourceIDErr)

			}
			t.Logf("find natResourceID %s", natResourceID)
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
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice@test",
						PublicIPName: "test-publicip",
						SubnetName:   "test-subnet",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IPSubnetRange: "10.0.0.0/24",
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("invalid Tag Name"),
		},
		{
			name: "check Bad name publicIp",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIPName: "test-publicip@test",
						SubnetName:   "test-subnet",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IPSubnetRange: "10.0.0.0/24",
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("invalid Tag Name"),
		},
		{
			name: "Check BadName subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice",
						PublicIPName: "test-publicip",
						SubnetName:   "test-subnet@test",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IPSubnetRange: "10.0.0.0/24",
							ResourceID:    "subnet-test-subnet-uid",
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatFormatParametersErr: fmt.Errorf("invalid Tag Name"),
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
			expCheckNatSubnetOscAssociateResourceNameErr: fmt.Errorf("cluster-api-subnet-nat-uid subnet does not exist in natService"),
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
						IPRange: "10.0.0.0/16",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "test-natservice@test",
						PublicIPName: "test-publicip",
						SubnetName:   "test-subnet-test",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IPSubnetRange: "10.0.0.0/24",
							ResourceID:    "subnet-test-subnet-uid",
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "test-publicip",
						},
					},
				},
			},
			expCheckNatSubnetOscAssociateResourceNameErr: fmt.Errorf("test-subnet-test-uid subnet does not exist in natService"),
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
		expGetNatServiceErr       error
		expCreateNatServiceFound  bool
		expCreateNatServiceErr    error
		expReconcileNatServiceErr error
	}{
		{
			name:                      "create natService (first time reconcile loop)",
			spec:                      defaultNatServiceInitialize,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expGetNatServiceErr:       nil,
			expCreateNatServiceFound:  true,
			expCreateNatServiceErr:    nil,
			expReconcileNatServiceErr: nil,
		},
		{
			name:                      "failed to create natService",
			spec:                      defaultNatServiceInitialize,
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
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)

			publicIpName := nstc.spec.Network.NatService.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
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

			if nstc.expCreateNatServiceFound {
				mockOscNatServiceInterface.
					EXPECT().
					CreateNatService(gomock.Eq(publicIpId), gomock.Eq(subnetId), gomock.Eq(natServiceName)).
					Return(natService.NatService, nstc.expCreateNatServiceErr)
			} else {
				mockOscNatServiceInterface.
					EXPECT().
					CreateNatService(gomock.Eq(publicIpId), gomock.Eq(subnetId), gomock.Eq(natServiceName)).
					Return(nil, nstc.expCreateNatServiceErr)
			}
			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface)
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
		expGetNatServiceErr       error
		expReconcileNatServiceErr error
	}{
		{
			name:                      "check natService exist (second time reconcile loop)",
			spec:                      defaultNatServiceReconcile,
			expNatServiceFound:        true,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expGetNatServiceErr:       nil,
			expReconcileNatServiceErr: nil,
		},
		{
			name:                      "failed to get natService",
			spec:                      defaultNatServiceReconcile,
			expNatServiceFound:        false,
			expPublicIpFound:          true,
			expSubnetFound:            true,
			expGetNatServiceErr:       fmt.Errorf("GetSubnet generic error"),
			expReconcileNatServiceErr: fmt.Errorf("GetSubnet generic error"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := nstc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if nstc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			publicIpName := nstc.spec.Network.NatService.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
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
			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileNatServiceErr, err, "reconcileNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileNatServiceErr)
			}
			t.Logf("Find reconcileNatService %v\n", reconcileNatService)
		})
	}
}

// TestReconcileNatServiceResourceID has several tests to cover the code of the function reconcileNatService
func TestReconcileNatServiceResourceID(t *testing.T) {
	natServiceTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expPublicIpFound          bool
		expSubnetFound            bool
		expReconcileNatServiceErr error
	}{
		{
			name:                      "PublicIp does not exist",
			spec:                      defaultNatServiceInitialize,
			expPublicIpFound:          false,
			expSubnetFound:            true,
			expReconcileNatServiceErr: fmt.Errorf("test-publicip-uid does not exist"),
		},
		{
			name:                      "Subnet does not exist",
			spec:                      defaultNatServiceInitialize,
			expPublicIpFound:          true,
			expSubnetFound:            false,
			expReconcileNatServiceErr: fmt.Errorf("test-subnet-uid does not exist"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			publicIpName := nstc.spec.Network.NatService.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
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

			reconcileNatService, err := reconcileNatService(ctx, clusterScope, mockOscNatServiceInterface)
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
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

			natServiceName := "cluster-api-natservice-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			natServiceRef.ResourceMap[natServiceName] = natServiceId

			clusterScope.OscCluster.Spec.Network.NatService.ResourceID = natServiceId

			publicIpName := "cluster-api-publicip-nat-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
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
			expReconcileDeleteNatServiceErr: fmt.Errorf("DeleteNatService generic error Can not delete natService for Osccluster test-system/test-osc"),
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

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

// TestReconcileDeleteNatServiceGet has several tests to cover the code of the function reconcileDeleteNatService
func TestReconcileDeleteNatServiceGet(t *testing.T) {
	natServiceTestCases := []struct {
		name                            string
		spec                            infrastructurev1beta1.OscClusterSpec
		expSubnetFound                  bool
		expPublicIpFound                bool
		expNatServiceFound              bool
		expGetNatServiceErr             error
		expReconcileDeleteNatServiceErr error
	}{
		{
			name:                            "failed to get natService",
			spec:                            defaultNatServiceReconcile,
			expSubnetFound:                  true,
			expPublicIpFound:                true,
			expNatServiceFound:              false,
			expGetNatServiceErr:             fmt.Errorf("GetNatService generic error"),
			expReconcileDeleteNatServiceErr: fmt.Errorf("GetNatService generic error"),
		},
		{
			name:                            "Remove finalizer (user delete natService without cluster-api)",
			spec:                            defaultNatServiceReconcile,
			expSubnetFound:                  true,
			expPublicIpFound:                true,
			expNatServiceFound:              false,
			expGetNatServiceErr:             nil,
			expReconcileDeleteNatServiceErr: nil,
		},
	}
	for _, nstc := range natServiceTestCases {
		t.Run(nstc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNatServiceInterface := SetupWithNatServiceMock(t, nstc.name, nstc.spec)

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
			reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope, mockOscNatServiceInterface)
			if err != nil {
				assert.Equal(t, nstc.expReconcileDeleteNatServiceErr, err, "reconcileDeleteNatService() should return the same error")
			} else {
				assert.Nil(t, nstc.expReconcileDeleteNatServiceErr)

			}
			t.Logf("Find reconcileDeleteNatService %v\n", reconcileDeleteNatService)
		})
	}
}
