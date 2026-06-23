/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"
	b64 "encoding/base64"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

const (
	AutoAttachExternalIPTag = "osc.fcu.eip.auto-attach"

	RepulseServerTag        = "osc.fcu.repulse_server"
	RepulseClusterTag       = "osc.fcu.repulse_cluster"
	RepulseServerStrictTag  = "osc.fcu.repulse_server_strict"
	RepulseClusterStrictTag = "osc.fcu.repulse_cluster_strict"

	AttractServerTag        = "osc.fcu.attract_server"
	AttractClusterTag       = "osc.fcu.attract_cluster"
	AttractServerStrictTag  = "osc.fcu.attract_server_strict"
	AttractClusterStrictTag = "osc.fcu.attract_cluster_strict"

	TagKeyNodeName        = "OscK8sNodeName"
	TagKeyClusterIDPrefix = "OscK8sClusterID/"
)

type VmInterface interface {
	CreateVm(ctx context.Context,
		machineScope *scope.MachineScope, spec *infrastructurev1beta2.OscVm, imageId, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken string, tags map[string]string,
		volumes []infrastructurev1beta2.OscVolume,
	) (*osc.Vm, error)
	CreateVmBastion(ctx context.Context, spec *infrastructurev1beta2.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken, imageId string, tags map[string]string) (*osc.Vm, error)
	DeleteVm(ctx context.Context, vmId string) error
	GetVm(ctx context.Context, vmId string) (*osc.Vm, error)
	GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error)
	AddCCMTags(ctx context.Context, clusterName string, hostname string, vmId string) error
	StartVm(ctx context.Context, vmId string) error
	StopVm(ctx context.Context, vmId string) error
}

func (s *Service) CreateVm(ctx context.Context,
	machineScope *scope.MachineScope, spec *infrastructurev1beta2.OscVm, imageId, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken string, tags map[string]string,
	volumes []infrastructurev1beta2.OscVolume,
) (*osc.Vm, error) {
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
			VolumeSize: new(int(rootDiskSize)),
		},
		DeviceName: new("/dev/sda1"),
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.Iops = new(int(rootDiskIops))
	}
	volMappings := []osc.BlockDeviceMappingVmCreation{
		rootDisk,
	}
	for _, vol := range volumes {
		bsuVol := osc.BlockDeviceMappingVmCreation{
			Bsu: &osc.BsuToCreate{
				VolumeType: &vol.VolumeType,
			},
			DeviceName: &vol.Device,
		}
		if vol.Size > 0 {
			bsuVol.Bsu.VolumeSize = new(int(vol.Size))
		}
		if vol.VolumeType == "io1" {
			bsuVol.Bsu.Iops = new(int(vol.Iops))
		}
		if vol.FromSnapshot != "" {
			bsuVol.Bsu.SnapshotId = &vol.FromSnapshot
		}
		volMappings = append(volMappings, bsuVol)
	}

	req := osc.CreateVmsRequest{
		ImageId:             imageId,
		KeypairName:         &keypairName,
		VmType:              &vmType,
		SubnetId:            &subnetId,
		SecurityGroupIds:    securityGroupIds,
		UserData:            &mergedUserDataEnc,
		BlockDeviceMappings: volMappings,
		ClientToken:         &vmClientToken,
	}

	if spec.FGPU != nil {
		req.BootOnCreation = new(false)
	}

	if len(privateIps) > 0 {
		req.PrivateIps = privateIps
	}

	resp, err := s.tenant.Client().CreateVms(ctx, req)
	if err != nil {
		return nil, err
	}
	vmID := (*resp.Vms)[0].VmId
	resourceIds := []string{vmID}
	vmTag := osc.ResourceTag{
		Key:   "Name",
		Value: vmName,
	}
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{vmTag},
	}
	err = s.tags.AddTag(ctx, vmTagRequest, resourceIds)
	if err != nil {
		return nil, err
	}
	if len(*resp.Vms) == 0 {
		return nil, nil
	} else {
		return &(*resp.Vms)[0], nil
	}
}

// CreateVmBastion create a bastion vm
func (s *Service) CreateVmBastion(ctx context.Context, spec *infrastructurev1beta2.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken, imageId string, tags map[string]string) (*osc.Vm, error) {
	keypairName := spec.KeypairName
	vmType := spec.VmType
	rootDiskIops := spec.RootDisk.RootDiskIops
	rootDiskSize := spec.RootDisk.RootDiskSize
	rootDiskType := spec.RootDisk.RootDiskType

	userDataEnc := b64.StdEncoding.EncodeToString([]byte(utils.ConvertsTagsToUserDataOutscaleSection(tags)))
	rootDisk := osc.BlockDeviceMappingVmCreation{
		Bsu: &osc.BsuToCreate{
			VolumeType: &rootDiskType,
			VolumeSize: new(int(rootDiskSize)),
		},
		DeviceName: new("/dev/sda1"),
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.Iops = new(int(rootDiskIops))
	}

	vmOpt := osc.CreateVmsRequest{
		ImageId:          imageId,
		KeypairName:      &keypairName,
		VmType:           &vmType,
		SubnetId:         &subnetId,
		SecurityGroupIds: securityGroupIds,
		UserData:         &userDataEnc,
		BlockDeviceMappings: []osc.BlockDeviceMappingVmCreation{
			rootDisk,
		},
		ClientToken: &vmClientToken,
	}

	if len(privateIps) > 0 {
		vmOpt.PrivateIps = privateIps
	}

	resp, err := s.tenant.Client().CreateVms(ctx, vmOpt)
	if err != nil {
		return nil, err
	}
	vmID := (*resp.Vms)[0].VmId
	resourceIds := []string{vmID}
	vmTag := osc.ResourceTag{
		Key:   "Name",
		Value: vmName,
	}
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{vmTag},
	}
	err = s.tags.AddTag(ctx, vmTagRequest, resourceIds)
	if err != nil {
		return nil, err
	}
	if len(*resp.Vms) == 0 {
		return nil, nil
	} else {
		return &(*resp.Vms)[0], nil
	}
}

// DeleteVm delete machine vm
func (s *Service) DeleteVm(ctx context.Context, vmId string) error {
	req := osc.DeleteVmsRequest{VmIds: []string{vmId}}
	_, err := s.tenant.Client().DeleteVms(ctx, req)
	return err
}

// GetVm retrieve vm from vmId
func (s *Service) GetVm(ctx context.Context, vmId string) (*osc.Vm, error) {
	req := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			VmIds: &[]string{vmId},
		},
	}

	resp, err := s.tenant.Client().ReadVms(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Vms) == 0:
		return nil, nil
	default:
		return &(*resp.Vms)[0], nil
	}
}

// GetVmFromClientToken retrieve vm from vmId
func (s *Service) GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error) {
	req := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			ClientTokens: &[]string{clientToken},
		},
	}

	resp, err := s.tenant.Client().ReadVms(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Vms) == 0:
		return nil, nil
	default:
		return &(*resp.Vms)[0], nil
	}
}

// StartVm starts a VM
func (s *Service) StartVm(ctx context.Context, vmId string) error {
	req := osc.StartVmsRequest{
		VmIds: []string{vmId},
	}
	_, err := s.tenant.Client().StartVms(ctx, req)
	return err
}

// StopVm stops a VM
func (s *Service) StopVm(ctx context.Context, vmId string) error {
	req := osc.StopVmsRequest{
		VmIds: []string{vmId},
	}
	_, err := s.tenant.Client().StopVms(ctx, req)
	return err
}

// HasCCMTags checks if a Vm has both CCM tags.
func HasCCMTags(vm *osc.Vm) bool {
	return tags.Has(vm.Tags, tags.VmNodeName)
}

// AddCCMTags add ccm tag
func (s *Service) AddCCMTags(ctx context.Context, clusterName string, hostname string, vmId string) error {
	resourceIds := []string{vmId}

	nodeTag := osc.ResourceTag{
		Key:   tags.VmNodeName,
		Value: hostname,
	}
	clusterTag := osc.ResourceTag{
		Key:   tags.ClusterIDKey(clusterName),
		Value: tags.ResourceLifecycleOwned,
	}
	nodeTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag, clusterTag},
	}
	return s.tags.AddTag(ctx, nodeTagRequest, resourceIds)
}
