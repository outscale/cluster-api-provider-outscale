package controllers

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net/mock_net"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestReconcileNet(t *testing.T) {
	netTestCases := []struct {
		name               string
		spec               infrastructurev1beta1.OscClusterSpec
		expCreateNetErr    error
		expDescribeNetErr  error
		expNetReconcileErr error
	}{
		{
			name: "create Net",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "172.195.95.128/25",
					},
				},
			},
			expCreateNetErr:    nil,
			expDescribeNetErr:  nil,
			expNetReconcileErr: nil,
		},
	}
	for _, ntc := range netTestCases {
		t.Run(ntc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockOscNetInterface := mock_net.NewMockOscNetInterface(mockCtrl)
			netName := ntc.spec.Network.Net.Name
			netRealName := ntc.spec.Network.Net.Name + "-aaaaa"
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
			mockOscNetInterface.EXPECT().CreateNet(&netSpec, netRealName).Return(net.Net, ntc.expCreateNetErr).AnyTimes()
			mockOscNetInterface.EXPECT().GetNet(netId).Return(&readNet[0], ntc.expDescribeNetErr).AnyTimes()
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: ntc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID: "aaaaa",
				},
			}

			//			fake.NewClientBuilder().WithObjects(oscCluster, credentialSecret).Build()
			log := ctrl.LoggerFrom(ctx)
			clusterScope := &scope.ClusterScope{
				Logger: log,
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						UID: "aaaaa",
					},
				},
				OscCluster: &oscCluster,
			}
			fmt.Printf("Find info %v", clusterScope)
			reconcileNet, err := reconcileNet(ctx, clusterScope, mockOscNetInterface)
			if err != nil {
				if err != ntc.expNetReconcileErr {
					t.Errorf("ReconcileNet() expected error = %s, received error %s", ntc.expNetReconcileErr, err.Error())
				}
			}
			fmt.Printf("Find reconcileNet info %v", reconcileNet)

		})

	}
}
