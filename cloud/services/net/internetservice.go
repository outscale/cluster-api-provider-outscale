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
	"errors"
	"fmt"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/internetservice_mock.go -package mock_net -source ./internetservice.go
type OscInternetServiceInterface interface {
	CreateInternetService(internetServiceName string) (*osc.InternetService, error)
	DeleteInternetService(internetServiceId string) error
	LinkInternetService(internetServiceId string, netId string) error
	UnlinkInternetService(internetServiceId string, netId string) error
	GetInternetService(internetServiceId string) (*osc.InternetService, error)
}

// CreateInternetService launch the internet service
func (s *Service) CreateInternetService(internetServiceName string) (*osc.InternetService, error) {
	internetServiceRequest := osc.CreateInternetServiceRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	internetServiceResponse, httpRes, err := oscApiClient.InternetServiceApi.CreateInternetService(oscAuthClient).CreateInternetServiceRequest(internetServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	resourceIds := []string{*internetServiceResponse.InternetService.InternetServiceId}
	err = tag.AddTag("Name", internetServiceName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	internetService, ok := internetServiceResponse.GetInternetServiceOk()
	if !ok {
		return nil, errors.New("Can not create internetService")
	}
	return internetService, nil
}

// DeleteInternetService delete the internet service
func (s *Service) DeleteInternetService(internetServiceId string) error {
	deleteInternetServiceRequest := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.InternetServiceApi.DeleteInternetService(oscAuthClient).DeleteInternetServiceRequest(deleteInternetServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}
	return nil
}

// LinkInternetService attach the internet service to the net
func (s *Service) LinkInternetService(internetServiceId string, netId string) error {
	linkInternetServiceRequest := osc.LinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.InternetServiceApi.LinkInternetService(oscAuthClient).LinkInternetServiceRequest(linkInternetServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}
	return nil
}

// UnlinkInternetService detach the internet service from the net
func (s *Service) UnlinkInternetService(internetServiceId string, netId string) error {
	unlinkInternetServiceRequest := osc.UnlinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.InternetServiceApi.UnlinkInternetService(oscAuthClient).UnlinkInternetServiceRequest(unlinkInternetServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}

	return nil
}

// GetInternetService retrieve internet service object using internet service id
func (s *Service) GetInternetService(internetServiceId string) (*osc.InternetService, error) {
	readInternetServiceRequest := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			InternetServiceIds: &[]string{internetServiceId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readInternetServiceResponse, httpRes, err := oscApiClient.InternetServiceApi.ReadInternetServices(oscAuthClient).ReadInternetServicesRequest(readInternetServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	internetServices, ok := readInternetServiceResponse.GetInternetServicesOk()
	if !ok {
		return nil, errors.New("Can not read internetService")
	}
	if len(*internetServices) == 0 {
		return nil, nil
	} else {
		internetService := *internetServices
		return &internetService[0], nil
	}
}
