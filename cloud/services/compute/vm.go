package compute

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/benbjohnson/clock"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"net"
	"time"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_compute/vm_mock.go -package mock_compute -source ./vm.go
type OscVmInterface interface {
	CreateVm(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string) (*osc.Vm, error)
	DeleteVm(vmId string) error
	GetVm(vmId string) (*osc.Vm, error)
	GetVmState(vmId string) (string, error)
	CheckVmState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, vmId string) error
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
func (s *Service) CreateVm(machineScope *scope.MachineScope, spec *infrastructurev1beta1.OscVm, subnetId string, securityGroupIds []string, privateIps []string, vmName string) (*osc.Vm, error) {
	imageId := spec.ImageId
	keypairName := spec.KeypairName
	vmType := spec.VmType
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
		ImageId:          imageId,
		KeypairName:      &keypairName,
		VmType:           &vmType,
		SubnetId:         &subnetId,
		PrivateIps:       &privateIps,
		SecurityGroupIds: &securityGroupIds,
		UserData:         &bootstrapDataEnc,
		Placement:        &placement,
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
	err = tag.AddTag("Name", vmName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
func (s *Service) DeleteVm(vmId string) error {
	deleteVmsRequest := osc.DeleteVmsRequest{VmIds: []string{vmId}}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VmApi.DeleteVms(oscAuthClient).DeleteVmsRequest(deleteVmsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
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
	readVmsResponse, httpRes, err := oscApiClient.VmApi.ReadVms(oscAuthClient).ReadVmsRequest(readVmsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
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

// CheckVmState check the vm state
func (s *Service) CheckVmState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, vmId string) error {
	clock_time := clock.New()
	currentTimeout := clock_time.Now().Add(time.Second * clockLoop)
	var getVmState = false
	for !getVmState {
		vm, err := s.GetVm(vmId)
		if err != nil {
			return err
		}
		vmState, ok := vm.GetStateOk()
		if !ok {
			return errors.New("Can not get vm state")
		}
		if *vmState == state {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)

		if clock_time.Now().After(currentTimeout) {
			return errors.New("Vm still not running")
		}
	}
	return nil
}

// GetVmState return vm state
func (s *Service) GetVmState(vmId string) (string, error) {
	vm, err := s.GetVm(vmId)
	if err != nil {
		return "", err
	}
	vmState, ok := vm.GetStateOk()
	if !ok {
		return "", errors.New("Can not get vm state")
	}
	return *vmState, nil
}
