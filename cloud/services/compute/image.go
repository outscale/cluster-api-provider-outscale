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
	"fmt"

	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_compute/image_mock.go -package mock_compute -source ./image.go
type OscImageInterface interface {
	GetImage(imageId string) (*osc.Image, error)
	GetImageId(imageName string) (string, error)
	GetImageName(imageId string) (string, error)
}

// GetImage retrieve image from imageId
func (s *Service) GetImage(imageId string) (*osc.Image, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageIds: &[]string{imageId}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readImageResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	if len(readImageResponse.GetImages()) == 0 {
		return nil, nil
	}
	image := readImageResponse.GetImages()[0]

	return &image, nil
}

// GetImageId retrieve imageId from imageName
func (s *Service) GetImageId(imageName string) (string, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageNames: &[]string{imageName}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readImageResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return "", fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return "", err
		}

	}
	if len(readImageResponse.GetImages()) == 0 {
		return "", nil
	}
	imageId := readImageResponse.GetImages()[0].ImageId

	return *imageId, nil
}

// GetImageName retrieve imageId from imageName
func (s *Service) GetImageName(imageId string) (string, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageNames: &[]string{imageId}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readImageResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return "", fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return "", err
		}
	}
	if len(readImageResponse.GetImages()) == 0 {
		return "", nil
	}
	imageName := readImageResponse.GetImages()[0].ImageName

	return *imageName, nil
}
