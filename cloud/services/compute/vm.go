/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"
	b64 "encoding/base64"
	"maps"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"k8s.io/utils/ptr"
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

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(ctx context.Context, spec *infrastructurev1beta2.OscVm, bootstrapData, imageId, subnetId string, securityGroupIds []string, vmName, vmClientToken string, publicIp string) (*osc.Vm, error)
	DeleteVm(ctx context.Context, vmId string) error
	GetVm(ctx context.Context, vmId string) (*osc.Vm, error)
	GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error)
	AddCCMTags(ctx context.Context, clusterName string, hostname string, vmId string) error
	StartVm(ctx context.Context, vmId string) error
	StopVm(ctx context.Context, vmId string) error
}

func volumeToCreate(vol infrastructurev1beta2.OscVolume) *osc.BsuToCreate {
	v := &osc.BsuToCreate{
		VolumeType: &vol.Type,
	}
	if vol.Size > 0 {
		v.VolumeSize = &vol.Size
	}
	if vol.Type == osc.VolumeTypeIo1 {
		v.Iops = &vol.Iops
	}
	if vol.FromSnapshot != "" {
		v.SnapshotId = &vol.FromSnapshot
	}
	return v
}

// CreateVm creates a VM.
func (s *Service) CreateVm(ctx context.Context,
	spec *infrastructurev1beta2.OscVm, bootstrapData, imageId, subnetId string, securityGroupIds []string, vmName, vmClientToken string, publicIp string,
) (*osc.Vm, error) {
	vmTags := spec.Tags
	if vmTags == nil {
		vmTags = map[string]string{}
	} else {
		// we need to clone the map to avoid changing the spec...
		vmTags = maps.Clone(vmTags)
	}

	var publicIp string
	if publicIp != "" {
		vmTags[AutoAttachExternalIPTag] = publicIp
	}

	placement := spec.Placement
	switch {
	case placement.RepulseCluster != "" && placement.ClusterStrict:
		vmTags[RepulseClusterStrictTag] = placement.RepulseCluster
	case placement.RepulseCluster != "" && !placement.ClusterStrict:
		vmTags[RepulseClusterTag] = placement.RepulseCluster

	case placement.AttractCluster != "" && placement.ClusterStrict:
		vmTags[AttractClusterStrictTag] = placement.AttractCluster
	case placement.AttractCluster != "" && !placement.ClusterStrict:
		vmTags[AttractClusterTag] = placement.AttractCluster
	}
	switch {
	case placement.RepulseServer != nil && *placement.RepulseServer != "" && placement.ServerStrict:
		vmTags[RepulseServerStrictTag] = *placement.RepulseServer
	case placement.RepulseServer != nil && *placement.RepulseServer != "" && !placement.ServerStrict:
		vmTags[RepulseServerTag] = *placement.RepulseServer

	case placement.AttractServer != "" && placement.ServerStrict:
		vmTags[AttractServerStrictTag] = placement.AttractServer
	case placement.AttractServer != "" && !placement.ServerStrict:
		vmTags[AttractServerTag] = placement.AttractServer
	}
	mergedUserData := utils.ConvertsTagsToUserDataOutscaleSection(vmTags) + bootstrapData
	mergedUserDataEnc := b64.StdEncoding.EncodeToString([]byte(mergedUserData))
	volMappings := []osc.BlockDeviceMappingVmCreation{
		{
			Bsu:        volumeToCreate(spec.RootVolume),
			DeviceName: new("/dev/sda1"),
		},
	}
	for _, vol := range spec.AdditionalVolumes {
		volMappings = append(volMappings, osc.BlockDeviceMappingVmCreation{
			Bsu:        volumeToCreate(vol),
			DeviceName: &vol.Device,
		})
	}

	req := osc.CreateVmsRequest{
		ImageId:             imageId,
		KeypairName:         &spec.KeypairName,
		VmType:              &spec.VmType,
		SubnetId:            &subnetId,
		SecurityGroupIds:    securityGroupIds,
		UserData:            &mergedUserDataEnc,
		BlockDeviceMappings: volMappings,
		ClientToken:         &vmClientToken,
	}

	if spec.FGPU != nil {
		req.BootOnCreation = ptr.To(false)
	}

	resp, err := s.tenant.Client().CreateVms(ctx, req)
	if err != nil {
		return nil, err
	}
	vm := &(*resp.Vms)[0]
	resourceIds := []string{vm.VmId}
	vmTag := osc.ResourceTag{
		Key:   "Name",
		Value: vmName,
	}
	vmTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{vmTag},
	}
	err = tag.AddTag(ctx, vmTagRequest, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}
	return vm, nil
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
	return tag.AddTag(ctx, nodeTagRequest, resourceIds, s.tenant.Client())
}
