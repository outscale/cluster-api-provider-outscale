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

package v1beta1_test

import (
	"context"
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscMachineTemplate_ValidateCreate check good and bad update of oscMachineTemplate
func TestOscMachineTemplate_ValidateCreate(t *testing.T) {
	machineTestCases := []struct {
		name        string
		machineSpec infrastructurev1beta1.OscMachineSpec
		errorCount  int
	}{
		{
			name: "create Valid Vm Spec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						ImageId:     "ami-01234567",
						KeypairName: "test-webhook",
						VmType:      "tinav3.c2r4p2",
						Tags:        map[string]string{"key1": "value1"},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with bad vmType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "oscv4.c2r4p2",
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad iops",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskIops: -15,
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad size",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskSize: -15,
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad volumeType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskType: "gp3",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with bad volume size",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       -30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with bad volume iops",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       -15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with missing volume device",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Iops:       20,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with invalid volume device",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "foo",
							Iops:       20,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with bad volumeType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       30,
							VolumeType: "gp3",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with valid io1 volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       20,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with valid gp2 volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "gp2",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with valid standard volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "standard",
						},
					},
				},
			},
			errorCount: 0,
		},
	}
	h := infrastructurev1beta1.OscMachineTemplateWebhook{}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscInfraMachineTemplate := createOscInfraMachineTemplate(mtc.machineSpec, "webhook-test", "default")
			_, err := h.ValidateCreate(context.TODO(), oscInfraMachineTemplate)
			if mtc.errorCount > 0 {
				require.Error(t, err, "ValidateCreate should return the same errror")
				require.Len(t, err.(*apierrors.StatusError).Status().Details.Causes, mtc.errorCount)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestOscMachineTemplate_ValidateUpdate check good and bad update of oscMachineTemplate
func TestOscMachineTemplate_ValidateUpdate(t *testing.T) {
	machineTestCases := []struct {
		name           string
		oldMachineSpec infrastructurev1beta1.OscMachineSpec
		machineSpec    infrastructurev1beta1.OscMachineSpec
		hasError       bool
	}{
		{
			name: "Update only oscMachineTemplate name",
			oldMachineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						VmType:      "tinav3.c2r4p2",
						PublicIp:    false,
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						VmType:      "tinav3.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
		},
		{
			name: "update one element (keypair)",
			oldMachineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook-1",
						ImageId:     "ami-00000000",
						VmType:      "tinav3.c2r4p2",
					},
				},
			},
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook-2",
						ImageId:     "ami-00000000",
						VmType:      "tinav3.c2r4p2",
					},
				},
			},
			hasError: true,
		},
	}
	h := infrastructurev1beta1.OscMachineTemplateWebhook{}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachineTemplate := createOscInfraMachineTemplate(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachineTemplate := createOscInfraMachineTemplate(mtc.machineSpec, "webhook-test", "default")
			_, err := h.ValidateUpdate(context.TODO(), oscInfraMachineTemplate, oscOldInfraMachineTemplate)
			if mtc.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createOscInfraMachineTemplate create oscInfraMachineTemplate
func createOscInfraMachineTemplate(infraMachineSpec infrastructurev1beta1.OscMachineSpec, name string, namespace string) *infrastructurev1beta1.OscMachineTemplate {
	oscInfraMachineTemplate := &infrastructurev1beta1.OscMachineTemplate{
		Spec: infrastructurev1beta1.OscMachineTemplateSpec{
			Template: infrastructurev1beta1.OscMachineTemplateResource{
				Spec: infraMachineSpec,
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraMachineTemplate
}
