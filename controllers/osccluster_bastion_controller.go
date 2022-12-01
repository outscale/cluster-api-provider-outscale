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

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
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
	clusterScope.Info("check match securityGroup with vm")
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
		checkOscAssociate = Contains(resourceNameList, validateBastionSecurityGroupName)
		if !checkOscAssociate {
			return fmt.Errorf("%s securityGroup does not exist in bastion", validateBastionSecurityGroupName)
		}
	}
	return nil
}

// checkBastionSubnetOscAssociateResourceName check the subnet tag name in both resource configuration are the same.
func checkBastionSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with bastion")
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionSubnetName := bastionSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := Contains(resourceNameList, bastionSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in bastion", bastionSubnetName)
	}
}

// checkBastionPublicIpOscAssociateResourceName check that PublicIp tag name in both resource configuration are the same.
func checkBastionPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match publicip with bastion")
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionPublicIpName := bastionSpec.PublicIpName + "-" + clusterScope.GetUID()
	publicIpsSpec := clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := Contains(resourceNameList, bastionPublicIpName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s publicIp does not exist in bastion", bastionPublicIpName)
	}
}

// checkBastionFormatParameters check Bastion parameters format.
func checkBastionFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Bastion parameters")
	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	bastionName := bastionSpec.Name + "-" + clusterScope.GetUID()
	bastionTagName, err := tag.ValidateTagNameValue(bastionName)
	if err != nil {
		return bastionTagName, err
	}

	imageName := bastionSpec.ImageName
	if imageName != "" {
		_, err := infrastructurev1beta1.ValidateImageName(imageName)
		if err != nil {
			return bastionTagName, err
		}
	} else {
		_, err := infrastructurev1beta1.ValidateImageId(bastionSpec.ImageId)
		if err != nil {
			return bastionTagName, err
		}
	}

	bastionKeypairName := bastionSpec.KeypairName
	_, err = infrastructurev1beta1.ValidateKeypairName(bastionKeypairName)
	if err != nil {
		return bastionTagName, err
	}

	vmType := bastionSpec.VmType
	_, err = infrastructurev1beta1.ValidateVmType(vmType)
	if err != nil {
		return bastionTagName, err
	}

	bastionDeviceName := bastionSpec.DeviceName
	_, err = infrastructurev1beta1.ValidateDeviceName(bastionDeviceName)
	if err != nil {
		return bastionTagName, err
	}

	bastionSubregionName := bastionSpec.SubregionName
	_, err = infrastructurev1beta1.ValidateSubregionName(bastionSubregionName)
	if err != nil {
		return bastionTagName, err
	}

	bastionSubnetName := bastionSpec.SubnetName
	clusterScope.Info("Get bastionSubnetName", "bastionSubnetName", bastionSubnetName)

	ipSubnetRange := clusterScope.GetIpSubnetRange(bastionSubnetName)
	bastionPrivateIps := clusterScope.GetBastionPrivateIps()
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	for _, bastionPrivateIp := range *bastionPrivateIps {
		privateIp := bastionPrivateIp.PrivateIp
		clusterScope.Info("### Get Valid IP ###", "privateIp", privateIp)
		clusterScope.Info("### Get Valid subnet ###", "ipSubnetRange", ipSubnetRange)
		_, err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return bastionTagName, err
		}
	}

	if bastionSpec.RootDisk.RootDiskIops != 0 {
		rootDiskIops := bastionSpec.RootDisk.RootDiskIops
		clusterScope.Info("Check rootDiskIops", "rootDiskIops", rootDiskIops)
		_, err := infrastructurev1beta1.ValidateIops(rootDiskIops)
		if err != nil {
			return bastionTagName, err
		}
	}

	rootDiskSize := bastionSpec.RootDisk.RootDiskSize
	clusterScope.Info("check rootDiskSize", "rootDiskSize", rootDiskSize)
	_, err = infrastructurev1beta1.ValidateSize(rootDiskSize)
	if err != nil {
		return bastionTagName, err
	}

	rootDiskType := bastionSpec.RootDisk.RootDiskType
	clusterScope.Info("Check rootDiskType", "rootDiskType", rootDiskType)
	_, err = infrastructurev1beta1.ValidateVolumeType(rootDiskType)
	if err != nil {
		return bastionTagName, err
	}
	return "", nil
}

// reconcileBastion reconcile the bastion of cluster.
func reconcileBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface, imageSvc compute.OscImageInterface) (reconcile.Result, error) {
	clusterScope.Info("Create Vm")
	bastionSpec := clusterScope.GetBastion()
	bastionRef := clusterScope.GetBastionRef()
	bastionName := bastionSpec.Name + "-" + clusterScope.GetUID()

	subnetName := bastionSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	imageId := bastionSpec.ImageId
	imageName := bastionSpec.ImageName
	if imageName != "" {
		if imageId, err = imageSvc.GetImageId(imageName); err != nil {
			return reconcile.Result{}, err
		}
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
		clusterScope.Info("Get bastionSecurityGroup", "bastionSecurityGroup", bastionSecurityGroup)
		securityGroupName := bastionSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		clusterScope.Info("get securityGroupId", "securityGroupId", securityGroupId)
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	var bastion *osc.Vm
	var vmID string
	if len(bastionRef.ResourceMap) == 0 {
		bastionRef.ResourceMap = make(map[string]string)
	}
	clusterScope.Info("### Get ResourceId ###", "resourceId", bastionSpec.ResourceId)
	clusterScope.Info("### Get ResourceMap ###", "resourceMap", bastionRef.ResourceMap)
	if bastionSpec.ResourceId != "" {
		bastionRef.ResourceMap[bastionName] = bastionSpec.ResourceId
		vmId := bastionSpec.ResourceId
		clusterScope.Info("Check if the desired bastion exist", "bastionName", bastionName)
		clusterScope.Info("### Get vmId ###", "bastion", bastionRef.ResourceMap)
		bastion, err = vmSvc.GetVm(vmId)
		if err != nil {
			return reconcile.Result{}, err
		}
		clusterScope.Info("Test GetVmState")
		vmState, err := vmSvc.GetVmState(vmId)
		if err != nil {
			clusterScope.SetVmState(infrastructurev1beta1.VmState("unknown"))
			return reconcile.Result{}, fmt.Errorf("%w Can not get bastion %s state for OscCluster %s/%s", err, vmId, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.SetVmState(infrastructurev1beta1.VmState(vmState))
		clusterScope.Info("Get bastion state", "vmState", vmState)
	}
	if bastion == nil || bastionSpec.ResourceId == "" {
		clusterScope.Info("Create the desired bastion", "bastionName", bastionName)
		keypairName := bastionSpec.KeypairName
		vmType := bastionSpec.VmType
		clusterScope.Info("### Info ImageId ###", "imageId", imageId)
		clusterScope.Info("### Info keypairName ###", "keypairName", keypairName)
		clusterScope.Info("### Info vmType ###", "vmType", vmType)

		vm, err := vmSvc.CreateVmUserData("", bastionSpec, subnetId, securityGroupIds, privateIps, bastionName, imageId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create bastion for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		vmID = vm.GetVmId()
		err = vmSvc.CheckVmState(5, 120, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscCluster %s/%s", err, vmID, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("Bastion is running", "vmID", vmID)
		clusterScope.SetVmState(infrastructurev1beta1.VmState("pending"))
		if bastionSpec.PublicIpName != "" {
			linkPublicIpId, err := publicIpSvc.LinkPublicIp(publicIpId, vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link publicIp %s with %s for OscCluster %s/%s", err, publicIpId, vmID, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			err = vmSvc.CheckVmState(5, 120, "running", vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscCluster %s/%s", err, vmID, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("Link public ip", "linkPublicIpId", linkPublicIpId)
			clusterScope.Info("Get bastionPublicIpName", "bastionPublicIpName", bastionPublicIpName)
			linkPublicIpRef.ResourceMap[bastionPublicIpName] = linkPublicIpId

		}
		clusterScope.Info("### Get Bastion ###", "bastion", bastion)
		bastionRef.ResourceMap[bastionName] = vmID
		bastionSpec.ResourceId = vmID
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteBastion reconcile the destruction of the machine bastion.
func reconcileDeleteBastion(ctx context.Context, clusterScope *scope.ClusterScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	clusterScope.Info("Delete vm")

	bastionSpec := clusterScope.GetBastion()
	bastionSpec.SetDefaultValue()
	vmId := bastionSpec.ResourceId
	clusterScope.Info("### VmiD ###", "vmId", vmId)
	bastionName := bastionSpec.Name
	if bastionSpec.ResourceId == "" {
		clusterScope.Info("The desired bastion is currently destroyed", "bastionName", bastionName)
		return reconcile.Result{}, nil
	}
	bastion, err := vmSvc.GetVm(vmId)
	if err != nil {
		return reconcile.Result{}, err
	}

	var securityGroupIds []string
	bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
	for _, bastionSecurityGroup := range *bastionSecurityGroups {
		securityGroupName := bastionSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	if bastion == nil {
		clusterScope.Info("The desired bastion does not exist anymore", "bastionName", bastionName)
		return reconcile.Result{}, nil
	}
	if bastionSpec.PublicIpName != "" {
		linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
		publicIpName := bastionSpec.PublicIpName + "-" + clusterScope.GetUID()
		err = publicIpSvc.UnlinkPublicIp(linkPublicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink publicIp for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	err = vmSvc.DeleteVm(vmId)
	bastionSpec.ResourceId = ""
	clusterScope.Info("Delete the desired bastion", "bastionName", bastionName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete vm for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
