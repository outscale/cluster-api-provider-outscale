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

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getVmResourceId return the vmId from the resourceMap base on resourceName (tag name + cluster uid)
func getVmResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	vmRef := machineScope.GetVMRef()
	if vmId, ok := vmRef.ResourceMap[resourceName]; ok {
		return vmId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkVMVolumeOscAssociateResourceName check that Volume dependancies tag name in both resource configuration are the same.
func checkVMVolumeOscAssociateResourceName(machineScope *scope.MachineScope) error {
	var resourceNameList []string
	machineScope.Info("check match volume with vm")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmVolumeName := vmSpec.VolumeName + "-" + machineScope.GetUID()
	volumesSpec := machineScope.GetVolume()
	for _, volumeSpec := range volumesSpec {
		volumeName := volumeSpec.Name + "-" + machineScope.GetUID()
		resourceNameList = append(resourceNameList, volumeName)
	}
	checkOscAssociate := Contains(resourceNameList, vmVolumeName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s volume does not exist in vm", vmVolumeName)
	}
}

// checkVMLoadBalancerOscAssociateResourceName  check that LoadBalancer dependancies tag name in both resource configuration are the same.
func checkVMLoadBalancerOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match loadbalancer with vm")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmLoadBalancerName := vmSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	machineScope.Info("Get LoadBalancerName", "loadBalancerName", loadBalancerName)
	machineScope.Info("Get VmLoadBalancerName", "vmLoadBalancerName", vmLoadBalancerName)
	machineScope.Info("Get Role", "Role", vmSpec.Role)
	resourceNameList = append(resourceNameList, loadBalancerName)
	checkOscAssociate := Contains(resourceNameList, vmLoadBalancerName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s loadBalancer does not exist in vm", vmLoadBalancerName)
	}

}

func checkVMVolumeSubregionName(machineScope *scope.MachineScope) error {
	machineScope.Info("check have the same subregionName for vm and for volume")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmVolumeName := vmSpec.VolumeName
	volumeSubregionName := machineScope.GetVolumeSubregionName(vmVolumeName)
	vmSubregionName := vmSpec.SubregionName
	vmName := vmSpec.Name
	if vmSubregionName != volumeSubregionName {
		return fmt.Errorf("volume %s and vm %s are not in the same subregion %s", vmVolumeName, vmName, vmSubregionName)
	} else {
		return nil
	}
}

// checkVMSecurityGroupOscAssociateResourceName check that SecurityGroup dependancies tag name in both resource configuration are the same.
func checkVMSecurityGroupOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var vmSecurityGroupNameList []string
	var checkOscAssociate bool
	machineScope.Info("check match securityGroup with vm")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmSecurityGroups := machineScope.GetVMSecurityGroups()
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
		checkOscAssociate = Contains(resourceNameList, validateVmSecurityGroupName)
		if !checkOscAssociate {
			return fmt.Errorf("%s securityGroup does not exist in vm", validateVmSecurityGroupName)
		}
	}
	return nil
}

// checkVMSubnetOscAssociateResourceName check that Subnet dependancies tag name in both resource configuration are the same.
func checkVMSubnetOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match subnet with vm")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmSubnetName := vmSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := Contains(resourceNameList, vmSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in vm", vmSubnetName)
	}
}

// checkVMPublicIPOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func checkVMPublicIPOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match publicip with vm")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmPublicIPName := vmSpec.PublicIPName + "-" + clusterScope.GetUID()
	publicIpsSpec := clusterScope.GetPublicIP()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := Contains(resourceNameList, vmPublicIPName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s publicIp does not exist in vm", vmPublicIPName)
	}
}

// checkVMFormatParameters check Volume parameters format
func checkVMFormatParameters(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	machineScope.Info("Check Vm parameters")
	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()
	vmTagName, err := tag.ValidateTagNameValue(vmName)
	if err != nil {
		return vmTagName, err
	}
	vmImageID := vmSpec.ImageID
	_, err = infrastructurev1beta1.ValidateImageID(vmImageID)
	if err != nil {
		return vmTagName, err
	}

	vmKeypairName := vmSpec.KeypairName
	_, err = infrastructurev1beta1.ValidateKeypairName(vmKeypairName)
	if err != nil {
		return vmTagName, err
	}

	vmType := vmSpec.VMType
	_, err = infrastructurev1beta1.ValidateVMType(vmType)
	if err != nil {
		return vmTagName, err
	}

	vmDeviceName := vmSpec.DeviceName
	_, err = infrastructurev1beta1.ValidateDeviceName(vmDeviceName)
	if err != nil {
		return vmTagName, err
	}

	vmSubregionName := vmSpec.SubregionName
	_, err = infrastructurev1beta1.ValidateSubregionName(vmSubregionName)
	if err != nil {
		return vmTagName, err
	}

	vmSubnetName := vmSpec.SubnetName
	machineScope.Info("Get vmSubnetName", "vmSubnetName", vmSubnetName)
	ipSubnetRange := clusterScope.GetIPSubnetRange(vmSubnetName)
	vmPrivateIps := machineScope.GetVMPrivateIPS()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	subnetsSpec = networkSpec.Subnets
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name
		ipSubnetRange := subnetSpec.IPSubnetRange
		machineScope.Info("Get IPSubnetRange", "ipSubnetRange", ipSubnetRange)
		machineScope.Info("Get SubnetName", "subnetName", subnetName)
	}
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIP
		machineScope.Info("### Get Valid IP", "privateIp", privateIp)
		machineScope.Info("### Get Valiid subnet ###", "ipSubnetRange", ipSubnetRange)
		_, err := compute.ValidateIPAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return vmTagName, err
		}
	}

	return "", nil
}

// checkVMPrivateIPOscDuplicateName check that there are not the same name for vm resource
func checkVMPrivateIPOscDuplicateName(machineScope *scope.MachineScope) error {
	machineScope.Info("check unique privateIp")
	var resourceNameList []string
	vmPrivateIps := machineScope.GetVMPrivateIPS()
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIpName := vmPrivateIp.Name
		resourceNameList = append(resourceNameList, privateIpName)
	}
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcileVm reconcile the vm of the machine
func reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVMInterface, volumeSvc storage.OscVolumeInterface, publicIpSvc security.OscPublicIPInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	machineScope.Info("Create Vm")
	vmSpec := machineScope.GetVM()
	vmRef := machineScope.GetVMRef()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()

	volumeName := vmSpec.VolumeName + "-" + machineScope.GetUID()
	volumeId, err := getVolumeResourceID(volumeName, machineScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := vmSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceID(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	var publicIpId string
	var vmPublicIPName string
	var linkPublicIpRef *infrastructurev1beta1.OscResourceMapReference
	if vmSpec.PublicIPName != "" {
		vmPublicIPName = vmSpec.PublicIPName + "-" + clusterScope.GetUID()
		publicIpId, err = getPublicIPResourceID(vmPublicIPName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		linkPublicIpRef = machineScope.GetLinkPublicIPRef()
		if len(linkPublicIpRef.ResourceMap) == 0 {
			linkPublicIpRef.ResourceMap = make(map[string]string)
		}
	}
	var privateIps []string
	vmPrivateIps := machineScope.GetVMPrivateIPS()
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIP
		privateIps = append(privateIps, privateIp)

	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVMSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		machineScope.Info("Get vmSecurityGroup", "vmSecurityGroup", vmSecurityGroup)
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceID(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		machineScope.Info("get securityGroupId", "securityGroupId", securityGroupId)
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}

	vmDeviceName := vmSpec.DeviceName

	var vm *osc.Vm
	var vmID string
	if len(vmRef.ResourceMap) == 0 {
		vmRef.ResourceMap = make(map[string]string)
	}
	machineScope.Info("### Get ResourceId ###", "resourceId", vmSpec.ResourceID)
	machineScope.Info("### Get ResourceMap ###", "resourceMap", vmRef.ResourceMap)
	if vmSpec.ResourceID != "" {
		vmRef.ResourceMap[vmName] = vmSpec.ResourceID
		vmId := vmSpec.ResourceID
		machineScope.Info("Check if the desired vm exist", "vmName", vmName)
		machineScope.Info("### Get VmId ####", "vm", vmRef.ResourceMap)
		vm, err = vmSvc.GetVM(vmId)
		if err != nil {
			return reconcile.Result{}, err
		}
		vmState, err := vmSvc.GetVMState(vmId)
		if err != nil {
			machineScope.SetVMState(infrastructurev1beta1.VMState("unknown"))
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s state for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.SetVMState(infrastructurev1beta1.VMState(vmState))
		machineScope.Info("Get vm state", "vmState", vmState)
	}
	if vm == nil || vmSpec.ResourceID == "" {
		machineScope.Info("Create the desired vm", "vmName", vmName)
		imageId := vmSpec.ImageID
		keypairName := vmSpec.KeypairName
		vmType := vmSpec.VMType
		machineScope.Info("### Info ImageID ###", "imageId", imageId)
		machineScope.Info("### Info keypairName ###", "keypairName", keypairName)
		machineScope.Info("### Info vmType ####", "vmType", vmType)

		vm, err := vmSvc.CreateVM(machineScope, vmSpec, subnetId, securityGroupIds, privateIps, vmName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}

		vmID = vm.GetVmId()
		err = vmSvc.CheckVMState(5, 120, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Vm is running", "vmId", vmID)

		err = volumeSvc.CheckVolumeState(5, 60, "available", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s available for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Volume is available", "volumeId", volumeId)

		machineScope.SetVMState(infrastructurev1beta1.VMState("pending"))
		err = volumeSvc.LinkVolume(volumeId, vmID, vmDeviceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not link volume %s with vm %s for OscMachine %s/%s", err, volumeId, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}

		machineScope.Info("Volume is linked", "volumeId", volumeId)
		err = volumeSvc.CheckVolumeState(5, 60, "in-use", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s in use for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Volume is in-use", "volumeId", volumeId)

		err = vmSvc.CheckVMState(5, 60, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Vm is running again", "vmId", vmID)

		if vmSpec.PublicIPName != "" {
			linkPublicIpId, err := publicIpSvc.LinkPublicIP(publicIpId, vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link publicIp  %s with %s for OscCluster %s/%s", err, publicIpId, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.Info("Link public ip", "linkPublicIpId", linkPublicIpId)
			linkPublicIpRef.ResourceMap[vmPublicIPName] = linkPublicIpId

			err = vmSvc.CheckVMState(5, 60, "running", vmID)
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
			machineScope.Info("Create LoadBalancer Sg")
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			machineScope.Info("Get IpProtocol", "IpProtocol", ipProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			machineScope.Info("Get fromPortRange", "fromPortRange", fromPortRange)
			toPortRange := loadBalancerSpec.Listener.BackendPort
			machineScope.Info("Get ToPortRange", "ToPortRange", toPortRange)
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
			associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
			machineScope.Info("Get sg", "associateSecurityGroupId", associateSecurityGroupId)
			_, err = securityGroupSvc.CreateSecurityGroupRule(associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%s Can not create outbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			_, err = securityGroupSvc.CreateSecurityGroupRule(securityGroupIds[0], "Inbound", ipProtocol, "", associateSecurityGroupId, fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create inbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
		}

		machineScope.Info("#### Get Vm ###", "vm", vm)
		vmRef.ResourceMap[vmName] = vmID
		vmSpec.ResourceID = vmID
		machineScope.SetProviderID(vmID)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVMInterface, publicIpSvc security.OscPublicIPInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine
	machineScope.Info("Delete vm")

	vmSpec := machineScope.GetVM()
	vmSpec.SetDefaultValue()

	vmId := vmSpec.ResourceID
	machineScope.Info("### VmiD ###", "vmId", vmId)
	vmName := vmSpec.Name
	vm, err := vmSvc.GetVM(vmId)
	if err != nil {
		return reconcile.Result{}, err
	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVMSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceID(securityGroupName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}
	if vm == nil {
		machineScope.Info("The desired vm does not exist anymore", "vmName", vmName)
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, nil
	}
	if vmSpec.PublicIPName != "" {
		linkPublicIpRef := machineScope.GetLinkPublicIPRef()
		publicIpName := vmSpec.PublicIPName + "-" + clusterScope.GetUID()
		err = vmSvc.CheckVMState(5, 120, "running", vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		err = publicIpSvc.UnlinkPublicIP(linkPublicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink publicIp for OscCluster %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}

	}
	if vmSpec.LoadBalancerName != "" {
		err = vmSvc.CheckVMState(5, 60, "running", vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		vmIds := []string{vmId}
		loadBalancerName := vmSpec.LoadBalancerName
		err := loadBalancerSvc.UnlinkLoadBalancerBackendMachines(vmIds, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink vm %s with loadBalancerName %s for OscCluster %s/%s", err, loadBalancerName, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Delete LoadBalancer sg")
		securityGroupsRef := clusterScope.GetSecurityGroupsRef()
		loadBalancerSpec := clusterScope.GetLoadBalancer()
		loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
		ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
		machineScope.Info("Get IpProtocol", "ipProtocol", ipProtocol)
		fromPortRange := loadBalancerSpec.Listener.BackendPort
		machineScope.Info("Get FromPortRange", "FromPortRange", fromPortRange)
		toPortRange := loadBalancerSpec.Listener.BackendPort
		machineScope.Info("Get ToPortRange", "ToPortRange", toPortRange)
		loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
		associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
		machineScope.Info("Get associate", "AssociateSecurityGroupId", associateSecurityGroupId)
		machineScope.Info("Get sg id", "securityGroupIds", securityGroupIds[0])
		err = securityGroupSvc.DeleteSecurityGroupRule(associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete outbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		err = securityGroupSvc.DeleteSecurityGroupRule(securityGroupIds[0], "Inbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete inbound securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}

	err = vmSvc.DeleteVM(vmId)
	machineScope.Info("Delete the desired vm", "vmName", vmName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
	}
	return reconcile.Result{}, nil
}
