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

// ValidateOscMachineSpec validate each parameters of OscMachine spec
func ValidateOscMachineSpec(spec OscMachineSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Node.Vm.KeypairName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.KeypairName, field.NewPath("keypairName"), ValidateKeypairName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.DeviceName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.DeviceName, field.NewPath("deviceName"), ValidateDeviceName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.VmType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.VmType, field.NewPath("vmType"), ValidateVmType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.SubregionName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.SubregionName, field.NewPath("subregionName"), ValidateSubregionName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if len(spec.Node.Volumes) != 0 {
		volumesSpec := spec.Node.Volumes
		for _, volumeSpec := range volumesSpec {
			if err := ValidateDeviceName(volumeSpec.Device); err != nil {
				allErrs = append(allErrs, field.Invalid(field.NewPath("node", "volumes", "device"), volumeSpec.Device, err.Error()))
			}

			if volumeSpec.Iops != 0 {
				if errs := ValidateAndReturnErrorList(volumeSpec.Iops, field.NewPath("iops"), ValidateIops); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volumeSpec.Size != 0 {
				if errs := ValidateAndReturnErrorList(volumeSpec.Size, field.NewPath("size"), ValidateSize); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volumeSpec.VolumeType != "" {
				if errs := ValidateAndReturnErrorList(volumeSpec.VolumeType, field.NewPath("volumeType"), ValidateVolumeType); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
			if volumeSpec.Iops != 0 && volumeSpec.Size != 0 && volumeSpec.VolumeType == "io1" {
				ratioIopsSize := volumeSpec.Iops / volumeSpec.Size
				if errs := ValidateAndReturnErrorList(ratioIopsSize, field.NewPath("size"), ValidateRatioSizeIops); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskIops != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskIops, field.NewPath("iops"), ValidateIops); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskSize != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskSize, field.NewPath("node", "volumes", "size"), ValidateSize); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskType, field.NewPath("diskType"), ValidateVolumeType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskIops != 0 && spec.Node.Vm.RootDisk.RootDiskSize != 0 && spec.Node.Vm.RootDisk.RootDiskType == "io1" {
		ratioIopsSize := spec.Node.Vm.RootDisk.RootDiskIops / spec.Node.Vm.RootDisk.RootDiskSize
		if errs := ValidateAndReturnErrorList(ratioIopsSize, field.NewPath("size"), ValidateRatioSizeIops); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

// ValidateKeypairName check that KeypairName is a valid name of keypair
func ValidateKeypairName(keypairName string) error {
	isValidateKeypairName := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateKeypairName(keypairName) {
		return nil
	} else {
		return errors.New("Invalid KeypairName")
	}
}

// ValidateImageId check that imageId is a valid imageId
func ValidateImageId(imageId string) error {
	switch {
	case strings.HasPrefix(imageId, "ami"):
		return nil
	default:
		return errors.New("Invalid imageId")
	}
}

// ValidateRatioSizeIops check that Ratio iops size should not exceed 300
func ValidateRatioSizeIops(ratioIopsSize int32) error {
	if ratioIopsSize < 300 {
		return nil
	} else {
		return errors.New("Invalid ratio Iops size that exceed 300")
	}

}

// ValidateIamegName check that Image name is a valide name
func ValidateImageName(imageName string) error {
	isValidateName := regexp.MustCompile(`^[0-9A-Za-z\-_\s\.\(\)\\]{0,255}$`).MatchString
	if isValidateName(imageName) {
		return nil
	} else {
		return errors.New("Invalid Image Name")
	}
}

// ValidateIops check that iops is valid
func ValidateIops(iops int32) error {
	if iops < maxIops && iops > minIops {
		return nil
	} else {
		return errors.New("invalid iops")
	}
}

// ValidateSize check that size is valid
func ValidateSize(size int32) error {
	if size < maxSize && size > minSize {
		return nil
	} else {
		return errors.New("invalid size")
	}
}

// ValidateVolumeType check that volumeType is a valid volumeType
func ValidateVolumeType(volumeType string) error {
	switch volumeType {
	case "standard", "gp2", "io1":
		return nil
	default:
		return errors.New("Invalid volumeType")
	}
}

// ValidateSubregionName check that subregionName is a valid az format
func ValidateSubregionName(subregionName string) error {
	switch {
	case strings.HasSuffix(subregionName, "1a") || strings.HasSuffix(subregionName, "1b") || strings.HasSuffix(subregionName, "1c") || strings.HasSuffix(subregionName, "2a") || strings.HasSuffix(subregionName, "2b") || strings.HasSuffix(subregionName, "2c"):
		return nil
	default:
		return errors.New("Invalid subregionName")
	}
}

var isValidateDeviceName = regexp.MustCompile(`^(\/dev\/sda1|\/dev\/sd[a-z]{1}|\/dev\/xvd[a-z]{1})$`).MatchString

// ValidateDeviceName check that DeviceName  is a valid DeviceName
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

// ValidateVmType check that vmType is a valid vmType
func ValidateVmType(vmType string) error {
	isValidateVmType := regexp.MustCompile(`^tinav[3-6].c[0-9]+r[0-9]+p[1-3]$`).MatchString
	switch {
	case isValidateVmType(vmType):
		return nil
	default:
		return errors.New("Invalid vmType")
	}
}
