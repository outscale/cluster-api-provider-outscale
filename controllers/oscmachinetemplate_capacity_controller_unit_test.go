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
	"testing"

	gomock "github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute/mock_compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var (
	defaultVmMachineTemplateClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:        "test-net",
				IpRange:     "10.0.0.0/16",
				ClusterName: "test-cluster",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
				},
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
						{
							Name:          "test-securitygrouprule",
							Flow:          "Inbound",
							IpProtocol:    "tcp",
							IpRange:       "0.0.0.0/0",
							FromPortRange: 6443,
							ToPortRange:   6443,
						},
					},
				},
			},
			LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
				LoadBalancerName:  "test-loadbalancer",
				LoadBalancerType:  "internet-facing",
				SubnetName:        "test-subnet",
				SecurityGroupName: "test-securitygroup",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name: "test-publicip",
				},
			},
		},
	}

	defaultVmMachineTemplateInitialize = infrastructurev1beta1.OscMachineTemplateSpec{
		Template: infrastructurev1beta1.OscMachineTemplateResource{
			Spec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          50,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
					Vm: infrastructurev1beta1.OscVm{
						ClusterName: "test-cluster",
						Name:        "test-vm",
						ImageId:     "ami-00000000",
						Role:        "controlplane",
						DeviceName:  "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						PublicIpName:     "test-publicip",
						VmType:           "tinav3.c2r4p2",
						Replica:          1,
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
		},
	}
)

// Setup set osccluster, oscmachine, machineScope and clusterScope
func SetupMachineTemplate(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineTemplateSpec) (clusterScope *scope.ClusterScope, machineTemplateScope *scope.MachineTemplateScope) {
	t.Logf("Validate to %s", name)

	oscCluster := infrastructurev1beta1.OscCluster{
		Spec: clusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			UID:       "uid",
			Name:      "test-osc",
			Namespace: "test-system",
		},
	}
	oscMachineTemplate := infrastructurev1beta1.OscMachineTemplate{
		Spec: machineSpec,
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
	machineTemplateScope = &scope.MachineTemplateScope{
		Logger:             log,
		OscMachineTemplate: &oscMachineTemplate,
	}
	return clusterScope, machineTemplateScope
}

// SetupWithVmCapacityMock set vmMock with clusterScope, machineScope and oscmachine
func SetupWithVmCapacityMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineTemplateSpec infrastructurev1beta1.OscMachineTemplateSpec) (clusterScope *scope.ClusterScope, machineTemplateScope *scope.MachineTemplateScope, ctx context.Context, mockOscVmInterface *mock_compute.MockOscVmInterface) {
	clusterScope, machineTemplateScope = SetupMachineTemplate(t, name, clusterSpec, machineTemplateSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVmInterface = mock_compute.NewMockOscVmInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineTemplateScope, ctx, mockOscVmInterface
}

// TestReconcileCapacity has several tests to cover the code of the function reconcileCapacity
func TestReconcileCapacity(t *testing.T) {
	capacityTestCases := []struct {
		name                    string
		clusterSpec             infrastructurev1beta1.OscClusterSpec
		machineTemplateSpec     infrastructurev1beta1.OscMachineTemplateSpec
		expGetCapacityFound     bool
		expGetCapacityErr       error
		expReconcileCapacityErr error
	}{
		{
			name:                    "create vm (first time reconcile loop)",
			clusterSpec:             defaultVmMachineTemplateClusterInitialize,
			machineTemplateSpec:     defaultVmMachineTemplateInitialize,
			expGetCapacityFound:     true,
			expGetCapacityErr:       nil,
			expReconcileCapacityErr: nil,
		},
	}
	for _, ctc := range capacityTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			clusterScope, machineTemplateScope, ctx, mockOscVmInterface := SetupWithVmCapacityMock(t, ctc.name, ctc.clusterSpec, ctc.machineTemplateSpec)
			clusterName := ctc.clusterSpec.Network.Net.ClusterName
			tagKey := "OscK8sClusterID/" + clusterName + "-uid"
			tagValue := "owned"
			vmType := ctc.machineTemplateSpec.Template.Spec.Node.Vm.VmType
			capacity := make(corev1.ResourceList)
			memory, err := resource.ParseQuantity("4G")
			require.NoError(t, err)
			cpu, err := resource.ParseQuantity("2")
			require.NoError(t, err)
			capacity[corev1.ResourceMemory] = memory
			capacity[corev1.ResourceCPU] = cpu

			if ctc.expGetCapacityFound {
				mockOscVmInterface.
					EXPECT().
					GetCapacity(gomock.Eq(tagKey), gomock.Eq(tagValue), gomock.Eq(vmType)).
					Return(capacity, ctc.expReconcileCapacityErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetCapacity(gomock.Eq(tagKey), gomock.Eq(tagValue), gomock.Eq(vmType)).
					Return(nil, ctc.expReconcileCapacityErr)
			}

			reconcileCapacity, err := reconcileCapacity(ctx, clusterScope, machineTemplateScope, mockOscVmInterface)
			if ctc.expReconcileCapacityErr != nil {
				assert.Equal(t, err, ctc.expReconcileCapacityErr.Error(), "reconcileVmCapacity() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileCapacity %v\n", reconcileCapacity)
		})
	}
}
