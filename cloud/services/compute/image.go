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
	"fmt"
	"net/http"

	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
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

	var readImagesResponse osc.ReadImagesResponse
	readImageCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readImagesResponse, httpRes, err = oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
		utils.LogAPICall(ctx, "ReadImages", readImageRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}
			requestStr := fmt.Sprintf("%v", readImageRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, readImageCallBack)
	if waitErr != nil {
		return nil, waitErr
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
	var readImagesResponse osc.ReadImagesResponse
	readImageCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readImagesResponse, httpRes, err = oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
		utils.LogAPICall(ctx, "ReadImages", readImageRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}
			requestStr := fmt.Sprintf("%v", readImageRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, readImageCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	if len(readImagesResponse.GetImages()) == 0 {
		return nil, nil
	}
	image := readImagesResponse.GetImages()[0]

	return &image, nil
}
