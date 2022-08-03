package v1beta1

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
        "regexp"
        "errors"
	"strings"
)

const (
	minIops = 0
	maxIops = 13000
	minSize = 0
	maxSize = 14901
)
func ValidateOscMachineSpec(spec OscMachineSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.Node.Vm.KeypairName != "" {
		if errs := ValidateVmKeypairName(spec.Node.Vm.KeypairName, field.NewPath("keypairName")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.ImageId != "" {
		if errs := ValidateVmImageId(spec.Node.Vm.ImageId, field.NewPath("imageId")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.DeviceName != "" {
		if errs := ValidateVmDeviceName(spec.Node.Vm.DeviceName, field.NewPath("deviceName")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.VmType != "" {
		if errs := ValidateVmVmType(spec.Node.Vm.VmType, field.NewPath("vmType")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	if spec.Node.Vm.SubregionName != "" {
		if errs := ValidateNetworkSubregionName(spec.Node.Vm.SubregionName, field.NewPath("subregionName")); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if len(spec.Node.Volumes) != 0 {
		volumesSpec := spec.Node.Volumes
		for _, volumeSpec := range volumesSpec{
			if volumeSpec.Iops != 0 {
				if errs := ValidateVolumeIops(volumeSpec.Iops, field.NewPath("iops")); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volumeSpec.Size != 0 {
				if errs := ValidateVolumeSize(volumeSpec.Size, field.NewPath("size")); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}

			if volumeSpec.VolumeType != "" {
				if errs := ValidateVolumeVolumeType(volumeSpec.VolumeType, field.NewPath("volumeType")); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
			if volumeSpec.SubregionName != "" {			
				if errs := ValidateNetworkSubregionName(volumeSpec.SubregionName, field.NewPath("subregionName")); len(errs) > 0 {
					allErrs = append(allErrs, errs...)
				}
			}
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
        case strings.Contains(imageId, "ami"):
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

func ValidateNetworkSubregionName(networkSubregionName string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateSubregionName(networkSubregionName)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, networkSubregionName, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVolumeIops(volumeIops int32, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateIops(volumeIops)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, volumeIops, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVolumeSize(volumeSize int32, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateSize(volumeSize)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, volumeSize, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVmKeypairName(vmKeypairName string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateKeypairName(vmKeypairName)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, vmKeypairName, err.Error()))
		return allErrs
	}
	return allErrs	
}

func ValidateVmImageId(vmImageId string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateImageId(vmImageId)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, vmImageId, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVmDeviceName(vmDeviceName string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateDeviceName(vmDeviceName)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, vmDeviceName, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVmVmType(vmType string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	_, err := ValidateVmType(vmType)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, vmType, err.Error()))
		return allErrs
	}
	return allErrs
}

func ValidateVolumeVolumeType(volumeType string, fieldPath *field.Path) field.ErrorList{
	allErrs := field.ErrorList{}
	_, err := ValidateVolumeType(volumeType)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, volumeType, err.Error()))
		return allErrs
	}
	return allErrs
}

// ValidateDeviceName check that DeviceName  is a valid DeviceName
func ValidateDeviceName(deviceName string) (string, error) {
	last := deviceName[len(deviceName)-1:]
	isValidateDeviceName := regexp.MustCompile(`^[a-z]$`).MatchString
	switch {
	case strings.Contains(deviceName, "/dev/xvd") && len(deviceName) == 9 && isValidateDeviceName(last):
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

