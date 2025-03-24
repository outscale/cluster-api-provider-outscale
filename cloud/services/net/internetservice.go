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

package net

import (
	"context"
	"errors"
	"net/http"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_net/internetservice_mock.go -package mock_net -source ./internetservice.go
type OscInternetServiceInterface interface {
	CreateInternetService(ctx context.Context, internetServiceName string) (*osc.InternetService, error)
	DeleteInternetService(ctx context.Context, internetServiceId string) error
	LinkInternetService(ctx context.Context, internetServiceId string, netId string) error
	UnlinkInternetService(ctx context.Context, internetServiceId string, netId string) error
	GetInternetService(ctx context.Context, internetServiceId string) (*osc.InternetService, error)
}

// CreateInternetService creates an internet service.
func (s *Service) CreateInternetService(ctx context.Context, internetServiceName string) (*osc.InternetService, error) {
	internetServiceRequest := osc.CreateInternetServiceRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	internetServiceResponse, httpRes, err := oscApiClient.InternetServiceApi.CreateInternetService(oscAuthClient).CreateInternetServiceRequest(internetServiceRequest).Execute()
	utils.LogAPICall(ctx, "CreateInternetService", internetServiceRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	resourceIds := []string{*internetServiceResponse.InternetService.InternetServiceId}
	internetServiceTag := osc.ResourceTag{
		Key:   "Name",
		Value: internetServiceName,
	}
	internetServiceTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{internetServiceTag},
	}
	err, httpRes = tag.AddTag(ctx, internetServiceTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	internetService, ok := internetServiceResponse.GetInternetServiceOk()
	if !ok {
		return nil, errors.New("Can not create internetService")
	}
	return internetService, nil
}

// DeleteInternetService deletes an internet service.
func (s *Service) DeleteInternetService(ctx context.Context, internetServiceId string) error {
	deleteInternetServiceRequest := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.InternetServiceApi.DeleteInternetService(oscAuthClient).DeleteInternetServiceRequest(deleteInternetServiceRequest).Execute()
	utils.LogAPICall(ctx, "DeleteInternetService", deleteInternetServiceRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// LinkInternetService attaches an internet service to a net.
func (s *Service) LinkInternetService(ctx context.Context, internetServiceId string, netId string) error {
	linkInternetServiceRequest := osc.LinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	linkInternetServiceCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.InternetServiceApi.LinkInternetService(oscAuthClient).LinkInternetServiceRequest(linkInternetServiceRequest).Execute()
		utils.LogAPICall(ctx, "LinkInternetService", linkInternetServiceRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) {
				return false, nil
			}
			return false, utils.ExtractOAPIError(err, httpRes)
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, linkInternetServiceCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// UnlinkInternetService detaches n internet service from a net.
func (s *Service) UnlinkInternetService(ctx context.Context, internetServiceId string, netId string) error {
	unlinkInternetServiceRequest := osc.UnlinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.InternetServiceApi.UnlinkInternetService(oscAuthClient).UnlinkInternetServiceRequest(unlinkInternetServiceRequest).Execute()
	utils.LogAPICall(ctx, "UnlinkInternetService", unlinkInternetServiceRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// GetInternetService retrieve internet service object using internet service id
func (s *Service) GetInternetService(ctx context.Context, internetServiceId string) (*osc.InternetService, error) {
	readInternetServiceRequest := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			InternetServiceIds: &[]string{internetServiceId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readInternetServicesResponse, httpRes, err := oscApiClient.InternetServiceApi.ReadInternetServices(oscAuthClient).ReadInternetServicesRequest(readInternetServiceRequest).Execute()
	utils.LogAPICall(ctx, "ReadInternetServices", readInternetServiceRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	internetServices, ok := readInternetServicesResponse.GetInternetServicesOk()
	if !ok {
		return nil, errors.New("cannot read internetService")
	}
	if len(*internetServices) == 0 {
		return nil, nil
	} else {
		internetService := *internetServices
		return &internetService[0], nil
	}
}
