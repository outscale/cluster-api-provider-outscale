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
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVMInterface interface {
	CreateVM(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVM, subnetID string, securityGroupIds []string, privateIps []string, vmName string) (*osc.Vm, error)
	DeleteVM(vmID string) error
	GetVM(vmID string) (*osc.Vm, error)
	GetVMState(vmID string) (string, error)
	CheckVMState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, vmID string) error
}

// ValidateIPAddrInCidr check that ipaddr is in cidr.
func ValidateIPAddrInCidr(ipAddr string, cidr string) (string, error) {
	_, ipnet, _ := net.ParseCIDR(cidr)
	ip := net.ParseIP(ipAddr)
	if ipnet.Contains(ip) {
		return ipAddr, nil
	}
	return ipAddr, errors.New("invalid ip in cidr")
}

// CreateVM create machine vm.
func (s *Service) CreateVM(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVM, subnetID string, securityGroupIds []string, privateIps []string, vmName string) (*osc.Vm, error) {
	imageID := spec.ImageID
	keypairName := spec.KeypairName
	vmType := spec.VMType
	subregionName := spec.SubregionName
	placement := osc.Placement{
		SubregionName: &subregionName,
	}
	bootstrapData, err := machineScope.GetBootstrapData()
	if err != nil {
		return nil, fmt.Errorf("%w failed to decode bootstrap data", err)
	}
	bootstrapDataEnc := b64.StdEncoding.EncodeToString([]byte(bootstrapData))
	vmOpt := osc.CreateVmsRequest{
		ImageId:          imageID,
		KeypairName:      &keypairName,
		VmType:           &vmType,
		SubnetId:         &subnetID,
		PrivateIps:       &privateIps,
		SecurityGroupIds: &securityGroupIds,
		UserData:         &bootstrapDataEnc,
		Placement:        &placement,
	}

	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	vmResponse, httpRes, err := oscAPIClient.VmApi.CreateVms(oscAuthClient).CreateVmsRequest(vmOpt).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	vms, ok := vmResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("can not get vm")
	}
	vmID := *(*vmResponse.Vms)[0].VmId
	resourceIds := []string{vmID}
	err = tag.AddTag(oscAuthClient, "Name", vmName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if len(*vms) == 0 {
		return nil, nil
	}
	vm := *vms
	return &vm[0], nil
}

// DeleteVM delete machine vm.
func (s *Service) DeleteVM(vmID string) error {
	deleteVmsRequest := osc.DeleteVmsRequest{VmIds: []string{vmID}}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.VmApi.DeleteVms(oscAuthClient).DeleteVmsRequest(deleteVmsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetVM retrieve vm from vmId.
func (s *Service) GetVM(vmID string) (*osc.Vm, error) {
	readVmsRequest := osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			VmIds: &[]string{vmID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readVmsResponse, httpRes, err := oscAPIClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	vms, ok := readVmsResponse.GetVmsOk()
	if !ok {
		return nil, errors.New("can not get vm")
	}
	if len(*vms) == 0 {
		return nil, nil
	}
	vm := *vms
	return &vm[0], nil
}

// CheckVMState check the vm state.
func (s *Service) CheckVMState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, vmID string) error {
	clocktime := clock.New()
	currentTimeout := clocktime.Now().Add(time.Second * clockLoop)
	var getVMState = false
	for !getVMState {
		vm, err := s.GetVM(vmID)
		if err != nil {
			return err
		}
		vmState, ok := vm.GetStateOk()
		if !ok {
			return errors.New("can not get vm state")
		}
		if *vmState == state {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)

		if clocktime.Now().After(currentTimeout) {
			return errors.New("vm still not running")
		}
	}
	return nil
}

// GetVMState return vm state.
func (s *Service) GetVMState(vmID string) (string, error) {
	vm, err := s.GetVM(vmID)
	if err != nil {
		return "", err
	}
	vmState, ok := vm.GetStateOk()
	if !ok {
		return "", errors.New("can not get vm state")
	}
	return *vmState, nil
}
