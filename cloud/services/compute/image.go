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

	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_compute/image_mock.go -package mock_compute -source ./image.go
type OscImageInterface interface {
	GetImage(ctx context.Context, imageId string) (*osc.Image, error)
	GetImageByName(ctx context.Context, imageName, accountId string) (*osc.Image, error)
}

// GetImage retrieve image from imageId
func (s *Service) GetImage(ctx context.Context, imageId string) (*osc.Image, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageIds: &[]string{imageId}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	readImagesResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadImages", readImageRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	if len(readImagesResponse.GetImages()) == 0 {
		return nil, nil
	}
	image := readImagesResponse.GetImages()[0]

	return &image, nil
}

// GetImageByName retrieve image from imageName/owner
func (s *Service) GetImageByName(ctx context.Context, imageName, accountId string) (*osc.Image, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{
			ImageNames: &[]string{imageName},
		},
	}
	if accountId != "" {
		readImageRequest.Filters.AccountIds = &[]string{accountId}
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readImagesResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadImages", readImageRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	if len(readImagesResponse.GetImages()) == 0 {
		return nil, nil
	}
	image := readImagesResponse.GetImages()[0]

	return &image, nil
}
