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
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/utils"
)

// checkVolumeOscDuplicateName check that there are not the same name for volume resource
func checkVolumeOscDuplicateName(machineScope *scope.MachineScope) error {
	err := utils.CheckDuplicates(machineScope.GetVolume(), func(vol *infrastructurev1beta1.OscVolume) string {
		return vol.Name
	})
	if err != nil {
		return err
	}
	return utils.CheckDuplicates(machineScope.GetVolume(), func(vol *infrastructurev1beta1.OscVolume) string {
		return vol.Device
	})
}

// checkVolumeFormatParameters check Volume parameters format
func checkVolumeFormatParameters(machineScope *scope.MachineScope) (string, error) {
	for _, volumeSpec := range machineScope.GetVolume() {
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		volumeTagName, err := tag.ValidateTagNameValue(volumeName)
		if err != nil {
			return volumeTagName, err
		}

		if volumeSpec.Iops != 0 {
			volumeIops := volumeSpec.Iops
			err = infrastructurev1beta1.ValidateIops(volumeIops)
			if err != nil {
				return volumeTagName, err
			}
		}

		volumeSize := volumeSpec.Size
		err = infrastructurev1beta1.ValidateSize(volumeSize)
		if err != nil {
			return volumeTagName, err
		}

		volumeType := volumeSpec.VolumeType
		err = infrastructurev1beta1.ValidateVolumeType(volumeType)
		if err != nil {
			return volumeTagName, err
		}
	}
	return "", nil
}
