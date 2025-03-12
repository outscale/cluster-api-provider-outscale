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

package compute

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"net"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/utils/ptr"
)

const (
	AutoAttachExternapIPTag = "osc.fcu.eip.auto-attach"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(ctx context.Context, machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, imageId, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken string, tags map[string]string, volumes []infrastructurev1beta1.OscVolume) (*osc.Vm, error)
	CreateVmBastion(ctx context.Context, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName, vmClientToken, imageId string, tags map[string]string) (*osc.Vm, error)
	DeleteVm(ctx context.Context, vmId string) error
	GetVm(ctx context.Context, vmId string) (*osc.Vm, error)
	GetVmFromClientToken(ctx context.Context, clientToken string) (*osc.Vm, error)
	AddCcmTag(ctx context.Context, clusterName string, hostname string, vmId string) error
}

// ValidateIpAddrInCidr check that ipaddr is in cidr
func ValidateIpAddrInCidr(ipAddr, cidr string) error {
	_, ipnet, _ := net.ParseCIDR(cidr)
	ip := net.ParseIP(ipAddr)
	if ipnet.Contains(ip) {
		return nil
	} else {
		return errors.New("ip is not in subnet")
	}
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

	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	vmResponse, httpRes, err := oscApiClient.VmApi.CreateVms(oscAuthClient).CreateVmsRequest(vmOpt).Execute()
	utils.LogAPICall(ctx, "CreateVms", vmOpt, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	vms, ok := vmResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("Can not get vm")
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
	err = tag.AddTag(ctx, vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
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

	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	vmResponse, httpRes, err := oscApiClient.VmApi.CreateVms(oscAuthClient).CreateVmsRequest(vmOpt).Execute()
	utils.LogAPICall(ctx, "CreateVms", vmOpt, httpRes, err)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	vms, ok := vmResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("Can not get vm")
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
	err = tag.AddTag(ctx, vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VmApi.DeleteVms(oscAuthClient).DeleteVmsRequest(deleteVmsRequest).Execute()
	utils.LogAPICall(ctx, "DeleteVms", deleteVmsRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// GetVm retrieve vm from vmId
func (s *Service) GetVm(ctx context.Context, vmId string) (*osc.Vm, error) {
	readVmsRequest := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			VmIds: &[]string{vmId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readVmsResponse, httpRes, err := oscApiClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
	utils.LogAPICall(ctx, "ReadVmsRequest", readVmsRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}

	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("Can not get vm")
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readVmsResponse, httpRes, err := oscApiClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
	utils.LogAPICall(ctx, "ReadVms", readVmsRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}

	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("Can not get vm")
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// AddCcmTag add ccm tag
func (s *Service) AddCcmTag(ctx context.Context, clusterName string, hostname string, vmId string) error {
	resourceIds := []string{vmId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	nodeTag := osc.ResourceTag{
		Key:   "OscK8sNodeName",
		Value: hostname,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	nodeTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag, clusterTag},
	}
	return tag.AddTag(ctx, nodeTagRequest, resourceIds, oscApiClient, oscAuthClient)
}
