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
	defaultInternetServiceInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			InternetService: infrastructurev1beta1.OscInternetService{
				Name: "test-internetservice",
			},
		},
	}

	defaultInternetServiceInitializeWithSkipReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			InternetService: infrastructurev1beta1.OscInternetService{
				Name:          "test-internetservice",
				SkipReconcile: true,
			},
		},
	}

	defaultInternetServiceReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			InternetService: infrastructurev1beta1.OscInternetService{
				Name:       "test-internetservice",
				ResourceId: "igw-test-internetservice-uid",
			},
		},
	}

	defaultInternetServiceReconcileWithSkipReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			InternetService: infrastructurev1beta1.OscInternetService{
				Name:          "test-internetservice",
				ResourceId:    "igw-test-internetservice-uid",
				SkipReconcile: true,
			},
		},
	}
)

// SetupWithInternetServiceMock set internetServiceMock with clusterScope and oscCluster
func SetupWithInternetServiceMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscInternetServiceInterface *mock_net.MockOscInternetServiceInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscInternetServiceInterface = mock_net.NewMockOscInternetServiceInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface
}

// TestGetInternetServiceResourceId has several tests to cover the code of the function getInternetServiceResourceId
func TestGetInternetServiceResourceId(t *testing.T) {
	internetServiceTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expInternetServiceFound            bool
		expGetInternetServiceResourceIdErr error
	}{
		{
			name:                               "get InternetServiceId",
			spec:                               defaultInternetServiceInitialize,
			expInternetServiceFound:            true,
			expGetInternetServiceResourceIdErr: nil,
		},
		{
			name:                               "can not get InternetServiceId",
			spec:                               defaultInternetServiceInitialize,
			expInternetServiceFound:            false,
			expGetInternetServiceResourceIdErr: errors.New("test-internetservice-uid does not exist"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope := Setup(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if istc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}
			internetServiceResourceId, err := getInternetServiceResourceId(internetServiceName, clusterScope)
			if istc.expGetInternetServiceResourceIdErr != nil {
				require.EqualError(t, err, istc.expGetInternetServiceResourceIdErr.Error(), "GetInternetServiceResourceId() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find internetServiceResourceId %s", internetServiceResourceId)
		})
	}
}

// TestCheckInternetServiceFormatParameters has several tests to cover the code of the func checkInternetServiceFormatParameters
func TestCheckInternetServiceFormatParameters(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                       string
		spec                                       infrastructurev1beta1.OscClusterSpec
		expCheckInternetServiceFormatParametersErr error
	}{
		{
			name: "check work without net and internetservice spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckInternetServiceFormatParametersErr: nil,
		},
		{
			name: "check internetService format",
			spec: defaultInternetServiceInitialize,
			expCheckInternetServiceFormatParametersErr: nil,
		},
		{
			name: "check Bad Name  internetService",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "test-internetservice@test",
					},
				},
			},
			expCheckInternetServiceFormatParametersErr: errors.New("Invalid Tag Name"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope := Setup(t, istc.name, istc.spec)
			internetServiceName, err := checkInternetServiceFormatParameters(clusterScope)
			if istc.expCheckInternetServiceFormatParametersErr != nil {
				require.EqualError(t, err, istc.expCheckInternetServiceFormatParametersErr.Error(), "CheckInternetServiceFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find internetServiceName %s\n", internetServiceName)
		})
	}
}

// TestReconcileInternetServiceLink has several tests to cover the code of the function reconcileInternetService
func TestReconcileInternetServiceLink(t *testing.T) {
	internetServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expNetFound                    bool
		expTagFound                    bool
		expInternetServiceFound        bool
		expCreateInternetServiceFound  bool
		expReadTagErr                  error
		expCreateInternetServiceErr    error
		expLinkInternetServiceErr      error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "create internet service (first time reconcile loop)",
			spec:                           defaultInternetServiceInitialize,
			expNetFound:                    true,
			expTagFound:                    false,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expCreateInternetServiceErr:    nil,
			expLinkInternetServiceErr:      nil,
			expReadTagErr:                  nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:                           "create internet service with skip reconcile (first time reconcile loop)",
			spec:                           defaultInternetServiceInitializeWithSkipReconcile,
			expNetFound:                    true,
			expTagFound:                    false,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expCreateInternetServiceErr:    nil,
			expLinkInternetServiceErr:      nil,
			expReadTagErr:                  nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:                           "failed to link internet service with net",
			spec:                           defaultInternetServiceInitialize,
			expNetFound:                    true,
			expTagFound:                    false,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expCreateInternetServiceErr:    nil,
			expReadTagErr:                  nil,
			expLinkInternetServiceErr:      errors.New("LinkInternetService generic error"),
			expReconcileInternetServiceErr: errors.New("cannot link internetService: LinkInternetService generic error"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if istc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			tag := osc.Tag{
				ResourceId: &internetServiceId,
			}
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			if istc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
					Return(&tag, istc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
					Return(nil, istc.expReadTagErr)
			}
			if istc.expCreateInternetServiceFound {
				mockOscInternetServiceInterface.
					EXPECT().
					CreateInternetService(gomock.Any(), gomock.Eq(internetServiceName)).
					Return(internetService.InternetService, istc.expCreateInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					CreateInternetService(gomock.Any(), gomock.Eq(internetServiceName)).
					Return(nil, istc.expCreateInternetServiceErr)
			}
			mockOscInternetServiceInterface.
				EXPECT().
				LinkInternetService(gomock.Any(), gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expLinkInternetServiceErr)
			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface, mockOscTagInterface)
			if istc.expReconcileInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileInternetServiceErr.Error(), "ReconcileInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileInternetService %v\n", reconcileInternetService)
		})
	}
}

// TestReconcileInternetServiceDelete has several tests to cover the code of the function reconcileInternetService
func TestReconcileInternetServiceDelete(t *testing.T) {
	internetServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expSkipReconcile               bool
		expTagFound                    bool
		expCreateInternetServiceErr    error
		expDescribeInternetServiceErr  error
		expLinkInternetServiceErr      error
		expReadTagErr                  error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "user delete internet service without cluster-api",
			spec:                           defaultInternetServiceReconcile,
			expTagFound:                    false,
			expReadTagErr:                  nil,
			expCreateInternetServiceErr:    nil,
			expDescribeInternetServiceErr:  nil,
			expLinkInternetServiceErr:      nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:             "user delete internet service with skip reconcile without cluster-api",
			spec:             defaultInternetServiceReconcileWithSkipReconcile,
			expSkipReconcile: true,
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			tag := osc.Tag{
				ResourceId: &internetServiceId,
			}
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			if !istc.expSkipReconcile {
				if istc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(&tag, istc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(nil, istc.expReadTagErr)
				}
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
				mockOscInternetServiceInterface.
					EXPECT().
					CreateInternetService(gomock.Any(), gomock.Eq(internetServiceName)).
					Return(internetService.InternetService, istc.expCreateInternetServiceErr)

				mockOscInternetServiceInterface.
					EXPECT().
					LinkInternetService(gomock.Any(), gomock.Eq(internetServiceId), gomock.Eq(netId)).
					Return(istc.expLinkInternetServiceErr)
			}

			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface, mockOscTagInterface)
			if istc.expReconcileInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileInternetServiceErr.Error(), "ReconcileInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileInternetService %v\n", reconcileInternetService)
		})
	}
}

// TestReconcileDeleteInternetServiceDelete has several tests to cover the code of the function reconcileDeleteInternetService
func TestReconcileDeleteInternetServiceDeleteWithoutSpec(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expSkipReconcile                     bool
		expDeleteInternetServiceErr          error
		expDescribeInternetServiceErr        error
		expUnlinkInternetServiceErr          error
		expReconcileDeleteInternetServiceErr error
	}{
		{
			name:                                 "delete internet service without spec (with default values)",
			expDeleteInternetServiceErr:          nil,
			expDescribeInternetServiceErr:        nil,
			expUnlinkInternetServiceErr:          nil,
			expReconcileDeleteInternetServiceErr: nil,
		},
		{
			name:             "delete internet service without spec (with default values)",
			spec:             defaultInternetServiceReconcileWithSkipReconcile,
			expSkipReconcile: true,
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, _ := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := "cluster-api-internetservice-uid"
			internetServiceId := "igw-" + internetServiceName
			clusterScope.OscCluster.Spec.Network.InternetService.ResourceId = internetServiceId
			netName := "cluster-api-net-uid"
			netId := "vpc-" + netName
			clusterScope.OscCluster.Spec.Network.Net.ResourceId = netId
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			readInternetServices := osc.ReadInternetServicesResponse{
				InternetServices: &[]osc.InternetService{
					*internetService.InternetService,
				},
			}
			readInternetService := *readInternetServices.InternetServices
			if !istc.expSkipReconcile {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)

				mockOscInternetServiceInterface.
					EXPECT().
					UnlinkInternetService(gomock.Any(), gomock.Eq(internetServiceId), gomock.Eq(netId)).
					Return(istc.expUnlinkInternetServiceErr)

				mockOscInternetServiceInterface.
					EXPECT().
					DeleteInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(istc.expDeleteInternetServiceErr)
			}
			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if istc.expReconcileDeleteInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileDeleteInternetServiceErr.Error(), "ReconcileDelteInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)
		})
	}
}

// TestReconcileInternetServiceGet has several tests to cover the code of the function reconcileInternetService
func TestReconcileInternetServiceGet(t *testing.T) {
	internetServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expSkipReconcile               bool
		expNetFound                    bool
		expInternetServiceFound        bool
		expTagFound                    bool
		expCreateInternetServiceFound  bool
		expReadTagErr                  error
		expDescribeInternetServiceErr  error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "check internet service exist (second time reconcile loop)",
			spec:                           defaultInternetServiceReconcile,
			expNetFound:                    true,
			expInternetServiceFound:        true,
			expTagFound:                    true,
			expCreateInternetServiceFound:  false,
			expDescribeInternetServiceErr:  nil,
			expReadTagErr:                  nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:             "skip reconciliation loop",
			spec:             defaultInternetServiceReconcileWithSkipReconcile,
			expSkipReconcile: true,
		},
		{
			name:                           "failed to get internet service",
			spec:                           defaultInternetServiceReconcile,
			expNetFound:                    true,
			expTagFound:                    true,
			expInternetServiceFound:        true,
			expCreateInternetServiceFound:  true,
			expReadTagErr:                  nil,
			expDescribeInternetServiceErr:  errors.New("GetSubnet generic error"),
			expReconcileInternetServiceErr: errors.New("GetSubnet generic error"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			tag := osc.Tag{
				ResourceId: &internetServiceId,
			}
			if !istc.expSkipReconcile {
				if istc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(&tag, istc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(nil, istc.expReadTagErr)
				}

				if istc.expNetFound {
					netRef.ResourceMap[netName] = netId
				}
				internetService := osc.CreateInternetServiceResponse{
					InternetService: &osc.InternetService{
						InternetServiceId: &internetServiceId,
					},
				}
				readInternetServices := osc.ReadInternetServicesResponse{
					InternetServices: &[]osc.InternetService{
						*internetService.InternetService,
					},
				}
				readInternetService := *readInternetServices.InternetServices
				if istc.expInternetServiceFound {
					mockOscInternetServiceInterface.
						EXPECT().
						GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
						Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
				} else {
					mockOscInternetServiceInterface.
						EXPECT().
						GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
						Return(nil, istc.expDescribeInternetServiceErr)
				}
			}
			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface, mockOscTagInterface)
			if istc.expReconcileInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileInternetServiceErr.Error(), "ReconcileInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileInternetService %v\n", reconcileInternetService)
		})
	}
}

// TestReconcileInternetServiceCreate has several tests to cover the code of the function reconcileInternetService
func TestReconcileInternetServiceCreate(t *testing.T) {
	internetServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expTagFound                    bool
		expCreateInternetServiceErr    error
		expReadTagErr                  error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "failed to create internet service",
			spec:                           defaultInternetServiceInitialize,
			expTagFound:                    false,
			expCreateInternetServiceErr:    errors.New("CreateInternetService generic error"),
			expReadTagErr:                  nil,
			expReconcileInternetServiceErr: errors.New("cannot create internetservice: CreateInternetService generic error"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			tag := osc.Tag{
				ResourceId: &internetServiceId,
			}
			if istc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
					Return(&tag, istc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
					Return(nil, istc.expReadTagErr)
			}

			mockOscInternetServiceInterface.
				EXPECT().
				CreateInternetService(gomock.Any(), gomock.Eq(internetServiceName)).
				Return(nil, istc.expCreateInternetServiceErr)

			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface, mockOscTagInterface)
			if istc.expReconcileInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileInternetServiceErr.Error(), "ReconcileInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileInternetService %v\n", reconcileInternetService)
		})
	}
}

// TestReconcileInternetServiceResourceId has several tests to cover the code of the function reconcileInternetService
func TestReconcileInternetServiceResourceId(t *testing.T) {
	internetServiceTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expTagFound                    bool
		expNetFound                    bool
		expReadTagErr                  error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "net does not exist",
			spec:                           defaultInternetServiceInitialize,
			expReconcileInternetServiceErr: errors.New("test-net-uid does not exist"),
			expTagFound:                    true,
			expNetFound:                    false,
			expReadTagErr:                  nil,
		},
		{
			name:                           "Failed to get tag",
			spec:                           defaultInternetServiceInitialize,
			expTagFound:                    true,
			expNetFound:                    true,
			expReadTagErr:                  errors.New("ReadTag generic error"),
			expReconcileInternetServiceErr: errors.New("cannot get tag: ReadTag generic error"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, mockOscTagInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			netRef := clusterScope.GetNetRef()

			netRef.ResourceMap = make(map[string]string)
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			tag := osc.Tag{
				ResourceId: &internetServiceId,
			}
			if istc.expNetFound {
				netRef.ResourceMap[netName] = netId
				if istc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(&tag, istc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(internetServiceName)).
						Return(nil, istc.expReadTagErr)
				}
			}
			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface, mockOscTagInterface)
			if istc.expReconcileInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileInternetServiceErr.Error(), "ReconcileInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileInternetService %v\n", reconcileInternetService)
		})
	}
}

// TestReconcileDeleteInternetServiceGet has several tests to cover the code of the function reconcileDeleteInternetService
func TestReconcileDeleteInternetServiceGet(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expNetFound                          bool
		expInternetServiceFound              bool
		expDescribeInternetServiceErr        error
		expReconcileDeleteInternetServiceErr error
	}{
		{
			name:                                 "failed to get internetservice",
			spec:                                 defaultInternetServiceReconcile,
			expNetFound:                          true,
			expInternetServiceFound:              false,
			expDescribeInternetServiceErr:        errors.New("GetInternetService generic error"),
			expReconcileDeleteInternetServiceErr: errors.New("get internetservice: GetInternetService generic error"),
		},
		{
			name:                                 "remove finalizer (user delete internetService without cluster-api)",
			spec:                                 defaultInternetServiceReconcile,
			expNetFound:                          true,
			expInternetServiceFound:              false,
			expDescribeInternetServiceErr:        nil,
			expReconcileDeleteInternetServiceErr: nil,
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, _ := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if istc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			readInternetServices := osc.ReadInternetServicesResponse{
				InternetServices: &[]osc.InternetService{
					*internetService.InternetService,
				},
			}
			readInternetService := *readInternetServices.InternetServices
			if istc.expInternetServiceFound {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
			}

			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if istc.expReconcileDeleteInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileDeleteInternetServiceErr.Error(), "ReconcileDeleteInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)
		})
	}
}

// TestReconcileDeleteInternetServiceUnlink has several tests to cover the code of the function reconcileDeleteInternetService
func TestReconcileDeleteInternetServiceUnlink(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expDescribeInternetServiceErr        error
		expUnlinkInternetServiceErr          error
		expReconcileDeleteInternetServiceErr error
	}{
		{
			name:                                 "failed to unlink internet service",
			spec:                                 defaultInternetServiceReconcile,
			expDescribeInternetServiceErr:        nil,
			expUnlinkInternetServiceErr:          errors.New("UnlinkInternetService generic error"),
			expReconcileDeleteInternetServiceErr: errors.New("cannot unlink internetService and net: UnlinkInternetService generic error"),
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, _ := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			readInternetServices := osc.ReadInternetServicesResponse{
				InternetServices: &[]osc.InternetService{
					*internetService.InternetService,
				},
			}
			readInternetService := *readInternetServices.InternetServices
			mockOscInternetServiceInterface.
				EXPECT().
				GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
				Return(&readInternetService[0], istc.expDescribeInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				UnlinkInternetService(gomock.Any(), gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expUnlinkInternetServiceErr)

			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if istc.expReconcileDeleteInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileDeleteInternetServiceErr.Error(), "ReconcileDelteInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)
		})
	}
}

// TestReconcileDeleteInternetServiceDelete has several tests to cover the code of the function reconcileDeleteInternetService
func TestReconcileDeleteInternetServiceDelete(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expNetFound                          bool
		expInternetServiceFound              bool
		expDeleteInternetServiceErr          error
		expDescribeInternetServiceErr        error
		expUnlinkInternetServiceErr          error
		expReconcileDeleteInternetServiceErr error
	}{
		{
			name:                                 "delete internet service",
			spec:                                 defaultInternetServiceReconcile,
			expNetFound:                          true,
			expInternetServiceFound:              true,
			expDeleteInternetServiceErr:          nil,
			expDescribeInternetServiceErr:        nil,
			expUnlinkInternetServiceErr:          nil,
			expReconcileDeleteInternetServiceErr: nil,
		},
		{
			name:                                 "failed to delete internet service",
			spec:                                 defaultInternetServiceReconcile,
			expNetFound:                          true,
			expInternetServiceFound:              true,
			expDeleteInternetServiceErr:          errors.New("DeleteInternetService generic error"),
			expDescribeInternetServiceErr:        nil,
			expUnlinkInternetServiceErr:          nil,
			expReconcileDeleteInternetServiceErr: errors.New("cannot delete internetService: DeleteInternetService generic error"),
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface, _ := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			internetService := osc.CreateInternetServiceResponse{
				InternetService: &osc.InternetService{
					InternetServiceId: &internetServiceId,
				},
			}
			readInternetServices := osc.ReadInternetServicesResponse{
				InternetServices: &[]osc.InternetService{
					*internetService.InternetService,
				},
			}
			readInternetService := *readInternetServices.InternetServices
			if istc.expInternetServiceFound {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
			}

			mockOscInternetServiceInterface.
				EXPECT().
				UnlinkInternetService(gomock.Any(), gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expUnlinkInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				DeleteInternetService(gomock.Any(), gomock.Eq(internetServiceId)).
				Return(istc.expDeleteInternetServiceErr)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if istc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if istc.expReconcileDeleteInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileDeleteInternetServiceErr.Error(), "ReconcileDelteInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)
		})
	}
}

// TestReconcileDeleteInternetServiceResourceId tests that reconcileDeleteInternetService does nothing if no net is known.
func TestReconcileDeleteInternetServiceResourceId(t *testing.T) {
	internetServiceTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expReconcileDeleteInternetServiceErr error
	}{
		{
			name: "check net failed without net and internetservice spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
		},
		{
			name: "net does not exist",
			spec: defaultInternetServiceInitialize,
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			t.Logf("Validate to %s", istc.name)
			clusterScope, ctx, mockOscInternetServiceInterface, _ := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if istc.expReconcileDeleteInternetServiceErr != nil {
				require.EqualError(t, err, istc.expReconcileDeleteInternetServiceErr.Error(), "ReconcileDeleteInternetService() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)
		})
	}
}
