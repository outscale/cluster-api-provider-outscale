package storage

import (
	"errors"
	"fmt"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"time"
)

//go:generate ../../../bin/mockgen -destination mock_storage/volume_mock.go -package mock_storage -source ./volume.go
type OscVolumeInterface interface {
	CreateVolume(spec *infrastructurev1beta1.OscVolume, volumeName string) (*osc.Volume, error)
	DeleteVolume(volumeId string) error
	GetVolume(volumeId string) (*osc.Volume, error)
	ValidateVolumeIds(volumeIds []string) ([]string, error)
	LinkVolume(volumeId string, vmId string, deviceName string) error
	CheckVolumeState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, volumeId string) error
	UnlinkVolume(volumeId string) error
}

// CreateVolume create machine volume
func (s *Service) CreateVolume(spec *infrastructurev1beta1.OscVolume, volumeName string) (*osc.Volume, error) {
	size := spec.Size
	subregionName := spec.SubregionName
	volumeType := spec.VolumeType
	volumeRequest := osc.CreateVolumeRequest{
		Size:          &size,
		SubregionName: subregionName,
		VolumeType:    &volumeType,
	}
	if volumeType == "io1" {
		iops := spec.Iops
		volumeRequest.SetIops(iops)
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	volumeResponse, httpRes, err := oscApiClient.VolumeApi.CreateVolume(oscAuthClient).CreateVolumeRequest(volumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	volume, ok := volumeResponse.GetVolumeOk()
	if !ok {
		return nil, errors.New("Can not create volume")
	}
	resourceIds := []string{*volumeResponse.Volume.VolumeId}
	err = tag.AddTag("Name", volumeName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	return volume, nil
}

// GetVolume retrieve volume from volumeId
func (s *Service) GetVolume(volumeId string) (*osc.Volume, error) {
	readVolumesRequest := osc.ReadVolumesRequest{
		Filters: &osc.FiltersVolume{
			VolumeIds: &[]string{volumeId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readVolumesResponse, httpRes, err := oscApiClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumesRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	volumes, ok := readVolumesResponse.GetVolumesOk()
	if !ok {
		return nil, errors.New("Can not get volume")
	}
	if len(*volumes) == 0 {
		return nil, nil
	} else {
		volume := *volumes
		return &volume[0], nil
	}
}

// LinkVolume link machine volume
func (s *Service) LinkVolume(volumeId string, vmId string, deviceName string) error {
	linkVolumeRequest := osc.LinkVolumeRequest{
		DeviceName: deviceName,
		VolumeId:   volumeId,
		VmId:       vmId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VolumeApi.LinkVolume(oscAuthClient).LinkVolumeRequest(linkVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// UnlinkVolume unlink machine volume
func (s *Service) UnlinkVolume(volumeId string) error {
	unlinkVolumeRequest := osc.UnlinkVolumeRequest{
		VolumeId: volumeId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VolumeApi.UnlinkVolume(oscAuthClient).UnlinkVolumeRequest(unlinkVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// DeleteVolume delete machine volume
func (s *Service) DeleteVolume(volumeId string) error {
	deleteVolumeRequest := osc.DeleteVolumeRequest{VolumeId: volumeId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VolumeApi.DeleteVolume(oscAuthClient).DeleteVolumeRequest(deleteVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// ValidatePublicIpIds validate the list of id by checking each volume resource and return volume resource that currently exist
func (s *Service) ValidateVolumeIds(volumeIds []string) ([]string, error) {
	readVolumeRequest := osc.ReadVolumesRequest{
		Filters: &osc.FiltersVolume{
			VolumeIds: &volumeIds,
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readVolume, httpRes, err := oscApiClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var validVolumeIds []string
	volumes, ok := readVolume.GetVolumesOk()
	if !ok {
		return nil, errors.New("Can not get volume")
	}
	if len(*volumes) != 0 {
		for _, volume := range *volumes {
			volumeId := volume.GetVolumeId()
			validVolumeIds = append(validVolumeIds, volumeId)
		}
	}
	return validVolumeIds, nil
}

// CheckVolumeState check volume in state
func (s *Service) CheckVolumeState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, volumeId string) error {
	clock_time := clock.New()
	currentTimeout := clock_time.Now().Add(time.Second * clockLoop)
	var getVolumeState = false
	for !getVolumeState {
		volume, err := s.GetVolume(volumeId)
		if err != nil {
			return err
		}
		volumeState, ok := volume.GetStateOk()
		if !ok {
			return errors.New("Can not get volume state")
		}
		if *volumeState == state {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)
		if clock_time.Now().After(currentTimeout) {
			return fmt.Errorf("Volume still not in %s state", state)
		}
	}
	return nil
}
