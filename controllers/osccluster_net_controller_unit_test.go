package controllers

import (
	"context"
	"fmt"

	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2/klogr"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var (
	defaultNetInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
		},
	}

	defaultNetReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
		},
	}
)

// Setup set osccluster and clusterScope
func Setup(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope) {
	t.Logf("Validate to %s", name)

	oscCluster := infrastructurev1beta1.OscCluster{
		Spec: spec,
		ObjectMeta: metav1.ObjectMeta{
			UID:       "uid",
			Name:      "test-osc",
			Namespace: "test-system",
		},
	}
	log := klogr.New()
	clusterScope = &scope.ClusterScope{
		Logger: log,
		Cluster: &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "uid",
				Name:      "test-osc",
				Namespace: "test-system",
			},
		},
		OscCluster: &oscCluster,
	}
	return clusterScope
}

// SetupWithNetMock set netMock with clusterScope and osccluster
func SetupWithNetMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscNetInterface *mock_net.MockOscNetInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscNetInterface = mock_net.NewMockOscNetInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscNetInterface
}

// TestGetNetResourceId has several tests to cover the code of the function getNetResourceId
func TestGetNetResourceId(t *testing.T) {
	netTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		expNetFound            bool
		expGetNetResourceIdErr error
	}{
		{
			name:                   "get NetId",
			spec:                   defaultNetInitialize,
			expNetFound:            true,
			expGetNetResourceIdErr: nil,
		},
		{
			name:                   "can not get netId",
			spec:                   defaultNetInitialize,
			expNetFound:            false,
			expGetNetResourceIdErr: fmt.Errorf("test-net-uid does not exist"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope := Setup(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if ntc.expNetFound {
				netRef := clusterScope.GetNetRef()
				netRef.ResourceMap = make(map[string]string)
				netRef.ResourceMap[netName] = netId
			}
			netResourceId, err := getNetResourceId(netName, clusterScope)
			if err != nil {
				assert.Equal(t, ntc.expGetNetResourceIdErr, err, "getNetResourceId() should return the same error")
			} else {
				assert.Nil(t, ntc.expGetNetResourceIdErr)
			}
			t.Logf("find netResourceId %s", netResourceId)
		})
	}
}

// TestCheckNetFormatParameters has several tests to cover the code of the func checkNetFormatParameters
func TestCheckNetFormatParameters(t *testing.T) {
	netTestCases := []struct {
		name                           string
		spec                           infrastructurev1beta1.OscClusterSpec
		expCheckNetFormatParametersErr error
	}{
		{
			name: "check work without net spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckNetFormatParametersErr: nil,
		},
		{
			name:                           "check Net Format",
			spec:                           defaultNetInitialize,
			expCheckNetFormatParametersErr: nil,
		},
		{
			name: "check Bad Ip Range Prefix Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/36",
					},
				},
			},
			expCheckNetFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "check Bad Ip Range IP Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.256/16",
					},
				},
			},
			expCheckNetFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "check Bad Name Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net@test",
						IpRange: "10.0.0.0/16",
					},
				},
			},
			expCheckNetFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope := Setup(t, ntc.name, ntc.spec)
			netName, err := checkNetFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, ntc.expCheckNetFormatParametersErr.Error(), err.Error(), "checkNetFormatParameters() should return the same error")
			} else {
				assert.Nil(t, ntc.expCheckNetFormatParametersErr)
			}
			t.Logf("find netName %s\n", netName)
		})
	}
}

// TestReconcileNetCreate has several tests to cover the code of the function reconcileNet
func TestReconcileNetCreate(t *testing.T) {
	netTestCases := []struct {
		name               string
		spec               infrastructurev1beta1.OscClusterSpec
		expNetFound        bool
		expCreateNetFound  bool
		expCreateNetErr    error
		expReconcileNetErr error
	}{
		{
			name:               "create Net (first time reconcile loop)",
			spec:               defaultNetInitialize,
			expNetFound:        false,
			expCreateNetFound:  true,
			expCreateNetErr:    nil,
			expReconcileNetErr: nil,
		},
		{
			name:               "failed create Net",
			spec:               defaultNetInitialize,
			expNetFound:        false,
			expCreateNetFound:  false,
			expCreateNetErr:    fmt.Errorf("CreateNet generic error"),
			expReconcileNetErr: fmt.Errorf("CreateNet generic error Can not create net for Osccluster test-system/test-osc"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netSpec := ntc.spec.Network.Net
			netId := "vpc-" + netName
			clusterName := ntc.spec.Network.ClusterName
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			if ntc.expCreateNetFound {
				mockOscNetInterface.
					EXPECT().
					CreateNet(gomock.Eq(&netSpec), gomock.Eq(clusterName), gomock.Eq(netName)).
					Return(net.Net, ntc.expCreateNetErr)
			} else {
				mockOscNetInterface.
					EXPECT().
					CreateNet(gomock.Eq(&netSpec), gomock.Eq(clusterName), gomock.Eq(netName)).
					Return(nil, ntc.expCreateNetErr)
			}
			reconcileNet, err := reconcileNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileNetErr.Error(), err.Error(), "reconcileNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileNetErr)
			}
			t.Logf("Find reconcileNet %v\n", reconcileNet)

		})

	}
}

// TestReconcileNetGet has several tests to cover the code of the function reconcileNet
func TestReconcileNetGet(t *testing.T) {
	netTestCases := []struct {
		name               string
		spec               infrastructurev1beta1.OscClusterSpec
		expNetFound        bool
		expCreateNetFound  bool
		expCreateNetErr    error
		expDescribeNetErr  error
		expReconcileNetErr error
	}{
		{
			name:               "check Net exist (second time reconcile loop)",
			spec:               defaultNetReconcile,
			expNetFound:        true,
			expCreateNetFound:  false,
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expReconcileNetErr: nil,
		},
		{
			name:               "failed Get Net",
			spec:               defaultNetReconcile,
			expNetFound:        false,
			expCreateNetFound:  false,
			expCreateNetErr:    nil,
			expDescribeNetErr:  fmt.Errorf("GetNet generic error"),
			expReconcileNetErr: fmt.Errorf("GetNet generic error"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			readNets := osc.ReadNetsResponse{
				Nets: &[]osc.Net{
					*net.Net,
				},
			}
			readNet := *readNets.Nets
			if ntc.expNetFound {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(&readNet[0], ntc.expDescribeNetErr)
			} else {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(nil, ntc.expDescribeNetErr)
			}
			reconcileNet, err := reconcileNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileNetErr, err, "reconcileNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileNetErr)
			}
			t.Logf("Find reconcileNet %v\n", reconcileNet)

		})

	}
}

// TestReconcileNetResourceId has several tests to cover the code of the function reconcileNet
func TestReconcileNetResourceId(t *testing.T) {
	netTestCases := []struct {
		name               string
		spec               infrastructurev1beta1.OscClusterSpec
		expCreateNetErr    error
		expDescribeNetErr  error
		expReconcileNetErr error
	}{
		{
			name:               "user delete net without cluster-api",
			spec:               defaultNetReconcile,
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expReconcileNetErr: nil,
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netSpec := ntc.spec.Network.Net
			clusterName := ntc.spec.Network.ClusterName
			netId := "vpc-" + netName
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			mockOscNetInterface.
				EXPECT().
				GetNet(gomock.Eq(netId)).
				Return(nil, ntc.expDescribeNetErr)
			mockOscNetInterface.
				EXPECT().
				CreateNet(gomock.Eq(&netSpec), gomock.Eq(clusterName), gomock.Eq(netName)).
				Return(net.Net, ntc.expCreateNetErr)
			reconcileNet, err := reconcileNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileNetErr, err, "reconcileNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileNetErr)
			}
			t.Logf("Find reconcileNet %v\n", reconcileNet)

		})

	}
}

// TestReconcileDeleteNetDelete has several tests to cover the code of the function reconcileDeleteNet
func TestReconcileDeleteNetDelete(t *testing.T) {
	netTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expNetFound              bool
		expDeleteNetErr          error
		expDescribeNetErr        error
		expReconcileDeleteNetErr error
	}{
		{
			name:                     "delete Net (first time reconcile loop)",
			spec:                     defaultNetReconcile,
			expNetFound:              true,
			expDeleteNetErr:          nil,
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: nil,
		},
		{
			name:                     "failed to delete Net",
			spec:                     defaultNetReconcile,
			expNetFound:              true,
			expDeleteNetErr:          fmt.Errorf("DeleteNet generic error"),
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: fmt.Errorf("DeleteNet generic error Can not delete net for Osccluster test-system/test-osc"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			readNets := osc.ReadNetsResponse{
				Nets: &[]osc.Net{
					*net.Net,
				},
			}
			readNet := *readNets.Nets
			if ntc.expNetFound {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(&readNet[0], ntc.expDescribeNetErr)
			} else {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(nil, ntc.expDescribeNetErr)
			}
			mockOscNetInterface.
				EXPECT().
				DeleteNet(gomock.Eq(netId)).
				Return(ntc.expDeleteNetErr)
			reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileDeleteNetErr.Error(), err.Error(), "reconcileDeleteNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileDeleteNetErr)
			}
			t.Logf("Find reconcileDeleteNet %v\n", reconcileDeleteNet)

		})

	}
}

// TestReconcileDeleteNetDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteNet
func TestReconcileDeleteNetDeleteWithoutSpec(t *testing.T) {
	netTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expNetFound              bool
		expDeleteNetErr          error
		expDescribeNetErr        error
		expReconcileDeleteNetErr error
	}{
		{
			name:                     "delete net without spec (with default values)",
			expNetFound:              true,
			expDeleteNetErr:          nil,
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: nil,
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := "cluster-api-net-uid"
			netId := "vpc-" + netName
			clusterScope.OscCluster.Spec.Network.Net.ResourceId = netId
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			readNets := osc.ReadNetsResponse{
				Nets: &[]osc.Net{
					*net.Net,
				},
			}
			readNet := *readNets.Nets
			mockOscNetInterface.
				EXPECT().
				GetNet(gomock.Eq(netId)).
				Return(&readNet[0], ntc.expDescribeNetErr)
			mockOscNetInterface.
				EXPECT().
				DeleteNet(gomock.Eq(netId)).
				Return(ntc.expDeleteNetErr)
			reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileDeleteNetErr, err, "reconcileDeleteNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileDeleteNetErr)
			}
			t.Logf("Find reconcileDeleteNet %v\n", reconcileDeleteNet)

		})

	}
}

// TestReconcileDeleteNetGet has several tests to cover the code of the function reconcileDeleteNet
func TestReconcileDeleteNetGet(t *testing.T) {
	netTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expNetFound              bool
		expDescribeNetErr        error
		expReconcileDeleteNetErr error
	}{
		{
			name:                     "Remove finalizer",
			spec:                     defaultNetReconcile,
			expNetFound:              false,
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: nil,
		},
		{
			name:                     "failed to get Net",
			spec:                     defaultNetReconcile,
			expNetFound:              false,
			expDescribeNetErr:        fmt.Errorf("GetNet generic error"),
			expReconcileDeleteNetErr: fmt.Errorf("GetNet generic error"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscNetInterface := SetupWithNetMock(t, ntc.name, ntc.spec)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			net := osc.CreateNetResponse{
				Net: &osc.Net{
					NetId: &netId,
				},
			}
			readNets := osc.ReadNetsResponse{
				Nets: &[]osc.Net{
					*net.Net,
				},
			}
			readNet := *readNets.Nets
			if ntc.expNetFound {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(&readNet[0], ntc.expDescribeNetErr)
			} else {
				mockOscNetInterface.
					EXPECT().
					GetNet(gomock.Eq(netId)).
					Return(nil, ntc.expDescribeNetErr)
			}
			reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				assert.Equal(t, ntc.expReconcileDeleteNetErr, err, "reconcileDeleteNet() should return the same error")
			} else {
				assert.Nil(t, ntc.expReconcileDeleteNetErr)
			}
			t.Logf("Find reconcileDeleteNet %v\n", reconcileDeleteNet)

		})

	}
}
