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
			name: "create with bad imageId",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId: "omi-00000000",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: imageId: Invalid value: \"omi-00000000\": Invalid imageId"),
		},
		{
			name: "create with bad deviceName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						DeviceName: "/dev/xvaa",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: deviceName: Invalid value: \"/dev/xvaa\": Invalid deviceName"),
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
			name: "create with bad subregionname",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						SubregionName: "eu-west-2c",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: subregionName: Invalid value: \"eu-west-2c\": Invalid subregionName"),
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
			name: "create with bad size",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          -30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: size: Invalid value: -30: Invalid size"),
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
							VolumeType:    "gp3",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: volumeType: Invalid value: \"gp3\": Invalid volumeType"),
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
			err := oscInfraMachine.ValidateCreate()
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
			name: "update imageId",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId: "ami-012345678",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId: "ami-00000000",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.imageId: Invalid value: \"ami-00000000\": field is immutable"),
		},
		{
			name: "update deviceName",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						DeviceName: "/dev/xvda",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						DeviceName: "/dev/xvdb",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.deviceName: Invalid value: \"/dev/xvdb\": field is immutable"),
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
		{
			name: "update volume iops",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
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
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          30,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.iops: Invalid value: 30: field is immutable"),
		},
		{
			name: "update volume size",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
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
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          45,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.size: Invalid value: 45: field is immutable"),
		},
		{
			name: "update subregion name",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
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
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2b",
						},
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: spec.subregionName: Invalid value: \"eu-west-2b\": field is immutable"),
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachine := createOscInfraMachine(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachine := createOscInfraMachine(mtc.machineSpec, "webhook-test", "default")
			err := oscInfraMachine.ValidateUpdate(oscOldInfraMachine)
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
