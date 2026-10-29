/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
)

// reconcileVm reconcile the vm of the machine
func (r *OscMachineReconciler) reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) Result {
	log := ctrl.LoggerFrom(ctx)
	if !machineScope.NeedReconciliation(infrastructurev1beta2.ReconcilerVm) && machineScope.IsReady() {
		log.V(4).Info("No need for vm reconciliation")
		return Result{Status: ResultStatusOK}
	}

	vmSpec := machineScope.GetVm()
	role := machineScope.GetSpec().Role

	var fgpu *osc.FlexibleGpu
	if vmSpec.FGPU != nil {
		var err error
		fgpu, err = r.Tracker.getFGPU(ctx, machineScope, clusterScope)
		if err != nil && !IsNotFound(err) {
			return Result{Status: ResultStatusUnknown, Error: fmt.Errorf("get fgpu: %w", err)}
		}
	}
	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case err == nil:
		machineScope.SetVmState(vm.State)
	case !IsNotFound(err):
		return Result{Status: ResultStatusUnknown, Error: fmt.Errorf("get existing: %w", err)}
	default:
		// Check if a machine needs to be placed in a subregion.
		var azs []infrastructurev1beta2.OscSubRegion
		switch {
		case fgpu != nil:
			azs = []infrastructurev1beta2.OscSubRegion{infrastructurev1beta2.OscSubRegion(fgpu.SubregionName)}
		case machineScope.Machine.Spec.FailureDomain != nil:
			azs = []infrastructurev1beta2.OscSubRegion{infrastructurev1beta2.OscSubRegion(*machineScope.Machine.Spec.FailureDomain)}
		default:
			azs = vmSpec.GetSubregions()
			if len(azs) == 0 {
				azs = clusterScope.GetSubregions()
				log.V(4).Info("Using cluster subregions")
			}
		}
		netSpec, subnetSpec, subregionName, err := r.AZAllocator.AllocateAZ(ctx, machineScope.OscMachine, vmSpec.AllocationMode, role, clusterScope.GetNets(), azs)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("allocate az: %w", err)}
		}

		// Allocate a FGPU
		// If the subregion has no FGPU capacity, an error is returned. Let the next loop try in another subregion.
		switch {
		case vmSpec.FGPU == nil:
		// no fGPU configured
		case machineScope.IsControlPlane():
			log.V(2).Info("Control-plane nodes are not allowed to have fGPUs")
		case fgpu == nil:
			log.V(3).Info("Allocating fGPU", "model", vmSpec.FGPU.Model)
			fgpu, err = r.Cloud.Compute(clusterScope.Tenant).AllocateFGPU(ctx, vmSpec.FGPU.Model, string(subregionName), machineScope)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("fgpu: %w", err)}
			}
			log.V(2).Info("fGPU allocated", "fGPUId", fgpu.FlexibleGpuId, "model", vmSpec.FGPU.Model)
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.FGPUAllocatedReason, "%s (%s) allocated", fgpu.FlexibleGpuId, vmSpec.FGPU.Model)
		}

		subnet, err := r.ClusterTracker.getSubnet(ctx, subnetSpec, clusterScope)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("find subnet: %w", err)}
		}
		subnetId := subnet.SubnetId
		annotations.AddAnnotations(machineScope.OscMachine, map[string]string{
			infrastructurev1beta2.NetAnnotation:       subnet.NetId,
			infrastructurev1beta2.SubnetAnnotation:    subnetId,
			infrastructurev1beta2.SubregionAnnotation: subnet.SubregionName,
			infrastructurev1beta2.NetRangeAnnotation:  netSpec.IpRange,
		})
		securityGroups, err := clusterScope.GetSecurityGroupsFor(machineScope.GetVmSecurityGroups(), role)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot find securityGroup: %w", err)}
		}
		securityGroupIds := make([]string, 0, len(securityGroups))
		for _, sgSpec := range securityGroups {
			securityGroupId, err := r.ClusterTracker.getSecurityGroupId(ctx, netSpec.IpRange, sgSpec, clusterScope)
			log.V(4).Info("Found securityGroup", "securityGroupId", securityGroupId)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("get sg: %w", err)}
			}
			securityGroupIds = append(securityGroupIds, securityGroupId)
		}
		vmPrivateIps := machineScope.GetVmPrivateIps()
		privateIps := make([]string, 0, len(vmPrivateIps))
		for _, vmPrivateIp := range vmPrivateIps {
			privateIp := vmPrivateIp.PrivateIp
			privateIps = append(privateIps, privateIp)
		}
		imageId, err := r.Tracker.getImageId(ctx, machineScope, clusterScope)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("get image: %w", err)}
		}
		vmName := machineScope.GetName()
		vmTags := vmSpec.Tags
		if vmTags == nil {
			vmTags = map[string]string{}
		} else {
			// we need to clone the map to avoid changing the spec...
			vmTags = maps.Clone(vmTags)
		}

		if vmSpec.PublicIp {
			_, publicIp, err := r.Tracker.IPAllocator(machineScope).AllocateIP(ctx, defaultResource, vmName, vmSpec.PublicIpPool, clusterScope)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("allocate IP: %w", err)}
			}
			vmTags[compute.AutoAttachExternalIPTag] = publicIp
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.IPAllocated, "IP %s allocated", publicIp)
		}

		repulse := machineScope.GetPlacement()
		switch {
		case repulse.RepulseCluster != "" && repulse.ClusterStrict:
			vmTags[compute.RepulseClusterStrictTag] = repulse.RepulseCluster
		case repulse.RepulseCluster != "" && !repulse.ClusterStrict:
			vmTags[compute.RepulseClusterTag] = repulse.RepulseCluster

		case repulse.AttractCluster != "" && repulse.ClusterStrict:
			vmTags[compute.AttractClusterStrictTag] = repulse.AttractCluster
		case repulse.AttractCluster != "" && !repulse.ClusterStrict:
			vmTags[compute.AttractClusterTag] = repulse.AttractCluster
		}
		switch {
		case repulse.RepulseServer != nil && *repulse.RepulseServer != "" && repulse.ServerStrict:
			vmTags[compute.RepulseServerStrictTag] = *repulse.RepulseServer
		case repulse.RepulseServer != nil && *repulse.RepulseServer != "" && !repulse.ServerStrict:
			vmTags[compute.RepulseServerTag] = *repulse.RepulseServer

		case repulse.AttractServer != "" && repulse.ServerStrict:
			vmTags[compute.AttractServerStrictTag] = repulse.AttractServer
		case repulse.AttractServer != "" && !repulse.ServerStrict:
			vmTags[compute.AttractServerTag] = repulse.AttractServer
		}

		keypairName := vmSpec.Keypair
		vmType := vmSpec.Type
		volumes := machineScope.GetVolumes()
		clientToken := machineScope.GetClientToken(clusterScope)
		log.V(3).Info("Creating VM", "vmName", vmName, "imageId", imageId, "keypairName", keypairName, "vmType", vmType, "tags", vmTags)
		vm, err = r.Cloud.Compute(clusterScope.Tenant).CreateVm(ctx, machineScope, &vmSpec, imageId, subnetId, securityGroupIds, privateIps, vmName, clientToken, vmTags, volumes)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot create vm: %w", err)}
		}
		vmId := vm.VmId
		log.V(2).Info("VM created", "vmId", vmId)
		r.Tracker.trackVm(machineScope, vm)
		machineScope.SetVmState(vm.State)
		machineScope.SetProviderID(vm.Placement.SubregionName, vmId)
		r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
			infrastructurev1beta2.VmCreatedReason, "VM %s created in %s/%s", vm.VmId, subnetId, subregionName)
	}

	switch {
	case fgpu == nil:
	case fgpu.State == osc.FlexibleGpuStateAllocated:
		switch vm.State {
		case osc.VmStateStopped:
			log.V(3).Info("Attaching fGPU", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId)
			err := r.Cloud.Compute(clusterScope.Tenant).LinkFGPU(ctx, fgpu.FlexibleGpuId, vm.VmId)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("link fgpu: %w", err)}
			}
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.FGPUAttachedReason, "%s attached", fgpu.FlexibleGpuId)
		default:
			log.V(4).Info("Waiting for VM state", "vmId", vm.VmId, "state", vm.State)
		}
		return Result{Status: ResultStatusNotOK}
	case fgpu.State == osc.FlexibleGpuStateAttached:
		log.V(4).Info("fGPU is attached", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId)
	default:
		log.V(4).Info("Waiting for fGPU state", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId, "state", fgpu.State)
		return Result{Status: ResultStatusNotOK}
	}

	switch vm.State {
	case osc.VmStateStopped:
		log.V(3).Info("Starting VM", "vmId", vm.VmId)
		err := r.Cloud.Compute(clusterScope.Tenant).StartVm(ctx, vm.VmId)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("start vm: %w", err)}
		}
		machineScope.SetNotReady()
		return Result{Status: ResultStatusNotOK}
	case osc.VmStateRunning:
		log.V(4).Info("VM is running", "vmId", vm.VmId)
	default:
		log.V(4).Info("VM is not yet running", "vmId", vm.VmId, "state", vm.State)
		machineScope.SetNotReady()
		return Result{Status: ResultStatusNotOK}
	}

	machineScope.SetReady()

	if role == infrastructurev1beta2.RoleControlPlane && !clusterScope.GetSpec().Disable.Loadbalancer {
		svc := r.Cloud.Net(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().Name
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot get loadbalancer: %w", err)}
		}
		if loadbalancer == nil {
			return Result{Status: ResultStatusNotOK, Error: errors.New("no loadbalancer found")}
		}
		if !slices.Contains(loadbalancer.BackendVmIds, vm.VmId) {
			log.V(2).Info("Linking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.LinkLoadBalancerBackendMachines(ctx, []string{vm.VmId}, loadBalancerName)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot link vm to loadBalancer %s: %w", loadBalancerName, err)}
			}
		}
	}

	if vm.PrivateDnsName == nil {
		return Result{Status: ResultStatusNotOK, Error: errors.New("cannot find privateDnsName")}
	}
	addresses := []corev1.NodeAddress{}
	addresses = append(
		addresses,
		corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: vm.PrivateIp,
		},
	)
	// Expose Public IP if one is set
	if vm.PublicIp != nil {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeExternalIP,
			Address: *vm.PublicIp,
		})
	}
	machineScope.SetAddresses(addresses)
	machineScope.SetFailureDomain(vm.Placement.SubregionName)

	err = r.Cloud.Compute(clusterScope.Tenant).CheckVM(ctx, vm, clusterScope.GetUID(), ptr.From(vm.PrivateDnsName))
	if err != nil {
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot update vm: %w", err)}
	}
	machineScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerVm)
	return Result{Status: ResultStatusOK}
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func (r *OscMachineReconciler) reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) Result {
	log := ctrl.LoggerFrom(ctx)

	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case IsNotFound(err):
		log.V(2).Info("VM is already deleted")
		return Result{Status: ResultStatusOK}
	case err != nil:
		return Result{Status: ResultStatusUnknown, Error: fmt.Errorf("get existing: %w", err)}
	case vm.State == "terminated":
		log.V(4).Info("VM is already deleted")
		return Result{Status: ResultStatusOK}
	}

	if machineScope.GetSpec().Role == infrastructurev1beta2.RoleControlPlane && !clusterScope.GetSpec().Disable.Loadbalancer {
		svc := r.Cloud.Net(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().Name
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot get loadbalancer: %w", err)}
		}
		if loadbalancer != nil && slices.Contains(loadbalancer.BackendVmIds, vm.VmId) {
			log.V(2).Info("Unlinking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.UnlinkLoadBalancerBackendMachines(ctx, []string{vm.VmId}, loadBalancerName)
			if err != nil {
				return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot unlink vm from loadBalancer %s: %w", loadBalancerName, err)}
			}
		}
	}

	log.V(2).Info("Deleting VM", "vmId", vm.VmId)
	err = r.Cloud.Compute(clusterScope.Tenant).DeleteVm(ctx, vm.VmId)
	if err != nil {
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot delete vm: %w", err)}
	}
	return Result{Status: ResultStatusOK}
}

/*
func addTag(clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, api *osc.APIClient, auth context.Context, vm *osc.Vm) error {
	// Retrieve VM private DNS name
	privateDnsName, ok := vm.GetPrivateDnsNameOk()
	if !ok {
		return fmt.Errorf("failed to get private DNS name for VM")
	}

	// Define the cluster name and VM ID
	vmId := vm.VmId
	vmTag := osc.ResourceTag{
		Key:   "OscK8sNodeName",
		Value: *privateDnsName,
	}

	// Create the tag request
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: []string{vmId},
		Tags:        []osc.ResourceTag{vmTag},
	}

	// Call the AddTag function
	err, httpRes := tag.AddTag(vmTagRequest, []string{vmId}, api, auth)
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("failed to add tag: %s, HTTP status: %s: %w", err.Error(), httpRes.Status)
		}
		return fmt.Errorf("failed to add tag: %w", err)
	}

	log.V(4).Info("Tag successfully added", "vmId", vmId, "tagKey", vmTag.Key, "tagValue", vmTag.Value)
	return nil
}
*/
