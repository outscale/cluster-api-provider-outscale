package controllers

import (
	"context"
	"fmt"

	"testing"

	"k8s.io/klog/v2/klogr"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// TestGetNetResourceId has several tests to cover the code of the function getNetResourceId
func TestGetNetResourceId(t *testing.T) {
	netTestCases := []struct {
		name                   string
		spec                   infrastructurev1beta1.OscClusterSpec
		expNetFound            bool
		expGetNetResourceIdErr error
	}{
		{
			name: "get NetId",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
				},
			},
			expNetFound:            true,
			expGetNetResourceIdErr: nil,
		},
		{
			name: "can not get netId",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
				},
			},
			expNetFound:            false,
			expGetNetResourceIdErr: fmt.Errorf("test-net-uid is not exist"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			t.Logf("Validate to %s", ntc.name)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: ntc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID: "uid",
				},
			}
			log := klogr.New()
			clusterScope := &scope.ClusterScope{
				Logger: log,
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						UID: "uid",
					},
				},
				OscCluster: &oscCluster,
			}
			if ntc.expNetFound {
				netRef := clusterScope.GetNetRef()
				netRef.ResourceMap = make(map[string]string)
				netRef.ResourceMap[netName] = netId
			}
			netResourceId, err := getNetResourceId(netName, clusterScope)
			if err != nil {
				if err.Error() != ntc.expGetNetResourceIdErr.Error() {
					t.Errorf("getNetResourceId() expected error = %s, received error = %s", ntc.expGetNetResourceIdErr, err.Error())
				}
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
			name: "Check default value",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckNetFormatParametersErr: nil,
		},
		{
			name: "check Net Format",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
				},
			},
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
			t.Logf("Validate to %s", ntc.name)
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: ntc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID: "uid",
				},
			}
			log := klogr.New()
			clusterScope := &scope.ClusterScope{
				Logger: log,
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						UID: "uid",
					},
				},
				OscCluster: &oscCluster,
			}
			netName, err := checkNetFormatParameters(clusterScope)
			if err != nil {
				if err.Error() != ntc.expCheckNetFormatParametersErr.Error() {
					t.Errorf("checkNetFormatParameters() expected error = %s, received error %s", ntc.expCheckNetFormatParametersErr, err.Error())
				}
			}
			t.Logf("find netName %s\n", netName)
		})
	}
}

// TestReconcileNet has several tests to cover the code of the function reconcileNet
func TestReconcileNet(t *testing.T) {
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
			name: "create Net (first time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
				},
			},
			expNetFound:        false,
			expCreateNetFound:  true,
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expReconcileNetErr: nil,
		},
		{
			name: "check Net exist (second time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:        true,
			expCreateNetFound:  false,
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expReconcileNetErr: nil,
		},
		{
			name: "delete Net without cluster-api",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:        false,
			expCreateNetFound:  true,
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expReconcileNetErr: nil,
		},
		{
			name: "failed create Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
				},
			},
			expNetFound:        false,
			expCreateNetFound:  false,
			expCreateNetErr:    fmt.Errorf("CreateNet generic error"),
			expDescribeNetErr:  nil,
			expReconcileNetErr: fmt.Errorf("CreateNet generic error Can not create net for Osccluster test-system/test-osc"),
		},
		{
			name: "failed Get Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:        false,
			expCreateNetFound:  false,
			expCreateNetErr:    nil,
			expDescribeNetErr:  fmt.Errorf("GetNet generic error"),
			expReconcileNetErr: fmt.Errorf("GetNet generic error"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			t.Logf("Validate to %s", ntc.name)
			mockCtrl := gomock.NewController(t)
			mockOscNetInterface := mock_net.NewMockOscNetInterface(mockCtrl)
			netName := ntc.spec.Network.Net.Name + "-uid"
			netSpec := ntc.spec.Network.Net
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
			ctx := context.Background()
			if ntc.expCreateNetFound {
				mockOscNetInterface.EXPECT().CreateNet(&netSpec, netName).Return(net.Net, ntc.expCreateNetErr).AnyTimes()
			} else {
				mockOscNetInterface.EXPECT().CreateNet(&netSpec, netName).Return(nil, ntc.expCreateNetErr).AnyTimes()
			}
			if ntc.expNetFound {
				mockOscNetInterface.EXPECT().GetNet(netId).Return(&readNet[0], ntc.expDescribeNetErr).AnyTimes()
			} else {
				mockOscNetInterface.EXPECT().GetNet(netId).Return(nil, ntc.expDescribeNetErr).AnyTimes()
			}
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: ntc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID:       "uid",
					Name:      "test-osc",
					Namespace: "test-system",
				},
			}

			log := klogr.New()
			clusterScope := &scope.ClusterScope{
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
			reconcileNet, err := reconcileNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				if err.Error() != ntc.expReconcileNetErr.Error() {
					t.Errorf("reconcileNet() expected error = %s, received error = %s", ntc.expReconcileNetErr, err.Error())
				}
			}
			t.Logf("Find reconcileNet %v\n", reconcileNet)

		})

	}
}

// TestReconcileDeleteNet has several tests to cover the code of the function reconcileDeleteNet
func TestReconcileDeleteNet(t *testing.T) {
	netTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expNetFound              bool
		expDeleteNetErr          error
		expDescribeNetErr        error
		expReconcileDeleteNetErr error
	}{
		{
			name: "delete Net (first time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:              true,
			expDeleteNetErr:          nil,
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: nil,
		},
		{
			name: "Remove finalizer",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:              false,
			expDeleteNetErr:          nil,
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: nil,
		},
		{
			name: "failed to get Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "172.195.95.128/25",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:              false,
			expDeleteNetErr:          nil,
			expDescribeNetErr:        fmt.Errorf("GetNet generic error"),
			expReconcileDeleteNetErr: fmt.Errorf("GetNet generic error"),
		},
		{
			name: "failed to delete Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "172.195.95.128/25",
						ResourceId: "vpc-test-net-uid",
					},
				},
			},
			expNetFound:              true,
			expDeleteNetErr:          fmt.Errorf("DeleteNet generic error"),
			expDescribeNetErr:        nil,
			expReconcileDeleteNetErr: fmt.Errorf("DeleteNet generic error Can not delete net for Osccluster test-system/test-osc"),
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			t.Logf("Validate to %s", ntc.name)
			mockCtrl := gomock.NewController(t)
			mockOscNetInterface := mock_net.NewMockOscNetInterface(mockCtrl)
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
			ctx := context.Background()
			mockOscNetInterface.EXPECT().DeleteNet(netId).Return(ntc.expDeleteNetErr).AnyTimes()
			if ntc.expNetFound {
				mockOscNetInterface.EXPECT().GetNet(netId).Return(&readNet[0], ntc.expDescribeNetErr).AnyTimes()
			} else {
				mockOscNetInterface.EXPECT().GetNet(netId).Return(nil, ntc.expDescribeNetErr).AnyTimes()
			}
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: ntc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID:       "uid",
					Name:      "test-osc",
					Namespace: "test-system",
				},
			}
			log := klogr.New()
			clusterScope := &scope.ClusterScope{
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
			reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				if err.Error() != ntc.expReconcileDeleteNetErr.Error() {
					t.Errorf("reconcileDeleteNet() expected error = %s, received error %s", ntc.expReconcileDeleteNetErr, err.Error())
				}
			}
			t.Logf("Find reconcileDeleteNet %v\n", reconcileDeleteNet)

		})

	}
}
