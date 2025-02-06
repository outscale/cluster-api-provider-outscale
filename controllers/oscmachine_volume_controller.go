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
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/storage"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func reconcileVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	var volumeId string
	var volumeIds []string
	volumesSpec := machineScope.GetVolume()
	volumeRef := machineScope.GetVolumeRef()
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeIds = append(volumeIds, volumeId)
	}
	validVolumeIds, err := volumeSvc.ValidateVolumeIds(ctx, volumeIds)
	log.V(4).Info("Check Id", "volume", validVolumeIds)

	if err != nil {
		return reconcile.Result{}, err
	}
	for _, volumeSpec := range volumesSpec {
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		if len(volumeRef.ResourceMap) == 0 {
			volumeRef.ResourceMap = make(map[string]string)
		}
		if volumeSpec.ResourceId != "" {
			volumeRef.ResourceMap[volumeName] = volumeSpec.ResourceId
		}
		tagKey := "Name"
		tagValue := volumeName
		tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
		}
		volumeId := volumeRef.ResourceMap[volumeName]
		log.V(2).Info("Check if volumes exist")
		if !slices.Contains(validVolumeIds, volumeId) && tag == nil {
			volume, err := volumeSvc.CreateVolume(ctx, volumeSpec, volumeName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create volume: %w", err)
			}
			volumeId := volume.GetVolumeId()
			log.V(4).Info("Get VolumeId", "volumeId", volumeId)
			if volumeId != "" {
				err = volumeSvc.CheckVolumeState(ctx, 5, 60, "available", volumeId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot get volume available: %w", err)
				}
				log.V(4).Info("Volume is available", "volumeId", volumeId)
			}
			log.V(4).Info("Get volume", "volume", volume)
			volumeRef.ResourceMap[volumeName] = volume.GetVolumeId()
			volumeSpec.ResourceId = volume.GetVolumeId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVolume reconcile the destruction of the volume of the machine
func reconcileDeleteVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	oscmachine := machineScope.OscMachine

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
	validVolumeIds, err := volumeSvc.ValidateVolumeIds(ctx, volumeIds)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot validate volume id: %w", err)
	}
	log.V(4).Info("Number of volume", "volumeLength", len(volumesSpec))
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		if !slices.Contains(validVolumeIds, volumeId) {
			controllerutil.RemoveFinalizer(oscmachine, "")
			return reconcile.Result{}, nil
		}
		err = volumeSvc.CheckVolumeState(ctx, 5, 60, "in-use", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get volume %s in use: %w", volumeId, err)
		}
		log.V(4).Info("Volume is in use", "volumeId", volumeId)

		log.V(2).Info("Unlinking volume", "volumeId", volumeId)
		err = volumeSvc.UnlinkVolume(ctx, volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot unlink volume %s in use: %w", volumeId, err)
		}

		err = volumeSvc.CheckVolumeState(ctx, 5, 60, "available", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get volume %s available: %w", volumeId, err)
		}
		log.V(4).Info("Volume is available", "volumeId", volumeId)
		log.V(2).Info("Deleting volume", "volumeName", volumeName)
		err = volumeSvc.DeleteVolume(ctx, volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete volume %s: %w", volumeId, err)
		}
	}
	return reconcile.Result{}, nil
}
