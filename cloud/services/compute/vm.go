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
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(ctx context.Context, machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string, tags map[string]string) (*osc.Vm, error)
	CreateVmUserData(ctx context.Context, userData string, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName string, imageId string) (*osc.Vm, error)
	DeleteVm(ctx context.Context, vmId string) error
	GetVm(ctx context.Context, vmId string) (*osc.Vm, error)
	GetVmListFromTag(ctx context.Context, tagKey string, tagName string) ([]osc.Vm, error)
	GetVmState(ctx context.Context, vmId string) (string, error)
	AddCcmTag(ctx context.Context, clusterName string, hostname string, vmId string) error
}

// ValidateIpAddrInCidr check that ipaddr is in cidr
func ValidateIpAddrInCidr(ipAddr string, cidr string) (string, error) {
	_, ipnet, _ := net.ParseCIDR(cidr)
	ip := net.ParseIP(ipAddr)
	if ipnet.Contains(ip) {
		return ipAddr, nil
	} else {
		return ipAddr, errors.New("Invalid ip in cidr")
	}
}

// CreateVm create machine vm
func (s *Service) CreateVm(ctx context.Context, machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string, tags map[string]string) (*osc.Vm, error) {
	imageId := spec.ImageId
	keypairName := spec.KeypairName
	vmType := spec.VmType
	subregionName := spec.SubregionName
	rootDiskIops := spec.RootDisk.RootDiskIops
	rootDiskSize := spec.RootDisk.RootDiskSize
	rootDiskType := spec.RootDisk.RootDiskType
	deviceName := spec.DeviceName

	placement := osc.Placement{
		SubregionName: &subregionName,
	}
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
		DeviceName: &deviceName,
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.SetIops(rootDiskIops)
	}

	vmOpt := osc.CreateVmsRequest{
		ImageId:          imageId,
		KeypairName:      &keypairName,
		VmType:           &vmType,
		SubnetId:         &subnetId,
		SecurityGroupIds: &securityGroupIds,
		UserData:         &mergedUserDataEnc,
		BlockDeviceMappings: &[]osc.BlockDeviceMappingVmCreation{
			rootDisk,
		},
		Placement: &placement,
	}

	if len(privateIps) > 0 {
		vmOpt.SetPrivateIps(privateIps)
	}

	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var vmResponse osc.CreateVmsResponse
	createVmCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		vmResponse, httpRes, err = oscApiClient.VmApi.CreateVms(oscAuthClient).CreateVmsRequest(vmOpt).Execute()
		utils.LogAPICall(ctx, "CreateVms", vmOpt, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", vmOpt)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createVmCallBack)
	if waitErr != nil {
		return nil, waitErr
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
	err, httpRes := tag.AddTag(ctx, vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vm := *vms
		return &vm[0], nil
	}
}

// CreateVmUserData create machine vm
func (s *Service) CreateVmUserData(ctx context.Context, userData string, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName string, imageId string) (*osc.Vm, error) {
	keypairName := spec.KeypairName
	vmType := spec.VmType
	subregionName := spec.SubregionName
	rootDiskIops := spec.RootDisk.RootDiskIops
	rootDiskSize := spec.RootDisk.RootDiskSize
	rootDiskType := spec.RootDisk.RootDiskType
	deviceName := spec.DeviceName

	placement := osc.Placement{
		SubregionName: &subregionName,
	}
	userDataEnc := b64.StdEncoding.EncodeToString([]byte(userData))
	rootDisk := osc.BlockDeviceMappingVmCreation{
		Bsu: &osc.BsuToCreate{
			VolumeType: &rootDiskType,
			VolumeSize: &rootDiskSize,
		},
		DeviceName: &deviceName,
	}
	if rootDiskType == "io1" {
		rootDisk.Bsu.SetIops(rootDiskIops)
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
		Placement: &placement,
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
	err, httpRes = tag.AddTag(ctx, vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
	deleteVmsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.VmApi.DeleteVms(oscAuthClient).DeleteVmsRequest(deleteVmsRequest).Execute()
		utils.LogAPICall(ctx, "DeleteVms", deleteVmsRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteVmsRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteVmsCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
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
	var readVmsResponse osc.ReadVmsResponse
	readVmsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readVmsResponse, httpRes, err = oscApiClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
		utils.LogAPICall(ctx, "ReadVmsRequest", readVmsRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readVmsRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readVmsCallBack)
	if waitErr != nil {
		return nil, waitErr
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

// GetVm retrieve vm from vmId
func (s *Service) GetVmListFromTag(ctx context.Context, tagKey string, tagValue string) ([]osc.Vm, error) {
	readVmsRequest := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			TagKeys:   &[]string{tagKey},
			TagValues: &[]string{tagValue},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readVmsResponse osc.ReadVmsResponse
	readVmsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readVmsResponse, httpRes, err = oscApiClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
		utils.LogAPICall(ctx, "ReadVms", readVmsRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readVmsRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readVmsCallBack)
	if waitErr != nil {
		return nil, waitErr
	}

	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("Can not get vm")
	}
	if len(*vms) == 0 {
		return nil, nil
	} else {
		vmList := *vms
		return vmList, nil
	}
}

// GetVmState return vm state
func (s *Service) GetVmState(ctx context.Context, vmId string) (string, error) {
	vm, err := s.GetVm(ctx, vmId)
	if err != nil {
		return "", err
	}
	vmState, ok := vm.GetStateOk()
	if !ok {
		return "", errors.New("cannot get vm state")
	}
	return *vmState, nil
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
	nodeTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag},
	}
	err, httpRes := tag.AddTag(ctx, nodeTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	clusterTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}

	err, httpRes = tag.AddTag(ctx, clusterTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}
