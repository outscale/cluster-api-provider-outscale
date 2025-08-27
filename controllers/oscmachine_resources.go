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

// OutscaleOpenSourceAccounts lists the accounts used to publish open source image accounts.
var OutscaleOpenSourceAccounts = map[string]string{
	"eu-west-2":           "671899555720",
	"us-east-2":           "852047997530",
	"cloudgouv-eu-west-1": "545146734248",
}

type MachineResourceTracker struct {
	Cloud services.Servicer
}

func (t *MachineResourceTracker) getVm(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*osc.Vm, error) {
	vm, id, err := t._getVmOrId(ctx, machineScope, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case vm != nil:
		t.trackVm(machineScope, vm)
		err := t.IPAllocator(machineScope).RetrackIP(ctx, defaultResource, vm.GetPublicIp(), clusterScope)
		return vm, err
	}
	vm, err = t.Cloud.VM(ctx, *clusterScope).GetVm(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case vm == nil:
		return nil, fmt.Errorf("get vm %s: %w", id, ErrMissingResource)
	default:
		t.trackVm(machineScope, vm)
		err := t.IPAllocator(machineScope).RetrackIP(ctx, defaultResource, vm.GetPublicIp(), clusterScope)
		return vm, err
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
		return vm, vm.GetVmId(), nil
	}
	// Search by name (retrocompatibility)
	name := machineScope.GetName() + "-" + clusterScope.GetUID()
	tg, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.VmResourceType, tag.NameKey, name)
	if err != nil {
		return nil, "", fmt.Errorf("get vm: %w", err)
	}
	if tg.GetResourceId() != "" {
		return nil, tg.GetResourceId(), nil
	}
	return nil, "", fmt.Errorf("get vm: %w", ErrNoResourceFound)
}

func (t *MachineResourceTracker) trackVm(machineScope *scope.MachineScope, vm *osc.Vm) {
	t.setVmId(machineScope, vm.GetVmId())
	t.setVolumeIds(machineScope, vm.GetBlockDeviceMappings())
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
		var accountId string
		switch {
		case imageSpec.OutscaleOpenSource:
			accountId = OutscaleOpenSourceAccounts[t.Cloud.OscClient().Region]
		case imageSpec.AccountId == "":
			ctrl.LoggerFrom(ctx).V(2).Info("[security] It is recommended to set the image account to control the origin of the image.")
		default:
			accountId = imageSpec.AccountId
		}
		image, err = t.Cloud.Image(ctx, *clusterScope).GetImageByName(ctx, imageSpec.Name, accountId)
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

func (t *MachineResourceTracker) IPAllocator(machineScope *scope.MachineScope) IPAllocatorInterface {
	return &IPAllocator{
		Cloud: t.Cloud,
		getPublicIP: func(key string) (id string, found bool) {
			rsrc := machineScope.GetResources()
			if rsrc.PublicIPs == nil {
				return "", false
			}
			ip := rsrc.PublicIPs[key]
			return ip, ip != ""
		},
		setPublicIP: func(key, id string) {
			rsrc := machineScope.GetResources()
			if rsrc.PublicIPs == nil {
				rsrc.PublicIPs = map[string]string{}
			}
			rsrc.PublicIPs[key] = id
		},
	}
}

func (t *MachineResourceTracker) getPublicIps(machineScope *scope.MachineScope) map[string]string {
	rsrc := machineScope.GetResources()
	return rsrc.PublicIPs
}

func (t *MachineResourceTracker) untrackIP(machineScope *scope.MachineScope, name string) {
	rsrc := machineScope.GetResources()
	if rsrc.PublicIPs == nil {
		return
	}
	delete(rsrc.PublicIPs, name)
}
