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
	"strings"
	"time"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getVmResourceId return the vmId from the resourceMap base on resourceName (tag name + cluster uid)
func getVmResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	vmRef := machineScope.GetVmRef()
	if vmId, ok := vmRef.ResourceMap[resourceName]; ok {
		return vmId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkVmVolumeOscAssociateResourceName check that Volume dependancies tag name in both resource configuration are the same.
func checkVmVolumeOscAssociateResourceName(machineScope *scope.MachineScope) error {
	var resourceNameList []string
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmVolumeName := vmSpec.VolumeName + "-" + machineScope.GetUID()
	volumesSpec := machineScope.GetVolume()
	for _, volumeSpec := range volumesSpec {
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		resourceNameList = append(resourceNameList, volumeName)
	}
	machineScope.V(2).Info("Check match volume with vm")
	checkOscAssociate := Contains(resourceNameList, vmVolumeName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s volume does not exist in vm", vmVolumeName)
	}
}

// checkVmLoadBalancerOscAssociateResourceName  check that LoadBalancer dependancies tag name in both resource configuration are the same.
func checkVmLoadBalancerOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmLoadBalancerName := vmSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	machineScope.V(4).Info("Get VmLoadBalancerName", "vmLoadBalancerName", vmLoadBalancerName)
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	machineScope.V(4).Info("Get LoadBalancerName", "loadBalancerName", loadBalancerName)
	machineScope.V(4).Info("Get Role", "Role", vmSpec.Role)
	resourceNameList = append(resourceNameList, loadBalancerName)
	checkOscAssociate := Contains(resourceNameList, vmLoadBalancerName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s loadBalancer does not exist in vm", vmLoadBalancerName)
	}

}

func checkVmVolumeSubregionName(machineScope *scope.MachineScope) error {
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmVolumeName := vmSpec.VolumeName
	volumeSubregionName := machineScope.GetVolumeSubregionName(vmVolumeName)
	vmSubregionName := vmSpec.SubregionName
	vmName := vmSpec.Name
	machineScope.V(2).Info("Check have the same subregionName for vm and for volume")
	if vmSubregionName != volumeSubregionName {
		return fmt.Errorf("volume %s and vm %s are not in the same subregion %s", vmVolumeName, vmName, vmSubregionName)
	} else {
		return nil
	}
}

// checkVmSecurityGroupOscAssociateResourceName check that SecurityGroup dependancies tag name in both resource configuration are the same.
func checkVmSecurityGroupOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var vmSecurityGroupNameList []string
	var checkOscAssociate bool
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		vmSecurityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		vmSecurityGroupNameList = append(vmSecurityGroupNameList, vmSecurityGroupName)
	}
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	for _, validateVmSecurityGroupName := range vmSecurityGroupNameList {
		machineScope.V(2).Info("Check match securityGroup with vm")
		checkOscAssociate = Contains(resourceNameList, validateVmSecurityGroupName)
		if !checkOscAssociate {
			return fmt.Errorf("%s securityGroup does not exist in vm", validateVmSecurityGroupName)
		}
	}
	return nil
}

// checkVmSubnetOscAssociateResourceName check that Subnet dependencies tag name in both resource configuration are the same.
func checkVmSubnetOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmSubnetName := vmSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	machineScope.V(2).Info("Check match subnet with vm")
	checkOscAssociate := Contains(resourceNameList, vmSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in vm", vmSubnetName)
	}
}

// checkVmPublicIpOscAssociateResourceName check that PublicIp dependencies tag name in both resource configuration are the same.
func checkVmPublicIpOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	if vmSpec.PublicIp {
		return nil
	}
	vmPublicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
	publicIpsSpec := clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	machineScope.V(2).Info("Check match publicip with vm on cluster")
	checkOscAssociate := Contains(resourceNameList, vmPublicIpName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s publicIp does not exist in vm", vmPublicIpName)
	}
}

// checkVmFormatParameters check Volume parameters format
func checkVmFormatParameters(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()
	vmTagName, err := tag.ValidateTagNameValue(vmName)
	if err != nil {
		return vmTagName, err
	}
	imageSpec := machineScope.GetImage()
	imageName := imageSpec.Name

	if imageName != "" {
		_, err = infrastructurev1beta1.ValidateImageName(imageName)
		if err != nil {
			return vmTagName, err
		}
	} else {
		_, err = infrastructurev1beta1.ValidateImageId(vmSpec.ImageId)
		if err != nil {
			return vmTagName, err
		}
	}

	vmKeypairName := vmSpec.KeypairName
	_, err = infrastructurev1beta1.ValidateKeypairName(vmKeypairName)
	if err != nil {
		return vmTagName, err
	}

	vmType := vmSpec.VmType
	_, err = infrastructurev1beta1.ValidateVmType(vmType)
	if err != nil {
		return vmTagName, err
	}

	vmDeviceName := vmSpec.DeviceName
	_, err = infrastructurev1beta1.ValidateDeviceName(vmDeviceName)
	if err != nil {
		return vmTagName, err
	}
	if vmSpec.VolumeDeviceName != "" {
		vmVolumeDeviceName := vmSpec.VolumeDeviceName
		_, err = infrastructurev1beta1.ValidateDeviceName(vmVolumeDeviceName)
		if err != nil {
			return vmTagName, err
		}
	}

	vmSubregionName := vmSpec.SubregionName
	_, err = infrastructurev1beta1.ValidateSubregionName(vmSubregionName)
	if err != nil {
		return vmTagName, err
	}

	vmSubnetName := vmSpec.SubnetName
	machineScope.V(4).Info("Get vmSubnetName", "vmSubnetName", vmSubnetName)
	ipSubnetRange := clusterScope.GetIpSubnetRange(vmSubnetName)
	vmPrivateIps := machineScope.GetVmPrivateIps()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	subnetsSpec = networkSpec.Subnets
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name
		machineScope.V(4).Info("Get SubnetName", "subnetName", subnetName)
		ipSubnetRange := subnetSpec.IpSubnetRange
		machineScope.V(4).Info("Get IpSubnetRange", "ipSubnetRange", ipSubnetRange)
	}
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIp
		machineScope.V(4).Info("Get Valid IP", "privateIp", privateIp)
		_, err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return vmTagName, err
		}
	}

	if vmSpec.RootDisk.RootDiskIops != 0 {
		rootDiskIops := vmSpec.RootDisk.RootDiskIops
		machineScope.V(4).Info("Check rootDiskIops", "rootDiskIops", rootDiskIops)
		_, err := infrastructurev1beta1.ValidateIops(rootDiskIops)
		if err != nil {
			return vmTagName, err
		}
	}
	rootDiskSize := vmSpec.RootDisk.RootDiskSize
	machineScope.V(4).Info("Check rootDiskSize", "rootDiskSize", rootDiskSize)
	_, err = infrastructurev1beta1.ValidateSize(rootDiskSize)
	if err != nil {
		return vmTagName, err
	}

	rootDiskType := vmSpec.RootDisk.RootDiskType
	machineScope.V(4).Info("Check rootDiskType", "rootDiskTyp", rootDiskType)
	_, err = infrastructurev1beta1.ValidateVolumeType(rootDiskType)
	if err != nil {
		return vmTagName, err
	}

	if vmSpec.RootDisk.RootDiskType == "io1" && vmSpec.RootDisk.RootDiskIops != 0 && vmSpec.RootDisk.RootDiskSize != 0 {
		ratioRootDiskSizeIops := vmSpec.RootDisk.RootDiskIops / vmSpec.RootDisk.RootDiskSize
		machineScope.V(4).Info("Check ratio rootdisk size iops", "ratioRootDiskSizeIops", ratioRootDiskSizeIops)
		_, err = infrastructurev1beta1.ValidateRatioSizeIops(ratioRootDiskSizeIops)
		if err != nil {
			return vmTagName, err
		}
	}
	return "", nil
}

// checkVmPrivateIpOscDuplicateName check that there are not the same name for vm resource
func checkVmPrivateIpOscDuplicateName(machineScope *scope.MachineScope) error {
	var resourceNameList []string
	vmPrivateIps := machineScope.GetVmPrivateIps()
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIpName := vmPrivateIp.Name
		resourceNameList = append(resourceNameList, privateIpName)
	}
	machineScope.V(2).Info("Check unique privateIp")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcileVm reconcile the vm of the machine
func reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, volumeSvc storage.OscVolumeInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	vmSpec := machineScope.GetVm()
	vmRef := machineScope.GetVmRef()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()

	var volumeId string
	var err error
	if vmSpec.VolumeName != "" {
		volumeName := vmSpec.VolumeName + "-" + machineScope.GetUID()
		volumeId, err = getVolumeResourceId(volumeName, machineScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	subnetName := vmSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	var publicIpId string
	var vmPublicIpName string
	var linkPublicIpRef *infrastructurev1beta1.OscResourceReference
	if vmSpec.PublicIp {
		vmSpec.PublicIpName = vmSpec.Name + "-publicIp"
		vmPublicIpName = vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		var ipFound bool
		publicIpIdRef := machineScope.GetPublicIpIdRef()
		publicIpId, ipFound = publicIpIdRef.ResourceMap[vmPublicIpName]
		if !ipFound {
			publicIp, err := publicIpSvc.CreatePublicIp(vmPublicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create publicIp for Vm %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.V(4).Info("Get publicIp for Vm", "publicip", publicIp)
			publicIpId = publicIp.GetPublicIpId()

			if len(publicIpIdRef.ResourceMap) == 0 {
				publicIpIdRef.ResourceMap = make(map[string]string)
			}
			publicIpIdRef.ResourceMap[vmPublicIpName] = publicIpId
		}
	}
	if vmSpec.PublicIpName != "" {
		vmPublicIpName = vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		if publicIpId == "" {
			publicIpId, err = getPublicIpResourceId(vmPublicIpName, clusterScope)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		linkPublicIpRef = machineScope.GetLinkPublicIpRef()
		if len(linkPublicIpRef.ResourceMap) == 0 {
			linkPublicIpRef.ResourceMap = make(map[string]string)
		}
	}
	var privateIps []string
	vmPrivateIps := machineScope.GetVmPrivateIps()
	if len(*vmPrivateIps) > 0 {
		for _, vmPrivateIp := range *vmPrivateIps {
			privateIp := vmPrivateIp.PrivateIp
			privateIps = append(privateIps, privateIp)
		}
	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		machineScope.V(4).Info("Get vmSecurityGroup", "vmSecurityGroup", vmSecurityGroup)
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		machineScope.V(4).Info("Get securityGroupId", "securityGroupId", securityGroupId)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}

	var vmVolumeDeviceName string
	if vmSpec.VolumeDeviceName != "" {
		vmVolumeDeviceName = vmSpec.VolumeDeviceName
	}
	var vm *osc.Vm
	var vmID string
	if len(vmRef.ResourceMap) == 0 {
		vmRef.ResourceMap = make(map[string]string)
	}
	tagKey := "Name"
	tagValue := vmName
	tag, err := tagSvc.ReadTag(tagKey, tagValue)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
	}
	machineScope.V(4).Info("Get ResourceId", "resourceId", vmSpec.ResourceId)
	machineScope.V(4).Info("Get ResourceMap", "resourceMap", vmRef.ResourceMap)
	if vmSpec.ResourceId != "" {
		vmRef.ResourceMap[vmName] = vmSpec.ResourceId
		vmId := vmSpec.ResourceId
		machineScope.V(4).Info("Check if the desired vm exist", "vmName", vmName)
		vm, err = vmSvc.GetVm(vmId)
		if err != nil {
			return reconcile.Result{}, err
		}
		clusterName := vmSpec.ClusterName + "-" + clusterScope.GetUID()
		privateDnsName, ok := vm.GetPrivateDnsNameOk()
		if !ok {
			return reconcile.Result{}, fmt.Errorf("Can not found privateDnsName %s/%s", machineScope.GetNamespace(), machineScope.GetName())
		}
		privateIp, ok := vm.GetPrivateIpOk()
		if !ok {
			return reconcile.Result{}, fmt.Errorf("Can not found privateIp %s/%s", machineScope.GetNamespace(), machineScope.GetName())
		}
		addresses := []corev1.NodeAddress{}
		addresses = append(
			addresses,
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: *privateIp,
			},
		)
		machineScope.SetAddresses(addresses)
		err = vmSvc.AddCcmTag(clusterName, *privateDnsName, vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w can not add ccm tag %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}
		vmState, err := vmSvc.GetVmState(vmId)
		if err != nil {
			machineScope.SetVmState(infrastructurev1beta1.VmState("unknown"))
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s state for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.SetVmState(infrastructurev1beta1.VmState(vmState))
		machineScope.V(4).Info("Get vm state", "vmState", vmState)
	}
	if (vm == nil && tag == nil) || (vmSpec.ResourceId == "" && tag == nil) {
		machineScope.V(4).Info("Create the desired vm", "vmName", vmName)
		imageId := vmSpec.ImageId
		machineScope.V(4).Info("Info ImageId", "imageId", imageId)
		keypairName := vmSpec.KeypairName
		machineScope.V(4).Info("Info keypairName", "keypairName", keypairName)
		vmType := vmSpec.VmType
		machineScope.V(4).Info("Info vmType", "vmType", vmType)

		vm, err := vmSvc.CreateVm(machineScope, vmSpec, subnetId, securityGroupIds, privateIps, vmName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}

		vmID = vm.GetVmId()
		err = vmSvc.CheckVmState(20, 240, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.V(4).Info("Vm is running", "vmId", vmID)
		machineScope.SetVmState(infrastructurev1beta1.VmState("pending"))
		if vmSpec.VolumeName != "" {
			err = volumeSvc.CheckVolumeState(20, 240, "available", volumeId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s available for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.V(4).Info("Volume is available", "volumeId", volumeId)
			err = volumeSvc.LinkVolume(volumeId, vmID, vmVolumeDeviceName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link volume %s with vm %s for OscMachine %s/%s", err, volumeId, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.V(4).Info("Volume is linked", "volumeId", volumeId)
			err = volumeSvc.CheckVolumeState(20, 240, "in-use", volumeId)
			machineScope.V(4).Info("Volume is in-use", "volumeId", volumeId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s in use for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
			}
		}
		err = vmSvc.CheckVmState(20, 240, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.V(4).Info("Vm is running again", "vmId", vmID)

		if vmSpec.PublicIpName != "" {
			linkPublicIpId, err := publicIpSvc.LinkPublicIp(publicIpId, vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link publicIp  %s with %s for OscCluster %s/%s", err, publicIpId, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.V(4).Info("Link public ip", "linkPublicIpId", linkPublicIpId)
			linkPublicIpRef.ResourceMap[vmPublicIpName] = linkPublicIpId

			err = vmSvc.CheckVmState(20, 240, "running", vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
		}
		if vmSpec.LoadBalancerName != "" {
			loadBalancerName := vmSpec.LoadBalancerName
			vmIds := []string{vmID}
			err := loadBalancerSvc.LinkLoadBalancerBackendMachines(vmIds, loadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link vm %s with loadBalancerName %s for OscCluster %s/%s", err, loadBalancerName, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			machineScope.V(4).Info("Get IpProtocol", "IpProtocol", ipProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			machineScope.V(4).Info("Get fromPortRange", "fromPortRange", fromPortRange)
			toPortRange := loadBalancerSpec.Listener.BackendPort
			machineScope.V(4).Info("Get ToPortRange", "ToPortRange", toPortRange)
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
			associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
			machineScope.V(4).Info("Get sg", "associateSecurityGroupId", associateSecurityGroupId)
			securityGroupFromSecurityGroupOutboundRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get outbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			if securityGroupFromSecurityGroupOutboundRule == nil {
				_, err = securityGroupSvc.CreateSecurityGroupRule(associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not create outbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}
			}
			securityGroupFromSecurityGroupInboundRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(securityGroupIds[0], "Inbound", ipProtocol, "", associateSecurityGroupId, fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get inbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			if securityGroupFromSecurityGroupInboundRule == nil {
				_, err = securityGroupSvc.CreateSecurityGroupRule(securityGroupIds[0], "Inbound", ipProtocol, "", associateSecurityGroupId, fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not create inbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}
			}

		}

		machineScope.V(2).Info("Get Vm", "vm", vm)
		vmRef.ResourceMap[vmName] = vmID
		vmSpec.ResourceId = vmID
		subregionName := vmSpec.SubregionName
		machineScope.SetProviderID(subregionName, vmID)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmId := vmSpec.ResourceId
	machineScope.V(4).Info("Get VmID", "vmId", vmId)
	vmName := vmSpec.Name
	if vmSpec.ResourceId == "" {
		machineScope.V(2).Info("The desired vm is currently destroyed", "vmName", vmName)
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, nil
	}
	keypairSpec := machineScope.GetKeypair()
	machineScope.V(4).Info("Check keypair", "keypair", keypairSpec.Name)
	deleteKeypair := machineScope.GetDeleteKeypair()

	vm, err := vmSvc.GetVm(vmId)
	if err != nil {
		return reconcile.Result{}, err
	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	if vmSpec.PublicIpName != "" {
		linkPublicIpRef := machineScope.GetLinkPublicIpRef()
		publicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		err = publicIpSvc.UnlinkPublicIp(linkPublicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink publicIp for OscCluster %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}
	}
	if vmSpec.PublicIp {
		publicIpIdRef := machineScope.GetPublicIpIdRef()
		publicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Delete the desired Vm publicip", "publicIpName", publicIpName)
		err = publicIpSvc.DeletePublicIp(publicIpIdRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete Vm publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	if vmSpec.LoadBalancerName != "" {
		vmIds := []string{vmId}
		loadBalancerName := vmSpec.LoadBalancerName
		err := loadBalancerSvc.UnlinkLoadBalancerBackendMachines(vmIds, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink vm %s with loadBalancerName %s for OscCluster %s/%s", err, loadBalancerName, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		clusterScope.V(2).Info("Get list OscMachine")
		var machineSize int
		var machineKcpCount int32
		var machineKwCount int32
		var machineCount int32

		var machines []*clusterv1.Machine
		if vmSpec.Replica != 1 {
			machines, _, err = clusterScope.ListMachines(ctx)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not get ListMachine", err)
			}
			machineSize = len(machines)
			clusterScope.V(4).Info("Get info OscMachine", "machineSize", machineSize)
		} else {
			machineSize = 1
			machineKcpCount = 1
			machineCount = 1

		}

		if machineSize > 0 {
			if vmSpec.Replica != 1 {
				clusterScope.V(2).Info("Get  MachineList")
				names := make([]string, len(machines))
				for i, m := range machines {
					names[i] = fmt.Sprintf("machine/%s", m.Name)
					machineScope.V(4).Info("Get Machines", "machine", m.Name)
					machineLabel := m.Labels
					for labelKey := range machineLabel {
						if labelKey == "cluster.x-k8s.io/control-plane" {
							machineScope.V(4).Info("Get Kcp Machine", "machineKcp", m.Name)
							machineKcpCount++
						}
						if labelKey == "cluster.x-k8s.io/deployment-name" {
							machineScope.V(4).Info("Get Kw Machine", "machineKw", m.Name)
							machineKwCount++
						}

					}
					machineCount = machineKwCount + machineKcpCount
				}
			}
			if machineCount != 1 {
				machineScope.SetDeleteKeypair(false)
				machineScope.V(2).Info("Keep Keypair from vm")
			}
			if machineKcpCount == 1 {
				machineScope.SetDeleteKeypair(deleteKeypair)
				securityGroupsRef := clusterScope.GetSecurityGroupsRef()
				loadBalancerSpec := clusterScope.GetLoadBalancer()
				loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
				ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
				machineScope.V(4).Info("Get IpProtocol", "ipProtocol", ipProtocol)
				fromPortRange := loadBalancerSpec.Listener.BackendPort
				machineScope.V(4).Info("Get FromPortRange", "FromPortRange", fromPortRange)
				toPortRange := loadBalancerSpec.Listener.BackendPort
				machineScope.V(4).Info("Get ToPortRange", "ToPortRange", toPortRange)
				loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
				associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
				machineScope.V(4).Info("Get associate", "AssociateSecurityGroupId", associateSecurityGroupId)
				machineScope.V(4).Info("Get sg id", "securityGroupIds", securityGroupIds[0])
				machineScope.V(2).Info("Delete LoadBalancer sg")
				err = securityGroupSvc.DeleteSecurityGroupRule(associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not delete outbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}
				err = securityGroupSvc.DeleteSecurityGroupRule(securityGroupIds[0], "Inbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not delete inbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}

			} else {
				machineScope.V(2).Info("Get several control plane machine, can not delete loadBalancer securityGroup", "machineKcp", machineKcpCount)
			}
		}
	}

	if vm == nil {
		machineScope.V(2).Info("The desired vm does not exist anymore", "vmName", vmName)
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, nil
	}

	err = vmSvc.DeleteVm(vmId)
	vmSpec.ResourceId = ""
	machineScope.V(2).Info("Delete the desired vm", "vmName", vmName)
	if err != nil {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("%w Can not delete vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
	}
	return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
}
