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
	b64 "encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/utils"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string, tags map[string]string) (*osc.Vm, error)
	CreateVmUserData(userData string, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName string, imageId string) (*osc.Vm, error)
	DeleteVm(vmId string) error
	GetVm(vmId string) (*osc.Vm, error)
	GetVmListFromTag(tagKey string, tagName string) ([]osc.Vm, error)
	GetVmState(vmId string) (string, error)
	AddCcmTag(clusterName string, hostname string, vmId string) error
	GetCapacity(tagKey string, tagValue string, vmType string) (corev1.ResourceList, error)
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
func (s *Service) CreateVm(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string, tags map[string]string) (*osc.Vm, error) {
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
	bootstrapData, err := machineScope.GetBootstrapData()
	if err != nil {
		return nil, fmt.Errorf("%w failed to decode bootstrap data", err)
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
	err, httpRes := tag.AddTag(vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
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
func (s *Service) CreateVmUserData(userData string, spec *infrastructurev1beta1.OscBastion, subnetId string, securityGroupIds []string, privateIps []string, vmName string, imageId string) (*osc.Vm, error) {
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
	err, httpRes = tag.AddTag(vmTagRequest, resourceIds, oscApiClient, oscAuthClient)
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
func (s *Service) DeleteVm(vmId string) error {
	deleteVmsRequest := osc.DeleteVmsRequest{VmIds: []string{vmId}}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteVmsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.VmApi.DeleteVms(oscAuthClient).DeleteVmsRequest(deleteVmsRequest).Execute()
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
func (s *Service) GetVm(vmId string) (*osc.Vm, error) {
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
func (s *Service) GetVmListFromTag(tagKey string, tagValue string) ([]osc.Vm, error) {
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

func (s *Service) GetCapacity(tagKey string, tagValue string, vmType string) (corev1.ResourceList, error) {
	capacity := make(corev1.ResourceList)
	vmList, err := s.GetVmListFromTag(tagKey, tagValue)
	if err != nil {
		return nil, err
	}
	var foundVmType bool
	for _, vm := range vmList {
		if *vm.VmType == vmType {
			foundVmType = true
			vmCore := strings.SplitN(strings.SplitN(vmType, "c", 2)[1], "r", 2)[0]
			vmMemory := strings.SplitN(strings.SplitN(vmType, "r", 2)[1], "p", 2)[0]
			core, err := strconv.Atoi(vmCore)
			if err != nil {
				return nil, err
			}
			cpu, err := GetCPUQuantityFromInt(core)
			if err != nil {
				return nil, fmt.Errorf("%w failed to parse quantity. CPU cores: %s. Vm Type: %s", err, vmCore, vmType)
			}
			capacity[corev1.ResourceCPU] = cpu
			ram, err := strconv.ParseFloat(vmMemory, 32)
			if err != nil {
				return nil, err
			}
			memory, err := GetMemoryQuantityFromFloat32(float32(ram))
			if err != nil {
				return nil, fmt.Errorf("%w failed to parse quantity. Memory: %s. Vm type: %s", err, vmMemory, vmType)
			}
			capacity[corev1.ResourceMemory] = memory
		}
	}
	if !foundVmType {
		return nil, fmt.Errorf("failed to find server type for %s", vmType)
	}

	return capacity, nil
}

// GetVmState return vm state
func (s *Service) GetVmState(vmId string) (string, error) {
	vm, err := s.GetVm(vmId)
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
func (s *Service) AddCcmTag(clusterName string, hostname string, vmId string) error {
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
	err, httpRes := tag.AddTag(nodeTagRequest, resourceIds, oscApiClient, oscAuthClient)
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

	err, httpRes = tag.AddTag(clusterTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

func GetCPUQuantityFromInt(cores int) (resource.Quantity, error) {
	return resource.ParseQuantity(strconv.Itoa(cores))
}

func GetMemoryQuantityFromFloat32(memory float32) (resource.Quantity, error) {
	return resource.ParseQuantity(fmt.Sprintf("%vG", memory))
}
