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

// TestGetSubnetResourceId has several tests to cover the code of the function getSubnetResourceId
func TestGetSubnetResourceId(t *testing.T) {
	subnetTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expSubnetFound            bool
		expGetSubnetResourceIdErr error
	}{
		{
			name: "get SubnetId",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:            true,
			expGetSubnetResourceIdErr: nil,
		},
		{
			name: "failed to get Subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:            false,
			expGetSubnetResourceIdErr: fmt.Errorf("test-subnet-uid is not exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			subnetsSpec := stc.spec.Network.Subnets
			t.Logf("Validate to %s", stc.name)
			log := klogr.New()
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				oscCluster := infrastructurev1beta1.OscCluster{
					Spec: stc.spec,
					ObjectMeta: metav1.ObjectMeta{
						UID: "uid",
					},
				}
				clusterScope := &scope.ClusterScope{
					Logger: log,
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							UID: "uid",
						},
					},
					OscCluster: &oscCluster,
				}
				if stc.expSubnetFound {
					subnetRef := clusterScope.GetSubnetRef()
					subnetRef.ResourceMap = make(map[string]string)
					subnetRef.ResourceMap[subnetName] = subnetId
				}
				subnetResourceId, err := getSubnetResourceId(subnetName, clusterScope)
				if err != nil {
					if err.Error() != stc.expGetSubnetResourceIdErr.Error() {
						t.Errorf("getSubnetResourceId() expected error = %s, received error = %s", stc.expGetSubnetResourceIdErr, err.Error())
					}
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
			name: "get separate Name",
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
							Name:          "test-subnet-second",
							IpSubnetRange: "10.0.1.0/24",
						},
					},
				},
			},
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
			t.Logf("Validate to %s", stc.name)
			log := klogr.New()
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: stc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID: "uid",
				},
			}
			clusterScope := &scope.ClusterScope{
				Logger: log,
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						UID: "uid",
					},
				},
				OscCluster: &oscCluster,
			}

			duplicateResourceSubnetErr := checkSubnetOscDuplicateName(clusterScope)
			if duplicateResourceSubnetErr != nil {
				if duplicateResourceSubnetErr.Error() != stc.expCheckSubnetOscDuplicateNameErr.Error() {
					t.Errorf("checkSubnetOscDupicateNamee() expected error = %s, received error %s", stc.expCheckSubnetOscDuplicateNameErr, duplicateResourceSubnetErr.Error())
				}
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
			name: "Check default value",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckSubnetFormatParametersErr: nil,
		},
		{
			name: "check Subnet format",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
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
			t.Logf("Validate to %s", stc.name)
			log := klogr.New()
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: stc.spec,
				ObjectMeta: metav1.ObjectMeta{
					UID: "uid",
				},
			}
			clusterScope := &scope.ClusterScope{
				Logger: log,
				Cluster: &clusterv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						UID: "uid",
					},
				},
				OscCluster: &oscCluster,
			}

			subnetName, err := checkSubnetFormatParameters(clusterScope)
			if err != nil {
				if err.Error() != stc.expCheckSubnetFormatParametersErr.Error() {
					t.Errorf("checkSubnetFormatParameters() expected error = %S, received error %s", stc.expCheckSubnetFormatParametersErr, err.Error())
				}
			}
			t.Logf("find subnetName %s\n", subnetName)

		})
	}
}

// TestReconcileSubnet has several tests to cover the code of the function reconcileSubnet
func TestReconcileSubnet(t *testing.T) {
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
			name: "create Subnet (first time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name: "check Subnet exist (second time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:        true,
			expNetFound:           true,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name: "create two Subnets (first time reconcile loop)",
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
							Name:          "test-subnet-second",
							IpSubnetRange: "10.0.1.0/24",
						},
					},
				},
			},
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name: "check two subnets exist (second time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:        true,
			expNetFound:           true,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name: "failed to create subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    fmt.Errorf("CreateSubnet generic error"),
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: fmt.Errorf("CreateSubnet generic error Can not create subnet for Osccluster test-system/test-osc"),
		},
		{
			name: "failed to get Subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    fmt.Errorf("GetSubnet generic error"),
			expReconcileSubnetErr: fmt.Errorf("GetSubnet generic error"),
		},
		{
			name: "delete subnet without cluster-api",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:        false,
			expNetFound:           true,
			expCreateSubnetFound:  true,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: nil,
		},
		{
			name: "Net does not exist",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:        false,
			expNetFound:           false,
			expCreateSubnetFound:  false,
			expCreateSubnetErr:    nil,
			expGetSubnetIdsErr:    nil,
			expReconcileSubnetErr: fmt.Errorf("test-net-uid is not exist"),
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			t.Logf("Validate to %s", stc.name)
			mockCtrl := gomock.NewController(t)
			mockOscSubnetInterface := mock_net.NewMockOscSubnetInterface(mockCtrl)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			subnetsSpec := stc.spec.Network.Subnets
			ctx := context.Background()
			var subnetIds []string
			var clusterScope *scope.ClusterScope
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				subnetIds = append(subnetIds, subnetId)
				subnet := osc.CreateSubnetResponse{
					Subnet: &osc.Subnet{
						SubnetId: &subnetId,
					},
				}
				oscCluster := infrastructurev1beta1.OscCluster{
					Spec: stc.spec,
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

				subnetRef := clusterScope.GetSubnetRef()
				subnetRef.ResourceMap = make(map[string]string)
				if stc.expCreateSubnetFound {
					subnetRef.ResourceMap[subnetName] = subnetId
					mockOscSubnetInterface.EXPECT().CreateSubnet(subnetSpec, netId, subnetName).Return(subnet.Subnet, stc.expCreateSubnetErr).AnyTimes()
				} else {
					mockOscSubnetInterface.EXPECT().CreateSubnet(subnetSpec, netId, subnetName).Return(nil, stc.expCreateSubnetErr).AnyTimes()
				}
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.EXPECT().GetSubnetIdsFromNetIds(netId).Return(subnetIds, stc.expGetSubnetIdsErr).AnyTimes()
			} else {
				mockOscSubnetInterface.EXPECT().GetSubnetIdsFromNetIds(netId).Return(nil, stc.expGetSubnetIdsErr).AnyTimes()
			}
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			reconcileSubnet, err := reconcileSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				if err.Error() != stc.expReconcileSubnetErr.Error() {
					t.Errorf("reconcileSubnet() expected error = %s, received error %s", stc.expReconcileSubnetErr, err.Error())
				}
			}
			t.Logf("Find reconcileSubnet  %v\n", reconcileSubnet)
		})
	}
}

// TestReconcileDeleteSubnet has several tests to cover the code of the function reconcileDeleteSubnet
func TestReconcileDeleteSubnet(t *testing.T) {
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
			name: "delete Net (first time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
		{
			name: "delete two Net (first time reconcile loop)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
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
			},
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
		{
			name: "Failed to get subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:              false,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          fmt.Errorf("GetSubnet generic error"),
			expReconcileDeleteSubnetErr: fmt.Errorf("GetSubnet generic error"),
		},
		{
			name: "failed to delete Subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:              true,
			expNetFound:                 true,
			expDeleteSubnetErr:          fmt.Errorf("DeleteSubnet generic error"),
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: fmt.Errorf("DeleteSubnet generic error Can not delete subnet for Osccluster test-system/test-osc"),
		},
		{
			name: "Net does not exist",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:              false,
			expNetFound:                 false,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: fmt.Errorf("test-net-uid is not exist"),
		},
		{
			name: "Remove finalizer (delete Subnets without cluster-api)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:       "test-net",
						IpRange:    "10.0.0.0/16",
						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
							ResourceId:    "subnet-test-subnet-uid",
						},
					},
				},
			},
			expSubnetFound:              false,
			expNetFound:                 true,
			expDeleteSubnetErr:          nil,
			expGetSubnetIdsErr:          nil,
			expReconcileDeleteSubnetErr: nil,
		},
	}
	for _, stc := range subnetTestCases {
		t.Run(stc.name, func(t *testing.T) {
			t.Logf("Validate to %s", stc.name)
			mockCtrl := gomock.NewController(t)
			mockOscSubnetInterface := mock_net.NewMockOscSubnetInterface(mockCtrl)
			netName := stc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			subnetsSpec := stc.spec.Network.Subnets
			var subnetIds []string
			ctx := context.Background()
			for _, subnetSpec := range subnetsSpec {
				subnetName := subnetSpec.Name + "-uid"
				subnetId := "subnet-" + subnetName
				subnetIds = append(subnetIds, subnetId)
				mockOscSubnetInterface.EXPECT().DeleteSubnet(subnetId).Return(stc.expDeleteSubnetErr).AnyTimes()
			}
			if stc.expSubnetFound {
				mockOscSubnetInterface.EXPECT().GetSubnetIdsFromNetIds(netId).Return(subnetIds, stc.expGetSubnetIdsErr).AnyTimes()
			} else {
				mockOscSubnetInterface.EXPECT().GetSubnetIdsFromNetIds(netId).Return(nil, stc.expGetSubnetIdsErr).AnyTimes()
			}
			oscCluster := infrastructurev1beta1.OscCluster{
				Spec: stc.spec,
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

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if stc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, mockOscSubnetInterface)
			if err != nil {
				if err.Error() != stc.expReconcileDeleteSubnetErr.Error() {
					t.Errorf("reconcileDeleteSubnet() expected error %s, received error %s", stc.expReconcileDeleteSubnetErr, err.Error())
				}
			}
			t.Logf("Find reconcileDeleteSubnet %v\n", reconcileDeleteSubnet)
		})
	}
}
