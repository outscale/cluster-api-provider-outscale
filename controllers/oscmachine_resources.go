package controllers

import (
	"context"
	"fmt"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
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

func (t *MachineResourceTracker) getVmId(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	vm, id, err := t._getVmOrId(ctx, machineScope, clusterScope)
	switch {
	case err != nil:
		return "", err
	case vm != nil:
		return vm.GetVmId(), nil
	default:
		return id, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *MachineResourceTracker) _getVmOrId(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (*osc.Vm, string, error) {
	id := machineScope.GetVm().ResourceId
	if id != "" {
		return nil, id, nil
	}

	clientToken := machineScope.GetName() + "-" + clusterScope.GetUID()
	rsrc := machineScope.GetResources()
	id = getResource(clientToken, rsrc.Vm)
	if id != "" {
		return nil, id, nil
	}
	vm, err := t.Cloud.VM(ctx, *clusterScope).GetVmFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get vm from client token: %w", err)
	case vm != nil:
		t.setVmId(machineScope, vm.GetVmId())
		return vm, vm.GetVmId(), nil
	}
	// Search by name (retrocompatibility)
	tg, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.NetResourceType, tag.NameKey, clientToken)
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
