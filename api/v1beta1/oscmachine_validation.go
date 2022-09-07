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

// ValidateOscMachineSpec validate each parameters of OscMachine spec.
func ValidateOscMachineSpec(spec OscMachineSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Node.VM.KeypairName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.VM.KeypairName, field.NewPath("keypairName"), ValidateKeypairName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.VM.ImageID != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.VM.ImageID, field.NewPath("imageId"), ValidateImageID); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.VM.DeviceName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.VM.DeviceName, field.NewPath("deviceName"), ValidateDeviceName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.VM.VMType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.VM.VMType, field.NewPath("vmType"), ValidateVMType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.VM.SubregionName != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.VM.SubregionName, field.NewPath("subregionName"), ValidateSubregionName); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if len(spec.Node.Volumes) != 0 {
		volumesSpec := spec.Node.Volumes
		for _, volumeSpec := range volumesSpec {
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
			if volumeSpec.SubregionName != "" {
				if errs := ValidateAndReturnErrorList(volumeSpec.SubregionName, field.NewPath("subregionName"), ValidateSubregionName); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
		}
	}
	return allErrs
}

// ValidateKeypairName check that KeypairName is a valid name of keypair.
func ValidateKeypairName(keypairName string) (string, error) {
	isValidateKeypairName := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateKeypairName(keypairName) {
		return keypairName, nil
	}
	return keypairName, errors.New("invalid KeypairName")
}

// ValidateImageID check that imageId is a valid imageId.
func ValidateImageID(imageID string) (string, error) {
	switch {
	case strings.HasPrefix(imageID, "ami"):
		return imageID, nil
	default:
		return imageID, errors.New("invalid imageId")
	}
}

// ValidateIops check that iops is valid.
func ValidateIops(iops int32) (int32, error) {
	if iops < maxIops && iops > minIops {
		return iops, nil
	}
	return iops, errors.New("invalid iops")
}

// ValidateSize check that size is valid.
func ValidateSize(size int32) (int32, error) {
	if size < maxSize && size > minSize {
		return size, nil
	}
	return size, errors.New("invalid size")
}

// ValidateVolumeType check that volumeType is a valid volumeType.
func ValidateVolumeType(volumeType string) (string, error) {
	switch volumeType {
	case "standard", "gp2", "io1":
		return volumeType, nil
	default:
		return volumeType, errors.New("invalid volumeType")
	}
}

// ValidateSubregionName check that subregionName is a valid az format.
func ValidateSubregionName(subregionName string) (string, error) {
	switch {
	case strings.HasSuffix(subregionName, "1a") || strings.HasSuffix(subregionName, "1b") || strings.HasSuffix(subregionName, "2a") || strings.HasSuffix(subregionName, "2b"):
		return subregionName, nil
	default:
		return subregionName, errors.New("invalid subregionName")
	}
}

// ValidateDeviceName check that DeviceName  is a valid DeviceName.
func ValidateDeviceName(deviceName string) (string, error) {
	isValidateDeviceName := regexp.MustCompile(`^(\/dev\/sda1|\/dev\/sd[a-z]{1}|\/dev\/xvd[a-z]{1})$`).MatchString
	switch {
	case isValidateDeviceName(deviceName):
		return deviceName, nil
	default:
		return deviceName, errors.New("invalid deviceName")
	}
}

// ValidateVMType check that vmType is a valid vmType.
func ValidateVMType(vmType string) (string, error) {
	isValidateVMType := regexp.MustCompile(`^tinav[1-5].c[0-9]+r[0-9]+p[1-3]$`).MatchString
	switch {
	case isValidateVMType(vmType):
		return vmType, nil
	default:
		return vmType, errors.New("invalid vmType")
	}
}
