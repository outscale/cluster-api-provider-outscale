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
	"errors"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	minIops = 0
	maxIops = 13000
	minSize = 0
	maxSize = 14901
)

// ValidateOscMachineSpec validates a OscMachineSpec.
func ValidateOscMachineSpec(spec OscMachineSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Node.Vm.KeypairName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.KeypairName, field.NewPath("node", "vm", "keypairName"), ValidateKeypairName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.KeyPair.Name != "" && spec.Node.Vm.KeypairName != spec.Node.KeyPair.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("node", "keypair", "name"), spec.Node.Vm.KeypairName, "keypairs must be the same in vm and keypair sections"))
	}
	if spec.Node.Vm.VmType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.VmType, field.NewPath("node", "vm", "vmType"), ValidateVmType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.SubregionName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.SubregionName, field.NewPath("node", "vm", "subregionName"), ValidateSubregionName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if len(spec.Node.Volumes) != 0 {
		volumesSpec := spec.Node.Volumes
		for _, volSpec := range volumesSpec {
			if err := ValidateDeviceName(volSpec.Device); err != nil {
				allErrs = append(allErrs, field.Invalid(field.NewPath("node", "volumes", "device"), volSpec.Device, err.Error()))
			}

			if volSpec.Iops != 0 {
				if errs := ValidateAndReturnErrorList(volSpec.Iops, field.NewPath("node", "volumes", "iops"), ValidateIops); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volSpec.Size != 0 {
				if errs := ValidateAndReturnErrorList(volSpec.Size, field.NewPath("node", "volumes", "size"), ValidateSize); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volSpec.VolumeType != "" {
				if errs := ValidateAndReturnErrorList(volSpec.VolumeType, field.NewPath("node", "volumes", "volumeType"), ValidateVolumeType); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
			if volSpec.Iops != 0 && volSpec.Size != 0 && volSpec.VolumeType == "io1" {
				ratioIopsSize := volSpec.Iops / volSpec.Size
				if errs := ValidateAndReturnErrorList(ratioIopsSize, field.NewPath("node", "volumes", "size"), ValidateRatioSizeIops); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskIops != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskIops, field.NewPath("node", "vm", "rootDisk", "rootDiskIops"), ValidateIops); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskSize != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskSize, field.NewPath("node", "vm", "rootDisk", "rootDiskSize"), ValidateSize); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskType, field.NewPath("node", "vm", "rootDisk", "rootDiskType"), ValidateVolumeType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskIops != 0 && spec.Node.Vm.RootDisk.RootDiskSize != 0 && spec.Node.Vm.RootDisk.RootDiskType == "io1" {
		ratioIopsSize := spec.Node.Vm.RootDisk.RootDiskIops / spec.Node.Vm.RootDisk.RootDiskSize
		if errs := ValidateAndReturnErrorList(ratioIopsSize, field.NewPath("node", "vm", "rootDisk", "rootDiskSize"), ValidateRatioSizeIops); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

var isValidateKeypairName = regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString

// ValidateKeypairName checks that KeypairName is a valid name of keypair
func ValidateKeypairName(keypairName string) error {
	if isValidateKeypairName(keypairName) {
		return nil
	} else {
		return errors.New("invalid keypair name")
	}
}

// ValidateImageId checks that imageId is a valid imageId
func ValidateImageId(imageId string) error {
	switch {
	case strings.HasPrefix(imageId, "ami"):
		return nil
	default:
		return errors.New("invalid image id")
	}
}

// ValidateRatioSizeIops checks that Ratio iops size should not exceed 300
func ValidateRatioSizeIops(ratioIopsSize int32) error {
	if ratioIopsSize < 300 {
		return nil
	} else {
		return errors.New("iops/size should be lower than 300")
	}
}

var isValidateName = regexp.MustCompile(`^[0-9A-Za-z\-_\s\.\(\)\\]{0,255}$`).MatchString

// ValidateIamegName checks that Image name is a valid name
func ValidateImageName(imageName string) error {
	if isValidateName(imageName) {
		return nil
	} else {
		return errors.New("invalid image name")
	}
}

// ValidateIops checks that iops is valid
func ValidateIops(iops int32) error {
	if iops < maxIops && iops > minIops {
		return nil
	} else {
		return errors.New("invalid iops")
	}
}

// ValidateSize checks that size is valid
func ValidateSize(size int32) error {
	if size < maxSize && size > minSize {
		return nil
	} else {
		return errors.New("invalid size")
	}
}

// ValidateVolumeType checks that volumeType is a valid volumeType
func ValidateVolumeType(volumeType string) error {
	switch volumeType {
	case "standard", "gp2", "io1":
		return nil
	default:
		return errors.New("invalid volume type (allowed: standard, gp2, io1)")
	}
}

var isValidSubregion = regexp.MustCompile(`(cloudgouv-)?(eu|us|ap)-(north|east|south|west|northeast|northwest|southeast|southwest)-[1-2][a-c]`).MatchString

// ValidateSubregionName checks that subregionName is a valid az format
func ValidateSubregionName(subregionName string) error {
	switch {
	case isValidSubregion(subregionName):
		return nil
	default:
		return errors.New("invalid subregion name")
	}
}

var isValidateDeviceName = regexp.MustCompile(`^(\/dev\/sda1|\/dev\/sd[a-z]{1}|\/dev\/xvd[a-z]{1})$`).MatchString

// ValidateDeviceName checks that DeviceName  is a valid DeviceName
func ValidateDeviceName(deviceName string) error {
	switch {
	case deviceName == "":
		return errors.New("device name is required")
	case isValidateDeviceName(deviceName):
		return nil
	default:
		return errors.New("invalid device name")
	}
}

var isValidateVmType = regexp.MustCompile(`^tinav([3-9]|[1-9][0-9]).c[1-9][0-9]*r[1-9][0-9]*p[1-3]$`).MatchString

// ValidateVmType checks that vmType is a valid vmType
func ValidateVmType(vmType string) error {
	switch {
	case isValidateVmType(vmType):
		return nil
	default:
		return errors.New("invalid vm type")
	}
}
