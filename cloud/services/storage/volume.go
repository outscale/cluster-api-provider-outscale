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

package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_storage/volume_mock.go -package mock_storage -source ./volume.go
type OscVolumeInterface interface {
	CreateVolume(spec *infrastructurev1beta1.OscVolume, volumeName string) (*osc.Volume, error)
	DeleteVolume(volumeID string) error
	GetVolume(volumeID string) (*osc.Volume, error)
	ValidateVolumeIds(volumeIDs []string) ([]string, error)
	LinkVolume(volumeID string, vmID string, deviceName string) error
	CheckVolumeState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, volumeID string) error
	UnlinkVolume(volumeID string) error
}

// CreateVolume create machine volume.
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
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	volumeResponse, httpRes, err := oscAPIClient.VolumeApi.CreateVolume(oscAuthClient).CreateVolumeRequest(volumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	volume, ok := volumeResponse.GetVolumeOk()
	if !ok {
		return nil, errors.New("can not create volume")
	}
	resourceIds := []string{*volumeResponse.Volume.VolumeId}
	err = tag.AddTag(oscAuthClient, "Name", volumeName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	return volume, nil
}

// GetVolume retrieve volume from volumeID.
func (s *Service) GetVolume(volumeID string) (*osc.Volume, error) {
	readVolumesRequest := osc.ReadVolumesRequest{
		Filters: &osc.FiltersVolume{
			VolumeIds: &[]string{volumeID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readVolumesResponse, httpRes, err := oscAPIClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumesRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	volumes, ok := readVolumesResponse.GetVolumesOk()
	if !ok {
		return nil, errors.New("can not get volume")
	}
	if len(*volumes) == 0 {
		return nil, nil
	}
	volume := *volumes
	return &volume[0], nil
}

// LinkVolume link machine volume.
func (s *Service) LinkVolume(volumeID string, vmID string, deviceName string) error {
	linkVolumeRequest := osc.LinkVolumeRequest{
		DeviceName: deviceName,
		VolumeId:   volumeID,
		VmId:       vmID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.VolumeApi.LinkVolume(oscAuthClient).LinkVolumeRequest(linkVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// UnlinkVolume unlink machine volume.
func (s *Service) UnlinkVolume(volumeID string) error {
	unlinkVolumeRequest := osc.UnlinkVolumeRequest{
		VolumeId: volumeID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.VolumeApi.UnlinkVolume(oscAuthClient).UnlinkVolumeRequest(unlinkVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// DeleteVolume delete machine volume.
func (s *Service) DeleteVolume(volumeID string) error {
	deleteVolumeRequest := osc.DeleteVolumeRequest{VolumeId: volumeID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.VolumeApi.DeleteVolume(oscAuthClient).DeleteVolumeRequest(deleteVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// ValidateVolumeIds validate the list of id by checking each volume resource and return volume resource that currently exist.
func (s *Service) ValidateVolumeIds(volumeIDs []string) ([]string, error) {
	readVolumeRequest := osc.ReadVolumesRequest{
		Filters: &osc.FiltersVolume{
			VolumeIds: &volumeIDs,
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readVolume, httpRes, err := oscAPIClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var validVolumeIds []string
	volumes, ok := readVolume.GetVolumesOk()
	if !ok {
		return nil, errors.New("can not get volume")
	}
	if len(*volumes) != 0 {
		for _, volume := range *volumes {
			volumeID := volume.GetVolumeId()
			validVolumeIds = append(validVolumeIds, volumeID)
		}
	}
	return validVolumeIds, nil
}

// CheckVolumeState check volume in state.
func (s *Service) CheckVolumeState(clockInsideLoop time.Duration, clockLoop time.Duration, state string, volumeID string) error {
	clocktime := clock.New()
	currentTimeout := clocktime.Now().Add(time.Second * clockLoop)
	var getVolumeState = false
	for !getVolumeState {
		volume, err := s.GetVolume(volumeID)
		if err != nil {
			return err
		}
		volumeState, ok := volume.GetStateOk()
		if !ok {
			return errors.New("can not get volume state")
		}
		if *volumeState == state {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)
		if clocktime.Now().After(currentTimeout) {
			return fmt.Errorf("volume still not in %s state", state)
		}
	}
	return nil
}
