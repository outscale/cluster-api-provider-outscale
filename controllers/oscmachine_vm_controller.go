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
	machineScope.Info("check match volume with vm")
	vmSpec := machineScope.GetVm()
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

// checkVmLoadBalancerOscAssociateResourceName  check that LoadBalancer dependancies tag name in both resource configuration are the same.
func checkVmLoadBalancerOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match loadbalancer with vm")
	vmSpec := machineScope.GetVm()
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

func checkVmVolumeSubregionName(machineScope *scope.MachineScope) error {
	machineScope.Info("check have the same subregionName for vm and for volume")
	vmSpec := machineScope.GetVm()
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

// checkVmSecurityGroupOscAssociateResourceName check that SecurityGroup dependancies tag name in both resource configuration are the same.
func checkVmSecurityGroupOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var vmSecurityGroupNameList []string
	var checkOscAssociate bool
	machineScope.Info("check match securityGroup with vm")
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
		checkOscAssociate = Contains(resourceNameList, validateVmSecurityGroupName)
		if !checkOscAssociate {
			return fmt.Errorf("%s securityGroup does not exist in vm", validateVmSecurityGroupName)
		}
	}
	return nil
}

// checkVmSubnetOscAssociateResourceName check that Subnet dependancies tag name in both resource configuration are the same.
func checkVmSubnetOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match subnet with vm")
	vmSpec := machineScope.GetVm()
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

// checkVmPublicIpOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func checkVmPublicIpOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	machineScope.Info("check match publicip with vm")
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmPublicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
	publicIpsSpec := clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := Contains(resourceNameList, vmPublicIpName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s publicIp does not exist in vm", vmPublicIpName)
	}
}

//checkVmFormatParameters check Volume parameters format
func checkVmFormatParameters(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (string, error) {
	machineScope.Info("Check Vm parameters")
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()
	vmTagName, err := tag.ValidateTagNameValue(vmName)
	if err != nil {
		return vmTagName, err
	}
	vmImageId := vmSpec.ImageId
	_, err = infrastructurev1beta1.ValidateImageId(vmImageId)
	if err != nil {
		return vmTagName, err
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

	vmSubregionName := vmSpec.SubregionName
	_, err = infrastructurev1beta1.ValidateSubregionName(vmSubregionName)
	if err != nil {
		return vmTagName, err
	}

	vmSubnetName := vmSpec.SubnetName
	machineScope.Info("Get vmSubnetName", "vmSubnetName", vmSubnetName)
	ipSubnetRange := clusterScope.GetIpSubnetRange(vmSubnetName)
	vmPrivateIps := machineScope.GetVmPrivateIps()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	subnetsSpec = networkSpec.Subnets
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name
		ipSubnetRange := subnetSpec.IpSubnetRange
		machineScope.Info("Get IpSubnetRange", "ipSubnetRange", ipSubnetRange)
		machineScope.Info("Get SubnetName", "subnetName", subnetName)
	}
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIp
		machineScope.Info("### Get Valid IP", "privateIp", privateIp)
		machineScope.Info("### Get Valiid subnet ###", "ipSubnetRange", ipSubnetRange)
		_, err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return vmTagName, err
		}
	}

	return "", nil
}

// checkVmPrivateIpOscDuplicateName check that there are not the same name for vm resource
func checkVmPrivateIpOscDuplicateName(machineScope *scope.MachineScope) error {
	machineScope.Info("check unique privateIp")
	var resourceNameList []string
	vmPrivateIps := machineScope.GetVmPrivateIps()
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
func reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, volumeSvc storage.OscVolumeInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	machineScope.Info("Create Vm")
	vmSpec := machineScope.GetVm()
	vmRef := machineScope.GetVmRef()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()

	volumeName := vmSpec.VolumeName + "-" + machineScope.GetUID()
	volumeId, err := getVolumeResourceId(volumeName, machineScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := vmSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	var publicIpId string
	var vmPublicIpName string
	var linkPublicIpRef *infrastructurev1beta1.OscResourceMapReference
	if vmSpec.PublicIpName != "" {
		vmPublicIpName = vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		publicIpId, err = getPublicIpResourceId(vmPublicIpName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		linkPublicIpRef = machineScope.GetLinkPublicIpRef()
		if len(linkPublicIpRef.ResourceMap) == 0 {
			linkPublicIpRef.ResourceMap = make(map[string]string)
		}
	}
	var privateIps []string
	vmPrivateIps := machineScope.GetVmPrivateIps()
	for _, vmPrivateIp := range *vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIp
		privateIps = append(privateIps, privateIp)

	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range *vmSecurityGroups {
		machineScope.Info("Get vmSecurityGroup", "vmSecurityGroup", vmSecurityGroup)
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
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
	machineScope.Info("### Get ResourceId ###", "resourceId", vmSpec.ResourceId)
	machineScope.Info("### Get ResourceMap ###", "resourceMap", vmRef.ResourceMap)
	if vmSpec.ResourceId != "" {
		vmRef.ResourceMap[vmName] = vmSpec.ResourceId
		vmId := vmSpec.ResourceId
		machineScope.Info("Check if the desired vm exist", "vmName", vmName)
		machineScope.Info("### Get VmId ####", "vm", vmRef.ResourceMap)
		vm, err = vmSvc.GetVm(vmId)
		if err != nil {
			return reconcile.Result{}, err
		}
		vmState, err := vmSvc.GetVmState(vmId)
		if err != nil {
			machineScope.SetVmState(infrastructurev1beta1.VmState("unknown"))
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s state for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.SetVmState(infrastructurev1beta1.VmState(vmState))
		machineScope.Info("Get vm state", "vmState", vmState)
	}
	if vm == nil || vmSpec.ResourceId == "" {
		machineScope.Info("Create the desired vm", "vmName", vmName)
		imageId := vmSpec.ImageId
		keypairName := vmSpec.KeypairName
		vmType := vmSpec.VmType
		machineScope.Info("### Info ImageId ###", "imageId", imageId)
		machineScope.Info("### Info keypairName ###", "keypairName", keypairName)
		machineScope.Info("### Info vmType ####", "vmType", vmType)

		vm, err := vmSvc.CreateVm(machineScope, vmSpec, subnetId, securityGroupIds, privateIps, vmName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}

		vmID = vm.GetVmId()
		err = vmSvc.CheckVmState(5, 120, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Vm is running", "vmId", vmID)

		err = volumeSvc.CheckVolumeState(5, 60, "available", volumeId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get volume %s available for OscMachine %s/%s", err, volumeId, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Volume is available", "volumeId", volumeId)

		machineScope.SetVmState(infrastructurev1beta1.VmState("pending"))
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

		err = vmSvc.CheckVmState(5, 60, "running", vmID)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmID, machineScope.GetNamespace(), machineScope.GetName())
		}
		machineScope.Info("Vm is running again", "vmId", vmID)

		if vmSpec.PublicIpName != "" {
			linkPublicIpId, err := publicIpSvc.LinkPublicIp(publicIpId, vmID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link publicIp  %s with %s for OscCluster %s/%s", err, publicIpId, vmID, machineScope.GetNamespace(), machineScope.GetName())
			}
			machineScope.Info("Link public ip", "linkPublicIpId", linkPublicIpId)
			linkPublicIpRef.ResourceMap[vmPublicIpName] = linkPublicIpId

			err = vmSvc.CheckVmState(5, 60, "running", vmID)
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
		vmSpec.ResourceId = vmID
		machineScope.SetProviderID(vmID)
	}

	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	oscmachine := machineScope.OscMachine
	machineScope.Info("Delete vm")

	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()

	vmId := vmSpec.ResourceId
	machineScope.Info("### VmiD ###", "vmId", vmId)
	vmName := vmSpec.Name
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
	if vm == nil {
		machineScope.Info("The desired vm does not exist anymore", "vmName", vmName)
		controllerutil.RemoveFinalizer(oscmachine, "")
		return reconcile.Result{}, nil
	}
	if vmSpec.PublicIpName != "" {
		linkPublicIpRef := machineScope.GetLinkPublicIpRef()
		publicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		err = vmSvc.CheckVmState(5, 120, "running", vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get vm %s running for OscMachine %s/%s", err, vmId, machineScope.GetNamespace(), machineScope.GetName())
		}
		err = publicIpSvc.UnlinkPublicIp(linkPublicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink publicIp for OscCluster %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
		}

	}
	if vmSpec.LoadBalancerName != "" {
		err = vmSvc.CheckVmState(5, 60, "running", vmId)
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

	err = vmSvc.DeleteVm(vmId)
	machineScope.Info("Delete the desired vm", "vmName", vmName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete vm for OscMachine %s/%s", err, machineScope.GetNamespace(), machineScope.GetName())
	}
	return reconcile.Result{}, nil
}
