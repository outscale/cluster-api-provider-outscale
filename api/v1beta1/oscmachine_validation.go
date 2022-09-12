package v1beta1

import (
	"errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"regexp"
	"strings"
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
	if spec.Node.Vm.ImageId != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.ImageId, field.NewPath("imageId"), ValidateImageId); len(errs) > 0 {
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
	if spec.Node.Vm.RootDisk.RootDiskIops != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskIops, field.NewPath("iops"), ValidateIops); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskSize != 0 {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskSize, field.NewPath("size"), ValidateSize); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.RootDisk.RootDiskType != "" {
		if errs := ValidateAndReturnErrorList(spec.Node.Vm.RootDisk.RootDiskType, field.NewPath("diskType"), ValidateVolumeType); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

// ValidateKeypairName check that KeypairName is a valid name of keypair
func ValidateKeypairName(keypairName string) (string, error) {
	isValidateKeypairName := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateKeypairName(keypairName) {
		return keypairName, nil
	} else {
		return keypairName, errors.New("Invalid KeypairName")
	}
}

// ValidateImageId check that imageId is a valid imageId
func ValidateImageId(imageId string) (string, error) {
	switch {
	case strings.HasPrefix(imageId, "ami"):
		return imageId, nil
	default:
		return imageId, errors.New("Invalid imageId")
	}
}

// ValidateIops check that iops is valid
func ValidateIops(iops int32) (int32, error) {
	if iops < maxIops && iops > minIops {
		return iops, nil
	} else {
		return iops, errors.New("Invalid iops")
	}
}

// ValidateSize check that size is valid
func ValidateSize(size int32) (int32, error) {
	if size < maxSize && size > minSize {
		return size, nil
	} else {
		return size, errors.New("Invalid size")
	}
}

// ValidateVolumeType check that volumeType is a valid volumeType
func ValidateVolumeType(volumeType string) (string, error) {
	switch volumeType {
	case "standard", "gp2", "io1":
		return volumeType, nil
	default:
		return volumeType, errors.New("Invalid volumeType")
	}
}

// ValidateSubregionName check that subregionName is a valid az format
func ValidateSubregionName(subregionName string) (string, error) {
	switch {
	case strings.HasSuffix(subregionName, "1a") || strings.HasSuffix(subregionName, "1b") || strings.HasSuffix(subregionName, "2a") || strings.HasSuffix(subregionName, "2b"):
		return subregionName, nil
	default:
		return subregionName, errors.New("Invalid subregionName")
	}
}

// ValidateDeviceName check that DeviceName  is a valid DeviceName
func ValidateDeviceName(deviceName string) (string, error) {
	last := deviceName[len(deviceName)-1:]
	isValidateDeviceName := regexp.MustCompile(`^[0-9a-z]$`).MatchString
	switch {
	case (strings.Contains(deviceName, "/dev/xvd") || strings.Contains(deviceName, "/dev/sda")) && len(deviceName) == 9 && isValidateDeviceName(last):
		return deviceName, nil
	default:
		return deviceName, errors.New("Invalid deviceName")
	}
}

// ValidateVmType check that vmType is a valid vmType
func ValidateVmType(vmType string) (string, error) {
	isValidateVmType := regexp.MustCompile(`^tinav[1-5].c[0-9]+r[0-9]+p[1-3]$`).MatchString
	switch {
	case isValidateVmType(vmType):
		return vmType, nil
	default:
		return vmType, errors.New("Invalid vmType")
	}
}
