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
	"net/http"
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
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
	var volumeResponse osc.CreateVolumeResponse
	createVolumeCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		volumeResponse, httpRes, err = oscApiClient.VolumeApi.CreateVolume(oscAuthClient).CreateVolumeRequest(volumeRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}

			requestStr := fmt.Sprintf("%v", volumeRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, createVolumeCallBack)
	if waitErr != nil {
		return nil, waitErr
	}

	volume, ok := volumeResponse.GetVolumeOk()
	if !ok {
		return nil, errors.New("Can not create volume")
	}
	resourceIds := []string{*volumeResponse.Volume.VolumeId}
	volumeTag := osc.ResourceTag{
		Key:   "Name",
		Value: volumeName,
	}
	volumeTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{volumeTag},
	}
	err, httpRes := tag.AddTag(volumeTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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

	var readVolumesResponse osc.ReadVolumesResponse
	readVolumesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readVolumesResponse, httpRes, err = oscApiClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumesRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readVolumesRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, readVolumesCallBack)
	if waitErr != nil {
		return nil, waitErr
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
	linkVolumeCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.VolumeApi.LinkVolume(oscAuthClient).LinkVolumeRequest(linkVolumeRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", linkVolumeRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, linkVolumeCallBack)
	if waitErr != nil {
		return waitErr
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

	unlinkVolumeCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.VolumeApi.UnlinkVolume(oscAuthClient).UnlinkVolumeRequest(unlinkVolumeRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", unlinkVolumeRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, err
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, unlinkVolumeCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// DeleteVolume delete machine volume
func (s *Service) DeleteVolume(volumeId string) error {
	deleteVolumeRequest := osc.DeleteVolumeRequest{VolumeId: volumeId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteVolumeCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.VolumeApi.DeleteVolume(oscAuthClient).DeleteVolumeRequest(deleteVolumeRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteVolumeRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, deleteVolumeCallBack)
	if waitErr != nil {
		return waitErr
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
	var readVolumesResponse osc.ReadVolumesResponse
	readVolumesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readVolumesResponse, httpRes, err = oscApiClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumeRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readVolumeRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, readVolumesCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	var validVolumeIds []string
	volumes, ok := readVolumesResponse.GetVolumesOk()
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
