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
	"time"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
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
	subnetsSpec := clusterScope.GetSubnet()
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
func reconcileBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface, imageSvc compute.OscImageInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	bastionSpec := clusterScope.GetBastion()
	bastionRef := clusterScope.GetBastionRef()
	bastionName := bastionSpec.Name + "-" + clusterScope.GetUID()
	bastionState := clusterScope.GetVmState()
	log.V(4).Info("Reconcile bastion", "resourceId", bastionSpec.ResourceId, "state", bastionState)

	subnetName := bastionSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	var publicIpId string
	var bastionPublicIpName string
	var linkPublicIpRef *infrastructurev1beta1.OscResourceReference
	if bastionSpec.PublicIpName != "" {
		bastionPublicIpName := bastionSpec.PublicIpName + "-" + clusterScope.GetUID()
		publicIpId, err = getPublicIpResourceId(bastionPublicIpName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		linkPublicIpRef = clusterScope.GetLinkPublicIpRef()
		if len(linkPublicIpRef.ResourceMap) == 0 {
			linkPublicIpRef.ResourceMap = make(map[string]string)
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
	var vm *osc.Vm
	var vmId string
	if len(bastionRef.ResourceMap) == 0 {
		bastionRef.ResourceMap = make(map[string]string)
	}

	if bastionState == nil {
		vms, err := vmSvc.GetVmListFromTag(ctx, "Name", bastionName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("could not list vms: %w", err)
		}
		if len(vms) > 0 {
			if bastionSpec.ResourceId != "" {
				clusterScope.SetVmState(infrastructurev1beta1.VmStatePending)
				bastionRef.ResourceMap[bastionName] = bastionSpec.ResourceId
				return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return reconcile.Result{}, fmt.Errorf("a bastion with name %s already exists", bastionName)
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

		keypairName := bastionSpec.KeypairName
		vmType := bastionSpec.VmType
		log.V(4).Info("Creating bastion", "bastionName", bastionName, "keypairName", keypairName, "vmType", vmType)

		vm, err = vmSvc.CreateVmUserData(ctx, "", bastionSpec, subnetId, securityGroupIds, privateIps, bastionName, imageId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("can not create bastion: %w", err)
		}
		vmId = vm.GetVmId()
		clusterScope.SetVmState(infrastructurev1beta1.VmStatePending)
		bastionState = &infrastructurev1beta1.VmStatePending
		bastionRef.ResourceMap[bastionName] = vmId
		bastionSpec.ResourceId = vmId
		log.V(3).Info("Bastion created", "bastionId", vmId)
	}

	if bastionState != nil {
		if *bastionState != infrastructurev1beta1.VmStateRunning {
			vmId := bastionSpec.ResourceId
			vm, err := vmSvc.GetVm(ctx, vmId)
			if err != nil {
				return reconcile.Result{}, err
			}
			currentVmState := vm.GetState()
			clusterScope.SetVmState(infrastructurev1beta1.VmState(currentVmState))
			log.V(4).Info("Bastion vm state", "vmState", currentVmState)

			if infrastructurev1beta1.VmState(currentVmState) != infrastructurev1beta1.VmStateRunning {
				log.V(3).Info("Bastion vm is not yet running", "vmId", vmId)
				return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("bastion %s is not running yet", vmId)
			}
			bastionState = &infrastructurev1beta1.VmStateRunning
			log.V(3).Info("Bastion is running", "vmId", vmId)
		}

		if bastionState != nil && *bastionState == infrastructurev1beta1.VmStateRunning {
			vmId := bastionSpec.ResourceId

			if bastionSpec.PublicIpName != "" && linkPublicIpRef.ResourceMap[bastionPublicIpName] == "" {
				linkPublicIpId, err := publicIpSvc.LinkPublicIp(ctx, publicIpId, vmId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot link publicIp %s with %s: %w", publicIpId, vmId, err)
				}
				log.V(4).Info("Get bastionPublicIpName", "bastionPublicIpName", bastionPublicIpName)
				linkPublicIpRef.ResourceMap[bastionPublicIpName] = linkPublicIpId
			}
		} else {
			log.V(4).Info("VM is not running, skipping public IP linking")
		}
	}
	log.V(4).Info("Bastion is reconciled")
	return reconcile.Result{}, nil
}

// reconcileDeleteBastion reconcile the destruction of the machine bastion.
func reconcileDeleteBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	vmId := bastionSpec.ResourceId
	log.V(4).Info("Reconcile delete bastion", "resourceId", vmId)
	bastionName := bastionSpec.Name
	if bastionSpec.ResourceId == "" {
		log.V(3).Info("The bastion is already destroyed")
		return reconcile.Result{}, nil
	}
	bastion, err := vmSvc.GetVm(ctx, vmId)
	if err != nil {
		return reconcile.Result{}, err
	}

	bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
	for _, bastionSecurityGroup := range *bastionSecurityGroups {
		securityGroupName := bastionSecurityGroup.Name + "-" + clusterScope.GetUID()
		_, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if bastion == nil {
		log.V(3).Info("The bastion is already destroyed")
		return reconcile.Result{}, nil
	}
	if bastionSpec.PublicIpName != "" {
		linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
		publicIpName := bastionSpec.PublicIpName + "-" + clusterScope.GetUID()
		publicIpRef := linkPublicIpRef.ResourceMap[publicIpName]
		if publicIpRef != "" {
			err = publicIpSvc.UnlinkPublicIp(ctx, publicIpRef)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink bastion publicIp: %w", err)
			}
		}
	}
	err = vmSvc.DeleteVm(ctx, vmId)
	bastionSpec.ResourceId = ""
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete bastion: %w", err)
	}
	log.V(2).Info("Bastion deleted", "bastionName", bastionName)
	return reconcile.Result{}, nil
}
