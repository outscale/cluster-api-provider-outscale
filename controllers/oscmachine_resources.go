package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

type MachineResourceTracker struct {
	Cloud services.Servicer
}

func (t *MachineResourceTracker) getVm(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*osc.Vm, error) {
	vm, id, err := t._getVmOrId(ctx, machineScope, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case vm != nil:
		return vm, nil
	}
	vm, err = t.Cloud.VM(ctx, *clusterScope).GetVm(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case vm == nil:
		return nil, fmt.Errorf("get vm %s: %w", id, ErrMissingResource)
	default:
		return vm, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *MachineResourceTracker) _getVmOrId(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*osc.Vm, string, error) {
	id := machineScope.GetVm().ResourceId
	if id != "" {
		return nil, id, nil
	}

	rsrc := machineScope.GetResources()
	id = getResource(defaultResource, rsrc.Vm)
	if id != "" {
		return nil, id, nil
	}
	clientToken := machineScope.GetClientToken(clusterScope)
	vm, err := t.Cloud.VM(ctx, *clusterScope).GetVmFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get vm from client token: %w", err)
	case vm != nil:
		t.setVmId(machineScope, vm.GetVmId())
		return vm, vm.GetVmId(), nil
	}
	// Search by name (retrocompatibility)
	name := machineScope.GetName() + "-" + clusterScope.GetUID()
	tg, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.VmResourceType, tag.NameKey, name)
	if err != nil {
		return nil, "", fmt.Errorf("get vm: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setVmId(machineScope, tg.GetResourceId())
		return nil, tg.GetResourceId(), nil
	}
	return nil, "", fmt.Errorf("get vm: %w", ErrNoResourceFound)
}

func (t *MachineResourceTracker) setVmId(machineScope *scope.MachineScope, id string) {
	rsrc := machineScope.GetResources()
	if rsrc.Vm == nil {
		rsrc.Vm = map[string]string{}
	}
	rsrc.Vm[defaultResource] = id
}

func (t *MachineResourceTracker) setVolumeIds(machineScope *scope.MachineScope, devices []osc.BlockDeviceMappingCreated) {
	rsrc := machineScope.GetResources()
	if rsrc.Volumes == nil {
		rsrc.Volumes = map[string]string{}
	}
	for _, device := range devices {
		rsrc.Volumes[device.GetDeviceName()] = device.Bsu.GetVolumeId()
	}
}

func (t *MachineResourceTracker) getImageId(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	rsrc := machineScope.GetResources()
	id := getResource(defaultResource, rsrc.Image)
	if id != "" {
		return id, nil
	}
	var image *osc.Image
	var err error
	imageSpec := machineScope.GetImage()
	if imageSpec.Name != "" {
		if imageSpec.AccountId == "" {
			ctrl.LoggerFrom(ctx).V(2).Info("[security] It is recommended to set the image account to control the origin of the image.")
		}
		image, err = t.Cloud.Image(ctx, *clusterScope).GetImageByName(ctx, imageSpec.Name, imageSpec.AccountId)
	} else {
		image, err = t.Cloud.Image(ctx, *clusterScope).GetImage(ctx, machineScope.GetImageId())
	}
	if err != nil {
		return "", fmt.Errorf("cannot get image: %w", err)
	}
	if image == nil {
		return "", errors.New("no image found")
	}
	t.setImageId(machineScope, image.GetImageId())
	return image.GetImageId(), nil
}

func (t *MachineResourceTracker) setImageId(machineScope *scope.MachineScope, imageId string) {
	rsrc := machineScope.GetResources()
	if rsrc.Image == nil {
		rsrc.Image = map[string]string{}
	}
	rsrc.Image[defaultResource] = imageId
}
