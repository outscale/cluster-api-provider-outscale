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
	"time"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileVm reconcile the vm of the machine
func (r *OscMachineReconciler) reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !machineScope.NeedReconciliation(infrastructurev1beta1.ReconcilerVm) {
		log.V(4).Info("No need for vm reconciliation")
		return reconcile.Result{}, nil
	}

	vmSpec := machineScope.GetVm()

	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case err == nil:
		machineScope.SetVmState(infrastructurev1beta1.VmState(vm.GetState()))
	case !errors.Is(err, ErrNoResourceFound):
		return reconcile.Result{}, fmt.Errorf("cannot get VM: %w", err)
	default:
		// Check if a machine needs to be placed in a subregion.
		// failure domain may either be a subnet name (CAPOSC up to v0.4.0) or a subregion (v0.5.0 or later).
		var subnetName, subregionName string
		if machineScope.Machine.Spec.FailureDomain != nil {
			subnetName = *machineScope.Machine.Spec.FailureDomain
			subregionName = *machineScope.Machine.Spec.FailureDomain
		} else {
			subnetName = vmSpec.SubnetName
			subregionName = vmSpec.SubregionName
		}

		subnetSpec, err := clusterScope.GetSubnet(subnetName, vmSpec.GetRole(), subregionName)
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
		vmPrivateIps := machineScope.GetVmPrivateIps()
		privateIps := make([]string, 0, len(vmPrivateIps))
		for _, vmPrivateIp := range vmPrivateIps {
			privateIp := vmPrivateIp.PrivateIp
			privateIps = append(privateIps, privateIp)
		}
		imageId, err := r.Tracker.getImageId(ctx, machineScope, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
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
				return reconcile.Result{}, fmt.Errorf("allocate IP: %w", err)
			}
			vmTags[compute.AutoAttachExternalIPTag] = publicIp
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

		keypairName := vmSpec.KeypairName
		vmType := vmSpec.VmType
		volumes := machineScope.GetVolumes()
		clientToken := machineScope.GetClientToken(clusterScope)
		log.V(3).Info("Creating VM", "vmName", vmName, "imageId", imageId, "keypairName", keypairName, "vmType", vmType, "tags", vmTags)
		vm, err = r.Cloud.VM(clusterScope.Tenant).CreateVm(ctx, machineScope, &vmSpec, imageId, subnetId, securityGroupIds, privateIps, vmName, clientToken, vmTags, volumes)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create vm: %w", err)
		}
		vmId := vm.GetVmId()
		log.V(2).Info("VM created", "vmId", vmId)
		r.Tracker.trackVm(machineScope, vm)
		machineScope.SetVmState(infrastructurev1beta1.VmState(vm.GetState()))
		machineScope.SetProviderID(vm.Placement.GetSubregionName(), vmId)
		r.Recorder.Event(machineScope.OscMachine, corev1.EventTypeNormal, infrastructurev1beta1.VmCreatedReason, "VM created")
	}

	if vm.GetState() != "running" {
		log.V(4).Info(fmt.Sprintf("VM %s is not yet running", vm.GetVmId()))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	machineScope.SetReady()

	if vmSpec.GetRole() == infrastructurev1beta1.RoleControlPlane {
		svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().LoadBalancerName
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
		}
		if loadbalancer == nil {
			return reconcile.Result{}, errors.New("no loadbalancer found")
		}
		if !slices.Contains(loadbalancer.GetBackendVmIds(), vm.GetVmId()) {
			log.V(2).Info("Linking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.LinkLoadBalancerBackendMachines(ctx, []string{vm.GetVmId()}, loadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot link vm %s to loadBalancerName %s: %w", vm.GetVmId(), loadBalancerName, err)
			}
		}
	}

	privateDnsName, ok := vm.GetPrivateDnsNameOk()
	if !ok {
		return reconcile.Result{}, errors.New("cannot find privateDnsName")
	}
	privateIp, ok := vm.GetPrivateIpOk()
	if !ok {
		return reconcile.Result{}, errors.New("cannot find privateIp")
	}
	addresses := []corev1.NodeAddress{}
	addresses = append(
		addresses,
		corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: *privateIp,
		},
	)
	// Expose Public IP if one is set
	if publicIp, ok := vm.GetPublicIpOk(); ok {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeExternalIP,
			Address: *publicIp,
		})
	}
	machineScope.SetAddresses(addresses)
	if vmSpec.GetRole() == infrastructurev1beta1.RoleControlPlane {
		machineScope.SetFailureDomain(vm.Placement.GetSubregionName())
	}

	if !compute.HasCCMTags(vm) {
		log.V(2).Info("Adding CCM tags")
		err = r.Cloud.VM(clusterScope.Tenant).AddCCMTags(ctx, clusterScope.GetUID(), *privateDnsName, vm.GetVmId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot add ccm tag: %w", err)
		}
	}
	machineScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerVm)
	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func (r *OscMachineReconciler) reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	vm, err := r.Tracker.getVm(ctx, machineScope, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(2).Info("VM is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("cannot get vm: %w", err)
	case vm.GetState() == "terminated":
		log.V(4).Info("VM is already deleted")
		return reconcile.Result{}, nil
	}

	vmSpec := machineScope.GetVm()
	if vmSpec.GetRole() == infrastructurev1beta1.RoleControlPlane {
		svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
		loadBalancerName := clusterScope.GetLoadBalancer().LoadBalancerName
		loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
		}
		if loadbalancer != nil && slices.Contains(loadbalancer.GetBackendVmIds(), vm.GetVmId()) {
			log.V(2).Info("Unlinking loadbalancer", "loadBalancerName", loadBalancerName)
			err := svc.UnlinkLoadBalancerBackendMachines(ctx, []string{vm.GetVmId()}, loadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink vm %s to loadBalancerName %s: %w", vm.GetVmId(), loadBalancerName, err)
			}
		}
	}

	log.V(2).Info("Deleting VM", "vmId", vm.GetVmId())
	err = r.Cloud.VM(clusterScope.Tenant).DeleteVm(ctx, vm.GetVmId())
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
	vmId := vm.GetVmId()
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
