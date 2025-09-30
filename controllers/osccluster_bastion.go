/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkBastionFormatParameters check Bastion parameters format.
// func checkBastionFormatParameters(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
// 	log := ctrl.LoggerFrom(ctx)
// 	bastionSpec := clusterScope.GetBastion()
// 	bastionSpec.SetDefaultValue()
// 	bastionName := bastionSpec.Name + "-" + clusterScope.GetUID()
// 	bastionTagName, err := tag.ValidateTagNameValue(bastionName)
// 	if err != nil {
// 		return bastionTagName, err
// 	}

// 	imageName := bastionSpec.ImageName
// 	ctrl.LoggerFrom(ctx).V(2).Info("Check Bastion parameters")
// 	if imageName != "" {
// 		err := infrastructurev1beta1.ValidateImageName(imageName)
// 		if err != nil {
// 			return bastionTagName, err
// 		}
// 	} else {
// 		err := infrastructurev1beta1.ValidateImageId(bastionSpec.ImageId)
// 		if err != nil {
// 			return bastionTagName, err
// 		}
// 	}

// 	bastionKeypairName := bastionSpec.KeypairName
// 	err = infrastructurev1beta1.ValidateKeypairName(bastionKeypairName)
// 	if err != nil {
// 		return bastionTagName, err
// 	}

// 	vmType := bastionSpec.VmType
// 	err = infrastructurev1beta1.ValidateVmType(vmType)
// 	if err != nil {
// 		return bastionTagName, err
// 	}

// 	bastionSubregionName := bastionSpec.SubregionName
// 	err = infrastructurev1beta1.ValidateSubregionName(bastionSubregionName)
// 	if err != nil {
// 		return bastionTagName, err
// 	}

// 	bastionSubnetName := bastionSpec.SubnetName
// 	log.V(4).Info("Get bastionSubnetName", "bastionSubnetName", bastionSubnetName)

// 	ipSubnetRange := clusterScope.GetIpSubnetRange(bastionSubnetName)
// 	log.V(4).Info("Get valid subnet", "ipSubnetRange", ipSubnetRange)
// 	bastionPrivateIps := clusterScope.GetBastionPrivateIps()
// 	for _, bastionPrivateIp := range bastionPrivateIps {
// 		privateIp := bastionPrivateIp.PrivateIp
// 		log.V(4).Info("Get valid IP", "privateIp", privateIp)

// 		err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
// 		if err != nil {
// 			return bastionTagName, err
// 		}
// 	}

// 	if bastionSpec.RootDisk.RootDiskIops != 0 {
// 		rootDiskIops := bastionSpec.RootDisk.RootDiskIops
// 		log.V(4).Info("Check rootDiskIops", "rootDiskIops", rootDiskIops)
// 		err := infrastructurev1beta1.ValidateIops(rootDiskIops)
// 		if err != nil {
// 			return bastionTagName, err
// 		}
// 	}

// 	rootDiskSize := bastionSpec.RootDisk.RootDiskSize
// 	log.V(4).Info("Check rootDiskSize", "rootDiskSize", rootDiskSize)
// 	err = infrastructurev1beta1.ValidateSize(rootDiskSize)
// 	if err != nil {
// 		return bastionTagName, err
// 	}

// 	rootDiskType := bastionSpec.RootDisk.RootDiskType
// 	log.V(4).Info("Check rootDiskType", "rootDiskType", rootDiskType)
// 	err = infrastructurev1beta1.ValidateVolumeType(rootDiskType)
// 	if err != nil {
// 		return bastionTagName, err
// 	}
// 	return "", nil
// }

// reconcileBastion reconcile the bastion of cluster.
func (r *OscClusterReconciler) reconcileBastion(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerBastion) {
		log.V(4).Info("No need for bastion reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Reusing existing bastion")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling bastion")

	vm, err := r.Tracker.getBastion(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound):
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
	default:
		log.V(4).Info("Found bastion", "vmId", vm.GetVmId(), "vmState", vm.GetState())
		clusterScope.SetVmState(infrastructurev1beta1.VmState(vm.GetState()))
		if vm.GetState() != "running" {
			return reconcile.Result{}, errors.New("bastion is not yet running")
		}
		clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerBastion)
		return reconcile.Result{}, nil
	}

	bastionSpec := clusterScope.GetBastion()
	subnetSpec, err := clusterScope.GetSubnet(bastionSpec.SubnetName, infrastructurev1beta1.RoleBastion, "")
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get subnet: %w", err)
	}
	subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("load subnet: %w", err)
	}

	_, publicIp, err := r.Tracker.IPAllocator(clusterScope).AllocateIP(ctx, bastionIPResourceKey, clusterScope.GetBastionName(), "", clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("allocate IP: %w", err)
	}
	bastionPrivateIps := clusterScope.GetBastionPrivateIps()
	privateIps := make([]string, 0, len(bastionPrivateIps))
	for _, bastionPrivateIp := range bastionPrivateIps {
		privateIp := bastionPrivateIp.PrivateIp
		privateIps = append(privateIps, privateIp)
	}

	bastionSecurityGroups, err := clusterScope.GetSecurityGroupsFor(bastionSpec.SecurityGroupNames, infrastructurev1beta1.RoleBastion)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot find securityGroup: %w", err)
	}
	if len(bastionSecurityGroups) == 0 {
		return reconcile.Result{}, errors.New("no securityGroup found")
	}
	securityGroupIds := make([]string, 0, len(bastionSecurityGroups))
	for _, sgSpec := range bastionSecurityGroups {
		securityGroupId, err := r.Tracker.getSecurityGroupId(ctx, sgSpec, clusterScope)
		log.V(4).Info("Found securityGroup", "securityGroupId", securityGroupId)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	imageId := bastionSpec.ImageId
	if imageId == "" && bastionSpec.ImageName != "" {
		image, err := r.Cloud.Image(clusterScope.Tenant).GetImageByName(ctx, bastionSpec.ImageName, bastionSpec.ImageAccountId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot find image %s: %w", bastionSpec.ImageName, err)
		}
		if image == nil {
			return reconcile.Result{}, fmt.Errorf("cannot find image %s", bastionSpec.ImageName)
		}
		imageId = *image.ImageId
	}

	log.V(3).Info("Creating bastion", "vmType", bastionSpec.VmType)
	tags := map[string]string{
		compute.AutoAttachExternapIPTag: publicIp,
	}
	bastionName := clusterScope.GetBastionName()
	clientToken := clusterScope.GetBastionClientToken()
	vm, err = r.Cloud.VM(clusterScope.Tenant).CreateVmBastion(ctx, &bastionSpec, subnetId, securityGroupIds, privateIps, bastionName, clientToken, imageId, tags)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create bastion: %w", err)
	}
	log.V(2).Info("Bastion created", "vmId", vm.GetVmId())
	r.Tracker.setBastionId(clusterScope, vm.GetVmId())
	clusterScope.SetVmState(infrastructurev1beta1.VmStatePending)
	r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.VmCreatedReason, "Bastion created")
	return reconcile.Result{}, errors.New("VM is not running yet")
}

// reconcileDeleteBastion reconcile the destruction of the machine bastion.
func (r *OscClusterReconciler) reconcileDeleteBastion(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing bastion")
		return reconcile.Result{}, nil
	}
	vm, err := r.Tracker.getBastion(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The bastion is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
	case vm.GetState() == "terminated":
		log.V(4).Info("The bastion is already deleted")
		return reconcile.Result{}, nil
	}

	log.V(3).Info("Deleting bastion")
	err = r.Cloud.VM(clusterScope.Tenant).DeleteVm(ctx, vm.GetVmId())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete bastion: %w", err)
	}
	log.V(2).Info("Bastion deleted", "vmId", vm.GetVmId())
	return reconcile.Result{}, nil
}
