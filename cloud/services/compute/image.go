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
	_nethttp "net/http"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
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

	var readImagesResponse osc.ReadImagesResponse
	readImageCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readImagesResponse, httpRes, err = oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
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

// GetImageId retrieve imageId from imageName
func (s *Service) GetImageId(imageName string) (string, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageNames: &[]string{imageName}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readImagesResponse osc.ReadImagesResponse
	readImageCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readImagesResponse, httpRes, err = oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
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
		return "", waitErr
	}
	if len(readImagesResponse.GetImages()) == 0 {
		return "", nil
	}
	imageId := readImagesResponse.GetImages()[0].ImageId

	return *imageId, nil
}

// GetImageName retrieve imageId from imageName
func (s *Service) GetImageName(imageId string) (string, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageNames: &[]string{imageId}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readImagesResponse osc.ReadImagesResponse
	readImageCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readImagesResponse, httpRes, err = oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
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
		return "", waitErr
	}
	if len(readImagesResponse.GetImages()) == 0 {
		return "", nil
	}
	imageName := readImagesResponse.GetImages()[0].ImageName
	return *imageName, nil
}
