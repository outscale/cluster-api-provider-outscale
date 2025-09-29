/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
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

	readImagesResponse, httpRes, err := s.tenant.Client().ImageApi.ReadImages(s.tenant.ContextWithAuth(ctx)).ReadImagesRequest(readImageRequest).Execute()
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

	readImagesResponse, httpRes, err := s.tenant.Client().ImageApi.ReadImages(s.tenant.ContextWithAuth(ctx)).ReadImagesRequest(readImageRequest).Execute()
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
