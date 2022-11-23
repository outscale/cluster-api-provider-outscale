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
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getVolumeResourceId return the volumeId from the resourceMap base on resourceName (tag name + cluster uid)
func getVolumeResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	volumeRef := machineScope.GetVolumeRef()
	if volumeId, ok := volumeRef.ResourceMap[resourceName]; ok {
		return volumeId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkVolumeOscDuplicateName check that there are not the same name for volume resource
func checkVolumeOscDuplicateName(machineScope *scope.MachineScope) error {
	machineScope.V(2).Info("Check unique name volume")
	var resourceNameList []string
	volumesSpec := machineScope.GetVolume()
	for _, volumeSpec := range volumesSpec {
		resourceNameList = append(resourceNameList, volumeSpec.Name)
	}
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// checkVolumeFormatParameters check Volume parameters format
func checkVolumeFormatParameters(machineScope *scope.MachineScope) (string, error) {
	machineScope.V(2).Info("Check Volumes parameters")
	var volumesSpec []*infrastructurev1beta1.OscVolume
	nodeSpec := machineScope.GetNode()
	if nodeSpec.Volumes == nil {
		nodeSpec.SetVolumeDefaultValue()
		volumesSpec = nodeSpec.Volumes
	} else {
		volumesSpec = machineScope.GetVolume()
	}
	for _, volumeSpec := range volumesSpec {

		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		volumeTagName, err := tag.ValidateTagNameValue(volumeName)
		if err != nil {
			return volumeTagName, err
		}

		if volumeSpec.Iops != 0 {
			volumeIops := volumeSpec.Iops
			_, err = infrastructurev1beta1.ValidateIops(volumeIops)
			if err != nil {
				return volumeTagName, err
			}
		}

		volumeSize := volumeSpec.Size
		machineScope.V(4).Info("Check volume info", "volumeSize", volumeSize)
		_, err = infrastructurev1beta1.ValidateSize(volumeSize)
		if err != nil {
			return volumeTagName, err
		}

		volumeSubregionName := volumeSpec.SubregionName
		_, err = infrastructurev1beta1.ValidateSubregionName(volumeSubregionName)
		if err != nil {
			return volumeTagName, err
		}
		volumeType := volumeSpec.VolumeType
		_, err = infrastructurev1beta1.ValidateVolumeType(volumeType)
		if err != nil {
			return volumeTagName, err
		}
	}
	return "", nil
}

// reconcileVolume reconcile the volume of the machine
func reconcileVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface) (reconcile.Result, error) {
	machineScope.V(2).Info("Create Volume")

	var volumeId string
	var volumeIds []string
	var volumesSpec []*infrastructurev1beta1.OscVolume
	volumesSpec = machineScope.GetVolume()
	volumeRef := machineScope.GetVolumeRef()
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeIds = append(volumeIds, volumeId)
	}

	machineScope.V(2).Info("Check if the desired volumes exist")
	validVolumeIds, err := volumeSvc.ValidateVolumeIds(volumeIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineScope.V(4).Info("### Check Id ###", "volume", volumeIds)
	for _, volumeSpec := range volumesSpec {
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		if len(volumeRef.ResourceMap) == 0 {
			volumeRef.ResourceMap = make(map[string]string)
		}
		if volumeSpec.ResourceId != "" {
			volumeRef.ResourceMap[volumeName] = volumeSpec.ResourceId
		}
		volumeId := volumeRef.ResourceMap[volumeName]
		if !Contains(validVolumeIds, volumeId) {
			volume, err := volumeSvc.CreateVolume(volumeSpec, volumeName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create volume for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
			}
			volumeId := volume.GetVolumeId()
			machineScope.V(4).Info("### Get VolumeId ###", "volumeId", volumeId)
			if volumeId != "" {
				err = volumeSvc.CheckVolumeState(5, 60, "available", volumeId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not get volume available for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
				}
				machineScope.V(4).Info("Volume is available", "volumeId", volumeId)
			}
			machineScope.V(4).Info("### Get volume ###", "volume", volume)
			volumeRef.ResourceMap[volumeName] = volume.GetVolumeId()
			volumeSpec.ResourceId = volume.GetVolumeId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVolume reconcile the destruction of the volume of the machine
func reconcileDeleteVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine

	machineScope.V(2).Info("Delete Volume")
	var volumesSpec []*infrastructurev1beta1.OscVolume
	nodeSpec := machineScope.GetNode()
	if nodeSpec.Volumes == nil {
		nodeSpec.SetVolumeDefaultValue()
		volumesSpec = nodeSpec.Volumes
	} else {
		volumesSpec = machineScope.GetVolume()
	}

	var volumeIds []string
	var volumeId string
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeIds = append(volumeIds, volumeId)
	}
	validVolumeIds, err := volumeSvc.ValidateVolumeIds(volumeIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineScope.V(4).Info("### Check Id ###", "volume", volumeIds)
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		if !Contains(validVolumeIds, volumeId) {
			controllerutil.RemoveFinalizer(oscmachine, "")
			return reconcile.Result{}, nil
		}
		err = volumeSvc.CheckVolumeState(5, 60, "in-use", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s in use for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.V(4).Info("Volume is in use", "volumeId", volumeId)

		err = volumeSvc.UnlinkVolume(volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink volume %s in use for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.V(4).Info("Volume is unlinked", "volumeId", volumeId)

		err = volumeSvc.CheckVolumeState(5, 60, "available", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s available for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.V(4).Info("Volume is available", "volumeId", volumeId)
		machineScope.V(2).Info("Remove volume")
		machineScope.V(4).Info("Delete the desired volume", "volumeName", volumeName)
		err = volumeSvc.DeleteVolume(volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete volume for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
