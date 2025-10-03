/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/utils/ptr"
)

const (
	AutoAttachExternapIPTag = "osc.fcu.eip.auto-attach"

	TagKeyNodeName        = "OscK8sNodeName"
	TagKeyClusterIDPrefix = "OscK8sClusterID/"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(ctx context.Context, machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, imageId, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken string, tags map[string]string, volumes []infrastructurev1beta1.OscVolume) (*osc.Vm, error)
	CreateVmBastion(ctx context.Context, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken, imageId string, tags map[string]string) (*osc.Vm, error)
	DeleteVm(ctx context.Context, vmId string) error
	GetVm(ctx context.Context, vmId string) (*osc.Vm, error)
	GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error)
	AddCCMTags(ctx context.Context, clusterName string, hostname string, vmId string) error
}

// CreateVm creates a VM.
func (s *Service) CreateVm(ctx context.Context,
	machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, imageId, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken string, tags map[string]string,
	volumes []infrastructurev1beta1.OscVolume) (*osc.Vm, error) {
	keypairName := spec.KeypairName
	vmType := spec.VmType
	rootDiskIops := spec.RootDisk.RootDiskIops
	rootDiskSize := spec.RootDisk.RootDiskSize
	rootDiskType := spec.RootDisk.RootDiskType
	bootstrapData, err := machineScope.GetBootstrapData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bootstrap data: %w", err)
	}
	mergedUserData := utils.ConvertsTagsToUserDataOutscaleSection(tags) + bootstrapData
	mergedUserDataEnc := b64.StdEncoding.EncodeToString([]byte(mergedUserData))
	rootDisk := osc.BlockDeviceMappingVmCreation{
		Bsu: &osc.BsuToCreate{
			VolumeType: &rootDiskType,
			VolumeSize: &rootDiskSize,
		},
		DeviceName: ptr.To("/dev/sda1"),
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.Iops = &rootDiskIops
	}
	volMappings := []osc.BlockDeviceMappingVmCreation{
		rootDisk,
	}
	for _, vol := range volumes {
		bsuVol := osc.BlockDeviceMappingVmCreation{
			Bsu: &osc.BsuToCreate{
				VolumeSize: &vol.Size,
				VolumeType: &vol.VolumeType,
			},
			DeviceName: &vol.Device,
		}
		if vol.VolumeType == "io1" {
			bsuVol.Bsu.Iops = &vol.Iops
		}
		if vol.FromSnapshot != "" {
			bsuVol.Bsu.SnapshotId = &vol.FromSnapshot
		}
		volMappings = append(volMappings, bsuVol)
	}

	vmOpt := osc.CreateVmsRequest{
		ImageId:             imageId,
		KeypairName:         &keypairName,
		VmType:              &vmType,
		SubnetId:            &subnetId,
		SecurityGroupIds:    &securityGroupIds,
		UserData:            &mergedUserDataEnc,
		BlockDeviceMappings: &volMappings,
		ClientToken:         &vmClientToken,
	}

	if len(privateIps) > 0 {
		vmOpt.SetPrivateIps(privateIps)
	}

	vmResponse, httpRes, err := s.tenant.Client().VmApi.CreateVms(s.tenant.ContextWithAuth(ctx)).CreateVmsRequest(vmOpt).Execute()
	err = utils.LogAndExtractError(ctx, "CreateVms", vmOpt, httpRes, err)
	if err != nil {
		return nil, err
	}
	vms, ok := vmResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("cannot get vm")
	}
	vmID := *(*vmResponse.Vms)[0].VmId
	resourceIds := []string{vmID}
	vmTag := osc.ResourceTag{
		Key:   "Name",
		Value: vmName,
	}
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{vmTag},
	}
	err = tag.AddTag(ctx, vmTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// CreateVmBastion create a bastion vm
func (s *Service) CreateVmBastion(ctx context.Context, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken, imageId string, tags map[string]string) (*osc.Vm, error) {
	keypairName := spec.KeypairName
	vmType := spec.VmType
	rootDiskIops := spec.RootDisk.RootDiskIops
	rootDiskSize := spec.RootDisk.RootDiskSize
	rootDiskType := spec.RootDisk.RootDiskType

	userDataEnc := b64.StdEncoding.EncodeToString([]byte(utils.ConvertsTagsToUserDataOutscaleSection(tags)))
	rootDisk := osc.BlockDeviceMappingVmCreation{
		Bsu: &osc.BsuToCreate{
			VolumeType: &rootDiskType,
			VolumeSize: &rootDiskSize,
		},
		DeviceName: ptr.To("/dev/sda1"),
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.Iops = &rootDiskIops
	}

	vmOpt := osc.CreateVmsRequest{
		ImageId:          imageId,
		KeypairName:      &keypairName,
		VmType:           &vmType,
		SubnetId:         &subnetId,
		SecurityGroupIds: &securityGroupIds,
		UserData:         &userDataEnc,
		BlockDeviceMappings: &[]osc.BlockDeviceMappingVmCreation{
			rootDisk,
		},
		ClientToken: &vmClientToken,
	}

	if len(privateIps) > 0 {
		vmOpt.SetPrivateIps(privateIps)
	}

	vmResponse, httpRes, err := s.tenant.Client().VmApi.CreateVms(s.tenant.ContextWithAuth(ctx)).CreateVmsRequest(vmOpt).Execute()
	err = utils.LogAndExtractError(ctx, "CreateVms", vmOpt, httpRes, err)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	vms, ok := vmResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("cannot get vm")
	}
	vmID := *(*vmResponse.Vms)[0].VmId
	resourceIds := []string{vmID}
	vmTag := osc.ResourceTag{
		Key:   "Name",
		Value: vmName,
	}
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{vmTag},
	}
	err = tag.AddTag(ctx, vmTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// DeleteVm delete machine vm
func (s *Service) DeleteVm(ctx context.Context, vmId string) error {
	deleteVmsRequest := osc.DeleteVmsRequest{VmIds: []string{vmId}}

	_, httpRes, err := s.tenant.Client().VmApi.DeleteVms(s.tenant.ContextWithAuth(ctx)).DeleteVmsRequest(deleteVmsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteVms", deleteVmsRequest, httpRes, err)
	return err
}

// GetVm retrieve vm from vmId
func (s *Service) GetVm(ctx context.Context, vmId string) (*osc.Vm, error) {
	readVmsRequest := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			VmIds: &[]string{vmId},
		},
	}

	readVmsResponse, httpRes, err := s.tenant.Client().VmApi.ReadVms(s.tenant.ContextWithAuth(ctx)).ReadVmsRequest(readVmsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadVmsRequest", readVmsRequest, httpRes, err)
	if err != nil {
		return nil, err
	}

	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("cannot get vm")
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// GetVmFromClientToken retrieve vm from vmId
func (s *Service) GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error) {
	readVmsRequest := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			ClientTokens: &[]string{clientToken},
		},
	}

	readVmsResponse, httpRes, err := s.tenant.Client().VmApi.ReadVms(s.tenant.ContextWithAuth(ctx)).ReadVmsRequest(readVmsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadVms", readVmsRequest, httpRes, err)
	if err != nil {
		return nil, err
	}

	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("cannot get vm")
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// HasCCMTags checks if a Vm has both CCM tags.
func HasCCMTags(vm *osc.Vm) bool {
	return slices.ContainsFunc(vm.GetTags(), func(t osc.ResourceTag) bool {
		return t.Key == TagKeyNodeName
	}) && slices.ContainsFunc(vm.GetTags(), func(t osc.ResourceTag) bool {
		return strings.HasPrefix(t.Key, TagKeyClusterIDPrefix)
	})
}

// AddCCMTags add ccm tag
func (s *Service) AddCCMTags(ctx context.Context, clusterName string, hostname string, vmId string) error {
	resourceIds := []string{vmId}

	nodeTag := osc.ResourceTag{
		Key:   TagKeyNodeName,
		Value: hostname,
	}
	clusterTag := osc.ResourceTag{
		Key:   TagKeyClusterIDPrefix + clusterName,
		Value: "owned",
	}
	nodeTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag, clusterTag},
	}
	return tag.AddTag(ctx, nodeTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
}
