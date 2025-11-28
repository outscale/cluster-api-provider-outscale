/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	minIops = 0
	maxIops = 13000
	minSize = 1
	maxSize = 14900
)

// ValidateOscMachineSpec validates a OscMachineSpec.
func ValidateOscMachineSpec(spec OscMachineSpec) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = AppendValidation(allErrs, ValidateRequired(field.NewPath("node", "vm", "keypairName"), spec.Node.Vm.KeypairName, "keypairName is required"))
	allErrs = AppendValidation(allErrs, ValidateVmType(field.NewPath("node", "vm", "vmType"), spec.Node.Vm.VmType))
	allErrs = AppendValidation(allErrs, ValidateSubregion(field.NewPath("node", "vm", "subregionName"), spec.Node.Vm.SubregionName))

	for _, spec := range spec.Node.Volumes {
		allErrs = AppendValidation(allErrs, ValidateVolume(field.NewPath("node", "vm", "subregionName"), spec)...)
	}
	allErrs = AppendValidation(allErrs, ValidateIops(field.NewPath("node", "vm", "rootDisk", "rootDiskIops"), spec.Node.Vm.RootDisk.RootDiskIops, spec.Node.Vm.RootDisk.RootDiskSize))
	allErrs = AppendValidation(allErrs, ValidateSize(field.NewPath("node", "vm", "rootDisk", "rootDiskSize"), spec.Node.Vm.RootDisk.RootDiskSize))
	allErrs = AppendValidation(allErrs, ValidateVolumeType(field.NewPath("node", "vm", "rootDisk", "rootDiskType"), spec.Node.Vm.RootDisk.RootDiskType))
	return allErrs
}

func ValidateVolume(path *field.Path, spec OscVolume) field.ErrorList {
	var allErrs field.ErrorList
	return AppendValidation(allErrs,
		ValidateDeviceName(path.Child("device"), spec.Device),
		ValidateIops(path.Child("iops"), spec.Iops, spec.Size),
		ValidateSize(path.Child("size"), spec.Size),
		ValidateVolumeType(path.Child("volumeType"), spec.VolumeType),
	)
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
func ValidateIops(path *field.Path, iops, size int32) *field.Error {
	switch {
	case iops == 0:
		return nil
	case iops > maxIops || iops < minIops:
		return field.Invalid(path, iops, fmt.Sprintf("iops must be between %d and %d", minIops, maxIops))
	case size > 0 && iops/size > 300:
		return field.Invalid(path, iops, "iops/size should be lower than 300")
	default:
		return nil
	}
}

// ValidateSize checks that size is valid
func ValidateSize(path *field.Path, size int32) *field.Error {
	switch {
	case size == 0:
		return nil
	case size > maxSize || size < minSize:
		return field.Invalid(path, size, fmt.Sprintf("size must be between %d and %d", minSize, maxSize))
	default:
		return nil
	}
}

// ValidateVolumeType checks that volumeType is a valid volumeType
func ValidateVolumeType(path *field.Path, volumeType string) *field.Error {
	switch volumeType {
	case "", "standard", "gp2", "io1":
		return nil
	default:
		return field.Invalid(path, volumeType, "invalid volume type (allowed: standard, gp2, io1)")
	}
}

var isValidateDeviceName = regexp.MustCompile(`^(\/dev\/sda1|\/dev\/sd[a-z]{1}|\/dev\/xvd[a-z]{1})$`).MatchString

// ValidateDeviceName checks that DeviceName  is a valid DeviceName
func ValidateDeviceName(path *field.Path, deviceName string) *field.Error {
	switch {
	case deviceName == "":
		return field.Required(path, "device name is required")
	case isValidateDeviceName(deviceName):
		return nil
	default:
		return field.Invalid(path, deviceName, "device must use the /dev/(s|xv)d[a-z] format")
	}
}

var isValidTinaVmType = regexp.MustCompile(`^tinav([3-9]|[1-9][0-9]).c[1-9][0-9]*r[1-9][0-9]*p[1-3]$`).MatchString
var isValidInferenceVmType = regexp.MustCompile(`^inference7-(?:l40\.(?:medium|large)|h100\.(?:medium|large|xlarge|2xlarge)|h200\.(?:2xsmall|2xmedium|2xlarge|4xlarge|4xlargeA))$`).MatchString

// ValidateVmType checks that vmType is a valid vmType
func ValidateVmType(path *field.Path, vmType string) *field.Error {
	switch {
	case vmType == "":
		return field.Required(path, "vmType is required")
	case isValidTinaVmType(vmType):
		return nil
	case isValidInferenceVmType(vmType):
		return nil
	default:
		return field.Invalid(path, vmType, "vmType must use either the tinavX.cXrXpX or inferenceX-{gpu}.{size} format")
	}
}
