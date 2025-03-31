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
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
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

// checkVmLoadBalancerOscAssociateResourceName  check that LoadBalancer dependencies tag name in both resource configuration are the same.
func checkVmLoadBalancerOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmLoadBalancerName := vmSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName + "-" + clusterScope.GetUID()
	resourceNameList = append(resourceNameList, loadBalancerName)
	checkOscAssociate := slices.Contains(resourceNameList, vmLoadBalancerName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s loadBalancer does not exist in vm", vmLoadBalancerName)
	}
}

// checkVmSecurityGroupOscAssociateResourceName check that SecurityGroup dependencies tag name in both resource configuration are the same.
func checkVmSecurityGroupOscAssociateResourceName(machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var vmSecurityGroupNameList []string
	var checkOscAssociate bool
	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range vmSecurityGroups {
		vmSecurityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		vmSecurityGroupNameList = append(vmSecurityGroupNameList, vmSecurityGroupName)
	}
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	for _, validateVmSecurityGroupName := range vmSecurityGroupNameList {
		checkOscAssociate = slices.Contains(resourceNameList, validateVmSecurityGroupName)
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
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := slices.Contains(resourceNameList, vmSubnetName)
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
	checkOscAssociate := slices.Contains(resourceNameList, vmPublicIpName)
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
		err = infrastructurev1beta1.ValidateImageName(imageName)
		if err != nil {
			return vmTagName, err
		}
	} else {
		err = infrastructurev1beta1.ValidateImageId(vmSpec.ImageId)
		if err != nil {
			return vmTagName, err
		}
	}

	vmKeypairName := vmSpec.KeypairName
	err = infrastructurev1beta1.ValidateKeypairName(vmKeypairName)
	if err != nil {
		return vmTagName, err
	}

	vmType := vmSpec.VmType
	err = infrastructurev1beta1.ValidateVmType(vmType)
	if err != nil {
		return vmTagName, err
	}

	vmSubregionName := vmSpec.SubregionName
	err = infrastructurev1beta1.ValidateSubregionName(vmSubregionName)
	if err != nil {
		return vmTagName, err
	}

	vmSubnetName := vmSpec.SubnetName
	ipSubnetRange := clusterScope.GetIpSubnetRange(vmSubnetName)
	vmPrivateIps := machineScope.GetVmPrivateIps()
	networkSpec := clusterScope.GetNetwork()
	networkSpec.SetSubnetDefaultValue()
	for _, vmPrivateIp := range vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIp
		err := compute.ValidateIpAddrInCidr(privateIp, ipSubnetRange)
		if err != nil {
			return vmTagName, err
		}
	}

	if vmSpec.RootDisk.RootDiskIops != 0 {
		rootDiskIops := vmSpec.RootDisk.RootDiskIops
		err := infrastructurev1beta1.ValidateIops(rootDiskIops)
		if err != nil {
			return vmTagName, err
		}
	}
	rootDiskSize := vmSpec.RootDisk.RootDiskSize
	err = infrastructurev1beta1.ValidateSize(rootDiskSize)
	if err != nil {
		return vmTagName, err
	}

	rootDiskType := vmSpec.RootDisk.RootDiskType
	err = infrastructurev1beta1.ValidateVolumeType(rootDiskType)
	if err != nil {
		return vmTagName, err
	}

	if vmSpec.RootDisk.RootDiskType == "io1" && vmSpec.RootDisk.RootDiskIops != 0 && vmSpec.RootDisk.RootDiskSize != 0 {
		ratioRootDiskSizeIops := vmSpec.RootDisk.RootDiskIops / vmSpec.RootDisk.RootDiskSize
		err = infrastructurev1beta1.ValidateRatioSizeIops(ratioRootDiskSizeIops)
		if err != nil {
			return vmTagName, err
		}
	}
	return "", nil
}

// checkVmPrivateIpOscDuplicateName check that there are not the same name for vm resource
func checkVmPrivateIpOscDuplicateName(machineScope *scope.MachineScope) error {
	return utils.CheckDuplicates(machineScope.GetVmPrivateIps(), func(ip infrastructurev1beta1.OscPrivateIpElement) string {
		return ip.Name
	})
}

func UseFailureDomain(clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	if machineScope.Machine.Spec.FailureDomain != nil && machineScope.GetVm().SubnetName == "" {
		machineScope.GetVm().SubnetName = *machineScope.Machine.Spec.FailureDomain

		subnetName := machineScope.GetVm().SubnetName + "-" + clusterScope.GetUID()
		subnetSpecs := clusterScope.GetSubnets()
		for _, subnetSpec := range subnetSpecs {
			if subnetSpec.Name+"-"+clusterScope.GetUID() == subnetName {
				machineScope.GetVm().SubregionName = subnetSpec.SubregionName
			}
		}
	}
}

// reconcileVm reconcile the vm of the machine
func (r *OscMachineReconciler) reconcileVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	vmSpec := machineScope.GetVm()
	vmRef := machineScope.GetVmRef()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()

	subnetSpec, err := clusterScope.GetSubnet(vmSpec.SubnetName, vmSpec.Role, vmSpec.SubregionName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile vm: %w")
	}
	subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile vm: %w")
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
			publicIp, err := publicIpSvc.CreatePublicIp(ctx, vmPublicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create publicIp for Vm : %w", err)
			}
			log.V(4).Info("Get publicIp for Vm", "publicip", publicIp)
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
	vmPrivateIps := machineScope.GetVmPrivateIps()
	privateIps := make([]string, 0, len(vmPrivateIps))
	for _, vmPrivateIp := range vmPrivateIps {
		privateIp := vmPrivateIp.PrivateIp
		privateIps = append(privateIps, privateIp)
	}

	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	securityGroupIds := make([]string, 0, len(vmSecurityGroups))
	for _, vmSecurityGroup := range vmSecurityGroups {
		securityGroupName := vmSecurityGroup.Name + "-" + clusterScope.GetUID()
		securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
		log.V(4).Info("Get securityGroupId", "securityGroupId", securityGroupId)
		if err != nil {
			return reconcile.Result{}, err
		}
		securityGroupIds = append(securityGroupIds, securityGroupId)
	}

	if len(vmRef.ResourceMap) == 0 {
		vmRef.ResourceMap = make(map[string]string)
	}

	var vm *osc.Vm
	switch {
	case vmSpec.ResourceId != "":
		vm, err = vmSvc.GetVm(ctx, vmSpec.ResourceId)
		if err != nil {
			return reconcile.Result{}, err
		}
	default:
		vms, err := vmSvc.GetVmListFromTag(ctx, "Name", vmName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot list VMs: %w", err)
		}
		if len(vms) > 0 {
			vm = &vms[0]
		}
	}
	if vm != nil {
		vmSpec.ResourceId = vm.GetVmId()
		vmRef.ResourceMap[vmName] = vmSpec.ResourceId
		machineScope.SetVmState(infrastructurev1beta1.VmState(vm.GetState()))
	} else {
		imageId := vmSpec.ImageId
		keypairName := vmSpec.KeypairName
		vmType := vmSpec.VmType
		vmTags := vmSpec.Tags
		log.V(3).Info("Creating VM", "vmName", vmName, "imageId", imageId, "keypairName", keypairName, "vmType", vmType, "tags", vmTags)
		volumes := machineScope.GetVolume()

		vm, err = vmSvc.CreateVm(ctx, machineScope, vmSpec, subnetId, securityGroupIds, privateIps, vmName, vmTags, volumes)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create vm: %w", err)
		}

		vmId := vm.GetVmId()
		machineScope.SetVmState(infrastructurev1beta1.VmStatePending)
		vmRef.ResourceMap[vmName] = vmId
		machineScope.SetVmID(vmId)
		subregionName := vmSpec.SubregionName
		machineScope.SetProviderID(subregionName, vmId)
		log.V(2).Info("VM created", "vmId", vmId)
	}

	vmState := machineScope.GetVmState()

	switch {
	case vmState == nil:
		// should never occur
		return reconcile.Result{}, errors.New("no vm state")
	case vm == nil:
		// should never occur
		return reconcile.Result{}, errors.New("no vm")
	case *vmState != infrastructurev1beta1.VmStateRunning:
		return reconcile.Result{}, fmt.Errorf("VM %s is not yet running", vm.GetVmId())
	}

	vmId := vm.GetVmId()

	if vmSpec.PublicIpName != "" && linkPublicIpRef.ResourceMap[vmPublicIpName] == "" {
		log.V(2).Info("Linking public ip", "publicIpId", publicIpId)
		linkPublicIpId, err := publicIpSvc.LinkPublicIp(ctx, publicIpId, vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot link publicIp %s with %s: %w", publicIpId, vmId, err)
		}
		log.V(4).Info("Linked public ip", "linkPublicIpId", linkPublicIpId)
		linkPublicIpRef.ResourceMap[vmPublicIpName] = linkPublicIpId
	}

	if vmSpec.LoadBalancerName != "" {
		loadBalancerName := vmSpec.LoadBalancerName
		vmIds := []string{vmId}
		log.V(2).Info("Linking loadbalancer", "loadBalancerName", loadBalancerName)
		err := loadBalancerSvc.LinkLoadBalancerBackendMachines(ctx, vmIds, loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot link vm %s with loadBalancerName %s: %w", vmId, loadBalancerName, err)
		}
		securityGroupsRef := clusterScope.GetSecurityGroupsRef()
		loadBalancerSpec := clusterScope.GetLoadBalancer()
		loadBalancerSpec.SetDefaultValue()
		loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
		ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
		fromPortRange := loadBalancerSpec.Listener.BackendPort
		toPortRange := loadBalancerSpec.Listener.BackendPort
		loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
		lbSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
		hasOutboundRule, err := securityGroupSvc.SecurityGroupHasRule(ctx, lbSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get outbound securityGroupRule: %w", err)
		}
		// FIXME: those rules should probably be created by the OscCluster.
		if !hasOutboundRule {
			log.V(2).Info("Creating outbound securityGroup rule")
			_, err = securityGroupSvc.CreateSecurityGroupRule(ctx, lbSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create outbound securityGroupRule: %w", err)
			}
		}
		hasInboundRule, err := securityGroupSvc.SecurityGroupHasRule(ctx, securityGroupIds[0], "Inbound", ipProtocol, "", lbSecurityGroupId, fromPortRange, toPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get inbound securityGroupRule: %w", err)
		}
		if !hasInboundRule {
			log.V(2).Info("Creating inbound securityGroup rule")
			_, err = securityGroupSvc.CreateSecurityGroupRule(ctx, securityGroupIds[0], "Inbound", ipProtocol, "", lbSecurityGroupId, fromPortRange, toPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create inbound securityGroupRule: %w", err)
			}
		}
	}

	clusterName := vmSpec.ClusterName + "-" + clusterScope.GetUID()
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

	// expose volumes
	if vm.BlockDeviceMappings != nil {
		volRef := machineScope.GetVolumeRef()
		if volRef.ResourceMap == nil {
			volRef.ResourceMap = map[string]string{}
		}
		for _, mapping := range *vm.BlockDeviceMappings {
			volRef.ResourceMap[mapping.GetDeviceName()] = mapping.Bsu.GetVolumeId()
		}
	}

	tag, err := tagSvc.ReadTag(ctx, tag.VmResourceType, "OscK8sNodeName", *privateDnsName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
	}
	if tag == nil {
		log.V(2).Info("Adding CCM tags")
		err = vmSvc.AddCcmTag(ctx, clusterName, *privateDnsName, vmId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot add ccm tag: %w", err)
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteVm reconcile the destruction of the vm of the machine
func reconcileDeleteVm(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, vmSvc compute.OscVmInterface, publicIpSvc security.OscPublicIpInterface, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	vmSpec := machineScope.GetVm()
	vmRef := machineScope.GetVmRef()
	vmName := vmSpec.Name + "-" + machineScope.GetUID()
	vmSpec.SetDefaultValue()
	vmId := vmSpec.ResourceId
	if vmId == "" { // We should not get in this situation but we sometimes do (To be investigated)
		vmId = vmRef.ResourceMap[vmName]
		machineScope.SetVmID(vmId)
	}
	if vmId == "" {
		log.V(2).Info("vm is already destroyed", "vmName", vmName)
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Get vmId", "vmId", vmId)
	vm, err := vmSvc.GetVm(ctx, vmId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if vm == nil {
		log.V(2).Info("vm is already destroyed", "vmName", vmName)
		return reconcile.Result{}, nil
	}

	var securityGroupIds []string
	vmSecurityGroups := machineScope.GetVmSecurityGroups()
	for _, vmSecurityGroup := range vmSecurityGroups {
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
		linkPublicIiId := linkPublicIpRef.ResourceMap[publicIpName]
		if linkPublicIiId != "" {
			err = publicIpSvc.UnlinkPublicIp(ctx, linkPublicIiId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink publicIp: %w", err)
			}
		}
	}
	if vmSpec.PublicIp {
		publicIpIdRef := machineScope.GetPublicIpIdRef()
		publicIpName := vmSpec.PublicIpName + "-" + clusterScope.GetUID()
		log.V(2).Info("Deleting Vm publicip", "publicIpName", publicIpName)
		err = publicIpSvc.DeletePublicIp(ctx, publicIpIdRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete Vm publicIp %s: %w", publicIpName, err)
		}
	}
	if vmSpec.LoadBalancerName != "" {
		lb, err := loadBalancerSvc.GetLoadBalancer(ctx, vmSpec.LoadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot unlink loadBalancer %s: %w", vmSpec.LoadBalancerName, err)
		}
		if lb.BackendVmIds != nil && slices.Contains(*lb.BackendVmIds, vmId) {
			err = loadBalancerSvc.UnlinkLoadBalancerBackendMachines(ctx, []string{vmId}, vmSpec.LoadBalancerName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink loadBalancer %s: %w", vmSpec.LoadBalancerName, err)
			}
		}

		// TODO: this cleanups resources from the pool of machines. this should probably move to machinetemplate or cluster.
		log.V(2).Info("Get list OscMachine")
		var machineSize int
		var machineKcpCount int32
		var machineKwCount int32

		var machines []*clusterv1.Machine
		if vmSpec.Replica != 1 {
			machines, _, err = clusterScope.ListMachines(ctx)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot call ListMachine: %w", err)
			}
			machineSize = len(machines)
			log.V(4).Info("Get info OscMachine", "machineSize", machineSize)
		} else {
			machineSize = 1
			machineKcpCount = 1
		}

		if machineSize > 0 {
			if vmSpec.Replica != 1 {
				for _, m := range machines {
					log.V(4).Info("Get Machines", "machine", m.Name)
					machineLabel := m.Labels
					for labelKey := range machineLabel {
						if labelKey == "cluster.x-k8s.io/control-plane" {
							log.V(4).Info("Get Kcp Machine", "machineKcp", m.Name)
							machineKcpCount++
						}
						if labelKey == "cluster.x-k8s.io/deployment-name" {
							log.V(4).Info("Get Kw Machine", "machineKw", m.Name)
							machineKwCount++
						}
					}
				}
			}
			if machineKcpCount == 1 {
				securityGroupsRef := clusterScope.GetSecurityGroupsRef()
				loadBalancerSpec := clusterScope.GetLoadBalancer()
				loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
				ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
				fromPortRange := loadBalancerSpec.Listener.BackendPort
				toPortRange := loadBalancerSpec.Listener.BackendPort
				loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-" + clusterScope.GetUID()
				associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
				log.V(2).Info("Deleting LoadBalancer sg outbound rule")
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, associateSecurityGroupId, "Outbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete outbound securityGroupRule: %w", err)
				}
				log.V(2).Info("Deleting LoadBalancer sg inbound rule")
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroupIds[0], "Inbound", ipProtocol, "", securityGroupIds[0], fromPortRange, toPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete inbound securityGroupRule: %w", err)
				}
			} else {
				log.V(3).Info("Get several control plane machine, can not delete loadBalancer securityGroup", "machineKcp", machineKcpCount)
			}
		}
	}

	if vm == nil {
		log.V(3).Info("vm is already deleted", "vmName", vmName)
		return reconcile.Result{}, nil
	}

	log.V(2).Info("Deleting vm", "vmName", vmName)
	err = vmSvc.DeleteVm(ctx, vmId)
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
