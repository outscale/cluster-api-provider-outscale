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

package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscMachine_ValidateCreate check good and bad validation of oscMachine spec
func TestOscMachine_ValidateCreate(t *testing.T) {
	machineTestCases := []struct {
		name                 string
		machineSpec          OscMachineSpec
		expValidateCreateErr error
	}{
		{
			name: "create Valid Vm Spec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId:     "ami-01234567",
						KeypairName: "test-webhook",
						DeviceName:  "/dev/xvdb",
						VmType:      "tinav4.c2r4p2",
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with bad keypairName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "rke λ",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: keypairName: Invalid value: \"rke λ\": Invalid KeypairName"),
		},
		{
			name: "create with bad vmType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						VmType: "oscv4.c2r4p2",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: vmType: Invalid value: \"oscv4.c2r4p2\": Invalid vmType"),
		},
		{
			name: "create with bad iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          -15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-weest-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: iops: Invalid value: -15: Invalid iops"),
		},
		{
			name: "create rootdisk with bad iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskIops: -15,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: iops: Invalid value: -15: Invalid iops"),
		},
		{
			name: "create rootdisk with bad size",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskSize: -15,
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: size: Invalid value: -15: Invalid size"),
		},
		{
			name: "create rootdisk with bad volumeType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskType: "ssd1",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: diskType: Invalid value: \"ssd1\": Invalid volumeType"),
		},
		{
			name: "create with bad volumeType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          30,
							VolumeType:    "ssd1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: volumeType: Invalid value: \"ssd1\": Invalid volumeType"),
		},
		{
			name: "create with bad subregionName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2c",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: subregionName: Invalid value: \"eu-west-2c\": Invalid subregionName"),
		},
		{
			name: "create with good io1 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          20,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with bad io1 ratio size iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          2000,
							Size:          20,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with good gp2 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Size:          20,
							VolumeType:    "gp2",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with good standard volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Size:          20,
							VolumeType:    "standard",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscInfraMachine := createOscInfraMachine(mtc.machineSpec, "webhook-test", "default")
			_, err := oscInfraMachine.ValidateCreate()
			if err != nil {
				assert.Equal(t, mtc.expValidateCreateErr.Error(), err.Error(), "ValidateCreate should return the same errror")
			} else {
				assert.Nil(t, mtc.expValidateCreateErr)
			}
		})
	}
}

// TestOscMachine_ValidateUpdate check good and bad update of oscMachine
func TestOscMachine_ValidateUpdate(t *testing.T) {
	machineTestCases := []struct {
		name                 string
		oldMachineSpec       OscMachineSpec
		machineSpec          OscMachineSpec
		expValidateUpdateErr error
	}{
		{
			name: "update only oscMachine name",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateUpdateErr: nil,
		},
		{
			name: "update keypairName",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook-1",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook-2",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.keyPairName: Invalid value: \"test-webhook-2\": field is immutable"),
		},
		{
			name: "update vmType",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						VmType: "tinav4.c2r4p2",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						VmType: "tinav4.c2r4p1",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.vmType: Invalid value: \"tinav4.c2r4p1\": field is immutable"),
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachine := createOscInfraMachine(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachine := createOscInfraMachine(mtc.machineSpec, "webhook-test", "default")
			_, err := oscInfraMachine.ValidateUpdate(oscOldInfraMachine)
			if err != nil {
				assert.Equal(t, mtc.expValidateUpdateErr.Error(), err.Error(), "ValidateUpdate() should return the same error")
			} else {
				assert.Nil(t, mtc.expValidateUpdateErr)
			}
		})
	}
}

// createOscInfraMachine create oscInfraMachine
func createOscInfraMachine(infraMachineSpec OscMachineSpec, name string, namespace string) *OscMachine {
	oscInfraMachine := &OscMachine{
		Spec: infraMachineSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraMachine
}
