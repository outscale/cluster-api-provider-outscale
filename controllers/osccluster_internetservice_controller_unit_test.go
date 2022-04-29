package controllers

import (
	"context"
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
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
)

// SetupWithInternetServiceMock set internetServiceMock with clusterScope and oscCluster
func SetupWithInternetServiceMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscInternetServiceInterface *mock_net.MockOscInternetServiceInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscInternetServiceInterface = mock_net.NewMockOscInternetServiceInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscInternetServiceInterface
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
			expGetInternetServiceResourceIdErr: fmt.Errorf("test-internetservice-uid is not exist"),
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
			if err != nil {
				assert.Equal(t, istc.expGetInternetServiceResourceIdErr.Error(), err.Error(), "GetInternetServiceResourceId() should return the same error")
			} else {
				assert.Nil(t, istc.expGetInternetServiceResourceIdErr)
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
			expCheckInternetServiceFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope := Setup(t, istc.name, istc.spec)
			internetServiceName, err := checkInternetServiceFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, istc.expCheckInternetServiceFormatParametersErr, err, "CheckInternetServiceFormatParameters() should return the same error")
			} else {
				assert.Nil(t, istc.expCheckInternetServiceFormatParametersErr)
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
		expInternetServiceFound        bool
		expCreateInternetServiceFound  bool
		expCreateInternetServiceErr    error
		expLinkInternetServiceErr      error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "create internet service (first time reconcile loop)",
			spec:                           defaultInternetServiceInitialize,
			expNetFound:                    true,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expCreateInternetServiceErr:    nil,
			expLinkInternetServiceErr:      nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:                           "failed to link internet service with net",
			spec:                           defaultInternetServiceInitialize,
			expNetFound:                    true,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expCreateInternetServiceErr:    nil,
			expLinkInternetServiceErr:      fmt.Errorf("LinkInternetService generic error"),
			expReconcileInternetServiceErr: fmt.Errorf("LinkInternetService generic error Can not link internetService with net for Osccluster test-system/test-osc"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
			if istc.expCreateInternetServiceFound {
				mockOscInternetServiceInterface.
					EXPECT().
					CreateInternetService(gomock.Eq(internetServiceName)).
					Return(internetService.InternetService, istc.expCreateInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					CreateInternetService(gomock.Eq(internetServiceName)).
					Return(nil, istc.expCreateInternetServiceErr)

			}
			mockOscInternetServiceInterface.
				EXPECT().
				LinkInternetService(gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expLinkInternetServiceErr)
			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileInternetServiceErr.Error(), err.Error(), "ReconcileInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileInternetServiceErr)
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
		expCreateInternetServiceErr    error
		expDescribeInternetServiceErr  error
		expLinkInternetServiceErr      error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "user delete internet service without cluster-api",
			spec:                           defaultInternetServiceReconcile,
			expCreateInternetServiceErr:    nil,
			expDescribeInternetServiceErr:  nil,
			expLinkInternetServiceErr:      nil,
			expReconcileInternetServiceErr: nil,
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
			mockOscInternetServiceInterface.
				EXPECT().
				GetInternetService(gomock.Eq(internetServiceId)).
				Return(nil, istc.expDescribeInternetServiceErr)
			mockOscInternetServiceInterface.
				EXPECT().
				CreateInternetService(gomock.Eq(internetServiceName)).
				Return(internetService.InternetService, istc.expCreateInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				LinkInternetService(gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expLinkInternetServiceErr)

			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileInternetServiceErr, err, "ReconcileInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileInternetServiceErr)
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
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
			mockOscInternetServiceInterface.
				EXPECT().
				GetInternetService(gomock.Eq(internetServiceId)).
				Return(&readInternetService[0], istc.expDescribeInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				UnlinkInternetService(gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expUnlinkInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				DeleteInternetService(gomock.Eq(internetServiceId)).
				Return(istc.expDeleteInternetServiceErr)

			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileDeleteInternetServiceErr, err, "ReconcileDelteInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileDeleteInternetServiceErr)
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
		expNetFound                    bool
		expInternetServiceFound        bool
		expCreateInternetServiceFound  bool
		expDescribeInternetServiceErr  error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "check internet service exist (second time reconcile loop)",
			spec:                           defaultInternetServiceReconcile,
			expNetFound:                    true,
			expInternetServiceFound:        true,
			expCreateInternetServiceFound:  false,
			expDescribeInternetServiceErr:  nil,
			expReconcileInternetServiceErr: nil,
		},
		{
			name:                           "failed to get internet service",
			spec:                           defaultInternetServiceReconcile,
			expNetFound:                    true,
			expInternetServiceFound:        false,
			expCreateInternetServiceFound:  true,
			expDescribeInternetServiceErr:  fmt.Errorf("GetSubnet generic error"),
			expReconcileInternetServiceErr: fmt.Errorf("GetSubnet generic error"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
			}

			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileInternetServiceErr, err, "ReconcileInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileInternetServiceErr)
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
		expCreateInternetServiceErr    error
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "failed to create internet service",
			spec:                           defaultInternetServiceInitialize,
			expCreateInternetServiceErr:    fmt.Errorf("CreateInternetService generic error"),
			expReconcileInternetServiceErr: fmt.Errorf("CreateInternetService generic error Can not create internetservice for Osccluster test-system/test-osc"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			internetServiceName := istc.spec.Network.InternetService.Name + "-uid"
			netName := istc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			mockOscInternetServiceInterface.
				EXPECT().
				CreateInternetService(gomock.Eq(internetServiceName)).
				Return(nil, istc.expCreateInternetServiceErr)

			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileInternetServiceErr.Error(), err.Error(), "ReconcileInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileInternetServiceErr)
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
		expReconcileInternetServiceErr error
	}{
		{
			name:                           "net does not exist",
			spec:                           defaultInternetServiceInitialize,
			expReconcileInternetServiceErr: fmt.Errorf("test-net-uid is not exist"),
		},
	}
	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileInternetServiceErr, err, "ReconcileInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileInternetServiceErr)
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
			name:                                 "failed to get interntservice",
			spec:                                 defaultInternetServiceReconcile,
			expNetFound:                          true,
			expInternetServiceFound:              false,
			expDescribeInternetServiceErr:        fmt.Errorf("GetInternetService generic error"),
			expReconcileDeleteInternetServiceErr: fmt.Errorf("GetInternetService generic error"),
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
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
			}

			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileDeleteInternetServiceErr, err, "ReconcileDeleteInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileDeleteInternetServiceErr)

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
			expUnlinkInternetServiceErr:          fmt.Errorf("UnlinkInternetService generic error"),
			expReconcileDeleteInternetServiceErr: fmt.Errorf("UnlinkInternetService generic error Can not unlink internetService and net for Osccluster test-system/test-osc"),
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
				GetInternetService(gomock.Eq(internetServiceId)).
				Return(&readInternetService[0], istc.expDescribeInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				UnlinkInternetService(gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expUnlinkInternetServiceErr)

			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileDeleteInternetServiceErr.Error(), err.Error(), "ReconcileDelteInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileDeleteInternetServiceErr)
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
			expDeleteInternetServiceErr:          fmt.Errorf("DeleteInternetService generic error"),
			expDescribeInternetServiceErr:        nil,
			expUnlinkInternetServiceErr:          nil,
			expReconcileDeleteInternetServiceErr: fmt.Errorf("DeleteInternetService generic error Can not delete internetService for Osccluster test-system/test-osc"),
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
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
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(&readInternetService[0], istc.expDescribeInternetServiceErr)
			} else {
				mockOscInternetServiceInterface.
					EXPECT().
					GetInternetService(gomock.Eq(internetServiceId)).
					Return(nil, istc.expDescribeInternetServiceErr)
			}

			mockOscInternetServiceInterface.
				EXPECT().
				UnlinkInternetService(gomock.Eq(internetServiceId), gomock.Eq(netId)).
				Return(istc.expUnlinkInternetServiceErr)

			mockOscInternetServiceInterface.
				EXPECT().
				DeleteInternetService(gomock.Eq(internetServiceId)).
				Return(istc.expDeleteInternetServiceErr)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if istc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileDeleteInternetServiceErr.Error(), err.Error(), "ReconcileDelteInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileDeleteInternetServiceErr)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)

		})
	}
}

// TestReconcileDeleteInternetServiceResourceId has several tests to cover the code of the function reconcileDeleteInternetService
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
			expReconcileDeleteInternetServiceErr: fmt.Errorf("cluster-api-net-uid is not exist"),
		},
		{
			name:                                 "net does not exist",
			spec:                                 defaultInternetServiceInitialize,
			expReconcileDeleteInternetServiceErr: fmt.Errorf("test-net-uid is not exist"),
		},
	}

	for _, istc := range internetServiceTestCases {
		t.Run(istc.name, func(t *testing.T) {
			t.Logf("Validate to %s", istc.name)
			clusterScope, ctx, mockOscInternetServiceInterface := SetupWithInternetServiceMock(t, istc.name, istc.spec)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, mockOscInternetServiceInterface)
			if err != nil {
				assert.Equal(t, istc.expReconcileDeleteInternetServiceErr, err, "ReconcileDeleteInternetService() should return the same error")
			} else {
				assert.Nil(t, istc.expReconcileDeleteInternetServiceErr)
			}
			t.Logf("Find reconcileDeleteInternetService %v\n", reconcileDeleteInternetService)

		})
	}
}
