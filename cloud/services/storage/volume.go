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
	"context"
	"errors"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_storage/volume_mock.go -package mock_storage -source ./volume.go
type OscVolumeInterface interface {
	CreateVolume(ctx context.Context, spec *infrastructurev1beta1.OscVolume, volumeName, subregionName string) (*osc.Volume, error)
	DeleteVolume(ctx context.Context, volumeId string) error
	GetVolume(ctx context.Context, volumeId string) (*osc.Volume, error)
	LinkVolume(ctx context.Context, volumeId string, vmId string, deviceName string) error
	UnlinkVolume(ctx context.Context, volumeId string) error
}

// CreateVolume create machine volume
func (s *Service) CreateVolume(ctx context.Context, spec *infrastructurev1beta1.OscVolume, volumeName, subregionName string) (*osc.Volume, error) {
	size := spec.Size
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
	volumeResponse, httpRes, err := oscApiClient.VolumeApi.CreateVolume(oscAuthClient).CreateVolumeRequest(volumeRequest).Execute()
	utils.LogAPICall(ctx, "CreateVolume", volumeRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
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
	err, httpRes = tag.AddTag(ctx, volumeTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, utils.ExtractOAPIError(err, httpRes)
		} else {
			return nil, err
		}
	}

	return volume, nil
}

// GetVolume retrieve volume from volumeId
func (s *Service) GetVolume(ctx context.Context, volumeId string) (*osc.Volume, error) {
	readVolumesRequest := osc.ReadVolumesRequest{
		Filters: &osc.FiltersVolume{
			VolumeIds: &[]string{volumeId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var readVolumesResponse osc.ReadVolumesResponse
	readVolumesResponse, httpRes, err := oscApiClient.VolumeApi.ReadVolumes(oscAuthClient).ReadVolumesRequest(readVolumesRequest).Execute()
	utils.LogAPICall(ctx, "ReadVolumes", readVolumesRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
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
func (s *Service) LinkVolume(ctx context.Context, volumeId string, vmId string, deviceName string) error {
	linkVolumeRequest := osc.LinkVolumeRequest{
		DeviceName: deviceName,
		VolumeId:   volumeId,
		VmId:       vmId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VolumeApi.LinkVolume(oscAuthClient).LinkVolumeRequest(linkVolumeRequest).Execute()
	utils.LogAPICall(ctx, "LinkVolume", linkVolumeRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// UnlinkVolume unlink machine volume
func (s *Service) UnlinkVolume(ctx context.Context, volumeId string) error {
	unlinkVolumeRequest := osc.UnlinkVolumeRequest{
		VolumeId: volumeId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	_, httpRes, err := oscApiClient.VolumeApi.UnlinkVolume(oscAuthClient).UnlinkVolumeRequest(unlinkVolumeRequest).Execute()
	utils.LogAPICall(ctx, "UnlinkVolume", unlinkVolumeRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// DeleteVolume delete machine volume
func (s *Service) DeleteVolume(ctx context.Context, volumeId string) error {
	deleteVolumeRequest := osc.DeleteVolumeRequest{VolumeId: volumeId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.VolumeApi.DeleteVolume(oscAuthClient).DeleteVolumeRequest(deleteVolumeRequest).Execute()
	utils.LogAPICall(ctx, "DeleteVolume", deleteVolumeRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}
