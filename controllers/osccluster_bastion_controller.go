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
	"errors"
	"fmt"
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getBastionResourceId return vmId from the resourceMap is based on resourceName (tag name + cluster uid).
func getBastionResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	bastionRef := clusterScope.GetBastionRef()
	if vmId, ok := bastionRef.ResourceMap[resourceName]; ok {
		return vmId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkBastionSecurityGroupOscAssociateResourceName check that SecurityGroup tag name in both resource configuration are the same.
func checkBastionSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var bastionSecurityGroupNameList []string
	var checkOscAssociate bool
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
	for _, bastionSecurityGroup := range *bastionSecurityGroups {
		bastionSecurityGroupName := bastionSecurityGroup.Name + "-" + clusterScope.GetUID()
		bastionSecurityGroupNameList = append(bastionSecurityGroupNameList, bastionSecurityGroupName)
	}
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	for _, validateBastionSecurityGroupName := range bastionSecurityGroupNameList {
		checkOscAssociate = slices.Contains(resourceNameList, validateBastionSecurityGroupName)
		if !checkOscAssociate {
			return fmt.Errorf("%s securityGroup does not exist in bastion", validateBastionSecurityGroupName)
		}
	}
	return nil
}

// checkBastionSubnetOscAssociateResourceName check the subnet tag name in both resource configuration are the same.
func checkBastionSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionSubnetName := bastionSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := slices.Contains(resourceNameList, bastionSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in bastion", bastionSubnetName)
	}
}

// checkBastionPublicIpOscAssociateResourceName check that PublicIp tag name in both resource configuration are the same.
func checkBastionPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionPublicIpName := bastionSpec.PublicIpName + "-" + clusterScope.GetUID()
	publicIpsSpec := clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := slices.Contains(resourceNameList, bastionPublicIpName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s publicIp does not exist in bastion", bastionPublicIpName)
	}
}

// checkBastionFormatParameters check Bastion parameters format.
func checkBastionFormatParameters(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionName := bastionSpec.Name + "-" + clusterScope.GetUID()
	bastionTagName, err := tag.ValidateTagNameValue(bastionName)
	if err != nil {
		return bastionTagName, err
	}

	imageName := bastionSpec.ImageName
	ctrl.LoggerFrom(ctx).V(2).Info("Check Bastion parameters")
	if imageName != "" {
		err := infrastructurev1beta1.ValidateImageName(imageName)
		if err != nil {
			return bastionTagName, err
		}
	} else {
		err := infrastructurev1beta1.ValidateImageId(bastionSpec.ImageId)
		if err != nil {
			return bastionTagName, err
		}
	}

	bastionKeypairName := bastionSpec.KeypairName
	err = infrastructurev1beta1.ValidateKeypairName(bastionKeypairName)
	if err != nil {
		return bastionTagName, err
	}

	vmType := bastionSpec.VmType
	err = infrastructurev1beta1.ValidateVmType(vmType)
	if err != nil {
		return bastionTagName, err
	}

	bastionSubregionName := bastionSpec.SubregionName
	err = infrastructurev1beta1.ValidateSubregionName(bastionSubregionName)
	if err != nil {
		return bastionTagName, err
	}

	bastionSubnetName := bastionSpec.SubnetName
	log.V(4).Info("Get bastionSubnetName", "bastionSubnetName", bastionSubnetName)

	ipSubnetRange := clusterScope.GetIpSubnetRange(bastionSubnetName)
	log.V(4).Info("Get valid subnet", "ipSubnetRange", ipSubnetRange)
	bastionPrivateIps := clusterScope.GetBastionPrivateIps()
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	for _, bastionPrivateIp := range *bastionPrivateIps {
		privateIp := bastionPrivateIp.PrivateIp
		log.V(4).Info("Get valid IP", "privateIp", privateIp)

		err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return bastionTagName, err
		}
	}

	if bastionSpec.RootDisk.RootDiskIops != 0 {
		rootDiskIops := bastionSpec.RootDisk.RootDiskIops
		log.V(4).Info("Check rootDiskIops", "rootDiskIops", rootDiskIops)
		err := infrastructurev1beta1.ValidateIops(rootDiskIops)
		if err != nil {
			return bastionTagName, err
		}
	}

	rootDiskSize := bastionSpec.RootDisk.RootDiskSize
	log.V(4).Info("Check rootDiskSize", "rootDiskSize", rootDiskSize)
	err = infrastructurev1beta1.ValidateSize(rootDiskSize)
	if err != nil {
		return bastionTagName, err
	}

	rootDiskType := bastionSpec.RootDisk.RootDiskType
	log.V(4).Info("Check rootDiskType", "rootDiskType", rootDiskType)
	err = infrastructurev1beta1.ValidateVolumeType(rootDiskType)
	if err != nil {
		return bastionTagName, err
	}
	return "", nil
}

// reconcileBastion reconcile the bastion of cluster.
func (r *OscClusterReconciler) reconcileBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface, imageSvc compute.OscImageInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerBastion) {
		log.V(4).Info("No need for bastion reconciliation")
		return reconcile.Result{}, nil
	}

	vm, err := r.Tracker.getBastion(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound):
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("reconcile nat service: %w", err)
	default:
		log.V(4).Info("Bastion vm state", "vmState", vm.GetState())
		clusterScope.SetVmState(infrastructurev1beta1.VmState(vm.GetState()))
		if vm.GetState() != "running" {
			return reconcile.Result{}, errors.New("bastion is not yet running")
		}
		clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerBastion)
		return reconcile.Result{}, nil
	}

	bastionSpec := clusterScope.GetBastion()
	subnetSpec, err := clusterScope.GetSubnet(bastionSpec.SubnetName, infrastructurev1beta1.RolePublic, "")
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile bastion: %w")
	}
	subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile bastion: %w")
	}

	publicIpId := bastionSpec.PublicIpId
	if publicIpId == "" {
		publicIpId, err = r.Tracker.allocateIP(ctx, "bastion", clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile bastion: %w")
		}
	}
	var privateIps []string
	bastionPrivateIps := clusterScope.GetBastionPrivateIps()
	if len(*bastionPrivateIps) > 0 {
		for _, bastionPrivateIp := range *bastionPrivateIps {
			privateIp := bastionPrivateIp.PrivateIp
			privateIps = append(privateIps, privateIp)
		}
	}

	var securityGroupIds []string
	bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
	for _, bastionSecurityGroup := range *bastionSecurityGroups {
		log.V(4).Info("Get bastionSecurityGroup", "bastionSecurityGroup", bastionSecurityGroup)
		securityGroupName := bastionSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		log.V(4).Info("Get securityGroupId", "securityGroupId", securityGroupId)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	imageId := bastionSpec.ImageId
	if imageId == "" && bastionSpec.ImageName != "" {
		image, err := imageSvc.GetImageByName(ctx, bastionSpec.ImageName, bastionSpec.ImageAccountId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to find image %s: %w", bastionSpec.ImageName, err)
		}
		if image == nil {
			return reconcile.Result{}, fmt.Errorf("unable to find image %s", bastionSpec.ImageName)
		}
		imageId = *image.ImageId
	}

	vmType := bastionSpec.VmType
	log.V(3).Info("Creating bastion", "vmType", vmType)
	tags := map[string]string{
		compute.AutoAttachExternapIPTag: publicIpId,
	}
	bastionName := clusterScope.GetBastionName()
	vm, err = r.Tracker.Cloud.VM(ctx, *clusterScope).CreateVmBastion(ctx, bastionSpec, subnetId, securityGroupIds, privateIps, bastionName, imageId, tags)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("can not create bastion: %w", err)
	}
	log.V(2).Info("Bastion created", "vmId", vm.GetVmId())
	r.Tracker.setBastionId(clusterScope, vm.GetVmId())
	clusterScope.SetVmState(infrastructurev1beta1.VmStatePending)
	return reconcile.Result{}, errors.New("VM is not running yet")
}

// reconcileDeleteBastion reconcile the destruction of the machine bastion.
func (r *OscClusterReconciler) reconcileDeleteBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(4).Info("Not deleting existing bastion")
		return reconcile.Result{}, nil
	}
	vm, err := r.Tracker.getBastion(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The bastion is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("reconcile delete bastion: %w", err)
	}

	// Status may have been reset, and IP tracking lost, we need to ensure the public ip is tracked to be deleted.
	publicIpId := tag.GetTagValue(compute.AutoAttachExternapIPTag, vm.GetTags())
	bastionSpec := clusterScope.GetBastion()

	if publicIpId != "" && bastionSpec.PublicIpId == "" {
		r.Tracker.trackIP(clusterScope, "bastion", publicIpId)
	}

	log.V(2).Info("Deleting bastion", "vmId", vm.GetVmId())
	err = vmSvc.DeleteVm(ctx, vm.GetVmId())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete bastion: %w", err)
	}
	log.V(2).Info("Bastion deleted")
	return reconcile.Result{}, nil
}
