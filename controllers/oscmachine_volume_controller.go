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
	machineScope.Info("Check unique name volume")
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
	machineScope.Info("Check Volumes parameters")
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

		volumeIops := volumeSpec.Iops
		_, err = storage.ValidateIops(volumeIops)
		if err != nil {
			return volumeTagName, err
		}

		volumeSize := volumeSpec.Size
		_, err = storage.ValidateSize(volumeSize)
		if err != nil {
			return volumeTagName, err
		}

		volumeSubregionName := volumeSpec.SubregionName
		_, err = storage.ValidateSubregionName(volumeSubregionName)
		if err != nil {
			return volumeTagName, err
		}
		volumeType := volumeSpec.VolumeType
		_, err = storage.ValidateVolumeType(volumeType)
		if err != nil {
			return volumeTagName, err
		}
	}
	return "", nil
}

// reconcileVolume reconcile the volume of the machine
func reconcileVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface) (reconcile.Result, error) {
	machineScope.Info("Create Volume")

	var volumeId string
	var volumeIds []string
	var volumesSpec []*infrastructurev1beta1.OscVolume
	volumesSpec = machineScope.GetVolume()
	volumeRef := machineScope.GetVolumeRef()
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeIds = append(volumeIds, volumeId)
	}
	machineScope.Info("Check if the desired volumes exist")
	validVolumeIds, err := volumeSvc.ValidateVolumeIds(volumeIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineScope.Info("### Check Id ###", "volume", volumeIds)
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
				return reconcile.Result{}, fmt.Errorf("%w Can not create volume for OscCluster %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.Info("### Get volume ###", "volume", volume)
			volumeRef.ResourceMap[volumeName] = volume.GetVolumeId()
			volumeSpec.ResourceId = volume.GetVolumeId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVolume reconcile the destruction of the volume of the machine
func reconcileDeleteVolume(ctx context.Context, machineScope *scope.MachineScope, volumeSvc storage.OscVolumeInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine

	machineScope.Info("Delete Volume")
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
	machineScope.Info("### Check Id ###", "volume", volumeIds)
	for _, volumeSpec := range volumesSpec {
		volumeId = volumeSpec.ResourceId
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		if !Contains(validVolumeIds, volumeId) {
			controllerutil.RemoveFinalizer(oscmachine, "")
			return reconcile.Result{}, nil
		}
		machineScope.Info("Remove volume")
		machineScope.Info("Delete the desired volume", "volumeName", volumeName)
		err = volumeSvc.DeleteVolume(volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delelete volume for OscCluster %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
