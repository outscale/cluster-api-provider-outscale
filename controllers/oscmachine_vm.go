/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileVm reconcile the vm of the machine
func (r *OscMachineReconciler) reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !machineScope.NeedReconciliation(infrastructurev1beta2.ReconcilerVm) {
		log.V(4).Info("No need for vm reconciliation")
		return reconcile.Result{}, nil
	}

	bootstrapData, err := machineScope.GetBootstrapData(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to decode bootstrap data: %w", err)
	}

	vmSpec := machineScope.GetVm()

	var fgpu *osc.FlexibleGpu
	if vmSpec.FGPU != nil {
		var err error
		fgpu, err = r.Tracker.getFGPU(ctx, machineScope, clusterScope)
		if err != nil && !IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}
	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case err == nil:
		machineScope.SetVmState(vm.State)
	case !errors.Is(err, ErrNoResourceFound):
		return reconcile.Result{}, fmt.Errorf("cannot get VM: %w", err)
	default:
		// Check if a machine needs to be placed in a subregion.
		var subregionName string
		switch {
		case fgpu != nil:
			subregionName = fgpu.SubregionName
		case machineScope.Machine.Spec.FailureDomain != "":
			subregionName = machineScope.Machine.Spec.FailureDomain
		default:
			azs := vmSpec.Subregions
			if len(azs) == 0 {
				azs = clusterScope.GetSubregions()
				log.V(4).Info("Using cluster subregions")
			}
			az, err := r.AZAllocator.AllocateAZ(ctx, machineScope.OscMachine, vmSpec.SubregionMode, azs)
			if err != nil {
				return reconcile.Result{}, err
			}
			subregionName = az
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
			fgpu, err = r.Cloud.FlexibleGPU(clusterScope.Tenant).AllocateFGPU(ctx, vmSpec.FGPU.Model, subregionName, machineScope)
			if err != nil {
				return reconcile.Result{}, err
			}
			log.V(2).Info("fGPU allocated", "fGPUId", fgpu.FlexibleGpuId, "model", vmSpec.FGPU.Model)
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.FGPUAllocatedReason, "%s (%s) allocated", fgpu.FlexibleGpuId, vmSpec.FGPU.Model)
		}

		subnetSpec, err := clusterScope.GetSubnet(vmSpec.GetRole(), subregionName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile vm: %w", err)
		}
		subnetId, err := r.ClusterTracker.getSubnetId(ctx, subnetSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile vm: %w", err)
		}
		securityGroups, err := clusterScope.GetSecurityGroupsFor(machineScope.GetVmSecurityGroups(), vmSpec.GetRole())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot find securityGroup: %w", err)
		}
		securityGroupIds := make([]string, 0, len(securityGroups))
		for _, sgSpec := range securityGroups {
			securityGroupId, err := r.ClusterTracker.getSecurityGroupId(ctx, sgSpec, clusterScope)
			log.V(4).Info("Found securityGroup", "securityGroupId", securityGroupId)
			if err != nil {
				return reconcile.Result{}, err
			}
			securityGroupIds = append(securityGroupIds, securityGroupId)
		}

		imageId, err := r.Tracker.getImageId(ctx, machineScope, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		vmName := machineScope.GetName()

		var publicIp string
		if vmSpec.PublicIp {
			var err error
			_, publicIp, err = r.Tracker.IPAllocator(machineScope).AllocateIP(ctx, defaultResource, vmName, vmSpec.PublicIpPool, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("allocate IP: %w", err)
			}
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.IPAllocated, "IP %s allocated", publicIp)
		}

		vmType := vmSpec.VmType
		clientToken := machineScope.GetClientToken(clusterScope)
		log.V(3).Info("Creating VM", "vmName", vmName, "imageId", imageId, "vmType", vmType, "publicIp", publicIp)
		vm, err = r.Cloud.VM(clusterScope.Tenant).CreateVm(ctx, &vmSpec, bootstrapData, imageId, subnetId, securityGroupIds, vmName, clientToken, publicIp)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create vm: %w", err)
		}
		log.V(2).Info("VM created", "vmId", vm.VmId)
		r.Tracker.trackVm(machineScope, vm)
		machineScope.SetVmState(vm.State)
		machineScope.SetProviderID(vm.Placement.SubregionName, vm.VmId)
		r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
			infrastructurev1beta2.VmCreatedReason, "VM %s created in %s", vm.VmId, subregionName)
	}

	switch {
	case fgpu == nil:
	case fgpu.State == "allocated":
		switch vm.State {
		case "stopped":
			log.V(3).Info("Attaching fGPU", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId)
			err := r.Cloud.FlexibleGPU(clusterScope.Tenant).LinkFGPU(ctx, fgpu.FlexibleGpuId, vm.VmId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("link fgpu: %w", err)
			}
			r.Recorder.Eventf(machineScope.OscMachine, corev1.EventTypeNormal,
				infrastructurev1beta2.FGPUAttachedReason, "%s attached", fgpu.FlexibleGpuId)
		default:
			log.V(4).Info("Waiting for VM state", "vmId", vm.VmId, "state", vm.State)
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	case fgpu.State == "attached":
		log.V(4).Info("fGPU is attached", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId)
	default:
		log.V(4).Info("Waiting for fGPU state", "vmId", vm.VmId, "fGPUId", fgpu.FlexibleGpuId, "state", fgpu.State)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	switch vm.State {
	case "stopped":
		log.V(3).Info("Starting VM", "vmId", vm.VmId)
		err := r.Cloud.VM(clusterScope.Tenant).StartVm(ctx, vm.VmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("start vm: %w", err)
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	case "running":
		log.V(4).Info("VM is running", "vmId", vm.VmId)
	default:
		log.V(4).Info("VM is not yet running", "vmId", vm.VmId, "state", vm.State)
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	machineScope.SetReady()

	if vmSpec.GetRole() == infrastructurev1beta2.RoleControlPlane && !clusterScope.IsLBDisabled() {
		svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().LoadBalancerName
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
		}
		if loadbalancer == nil {
			return reconcile.Result{}, errors.New("no loadbalancer found")
		}
		if !slices.Contains(loadbalancer.BackendVmIds, vm.VmId) {
			log.V(2).Info("Linking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.LinkLoadBalancerBackendMachines(ctx, []string{vm.VmId}, loadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot link vm %s to loadBalancerName %s: %w", vm.VmId, loadBalancerName, err)
			}
		}
	}

	privateDnsName := ptr.From(vm.PrivateDnsName)
	addresses := []clusterv1.MachineAddress{}
	addresses = append(
		addresses,
		clusterv1.MachineAddress{
			Type:    clusterv1.MachineInternalIP,
			Address: vm.PrivateIp,
		},
	)
	// Expose Public IP if one is set
	if vm.PublicIp != nil {
		addresses = append(addresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP,
			Address: *vm.PublicIp,
		})
	}
	machineScope.SetAddresses(addresses)
	machineScope.SetFailureDomain(vm.Placement.SubregionName)

	if !compute.HasCCMTags(vm) {
		log.V(2).Info("Adding CCM tags")
		err = r.Cloud.VM(clusterScope.Tenant).AddCCMTags(ctx, clusterScope.GetUID(), privateDnsName, vm.VmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot add ccm tag: %w", err)
		}
	}
	machineScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerVm)
	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func (r *OscMachineReconciler) reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case IsNotFound(err):
		log.V(2).Info("VM is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("cannot get vm: %w", err)
	case vm.State == osc.VmStateTerminated:
		log.V(4).Info("VM is already deleted")
		return reconcile.Result{}, nil
	}

	vmSpec := machineScope.GetVm()
	if vmSpec.GetRole() == infrastructurev1beta2.RoleControlPlane && !clusterScope.IsLBDisabled() {
		svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().LoadBalancerName
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
		}
		if loadbalancer != nil && slices.Contains(loadbalancer.BackendVmIds, vm.VmId) {
			log.V(2).Info("Unlinking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.UnlinkLoadBalancerBackendMachines(ctx, []string{vm.VmId}, loadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink vm %s to loadBalancerName %s: %w", vm.VmId, loadBalancerName, err)
			}
		}
	}

	log.V(2).Info("Deleting VM", "vmId", vm.VmId)
	err = r.Cloud.VM(clusterScope.Tenant).DeleteVm(ctx, vm.VmId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete vm: %w", err)
	}
	return reconcile.Result{}, nil
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
