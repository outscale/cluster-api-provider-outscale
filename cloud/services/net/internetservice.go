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
	DeleteInternetService(internetServiceID string) error
	LinkInternetService(internetServiceID string, netID string) error
	UnlinkInternetService(internetServiceID string, netID string) error
	GetInternetService(internetServiceID string) (*osc.InternetService, error)
}

// CreateInternetService launch the internet service.
func (s *Service) CreateInternetService(internetServiceName string) (*osc.InternetService, error) {
	internetServiceRequest := osc.CreateInternetServiceRequest{}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	internetServiceResponse, httpRes, err := oscAPIClient.InternetServiceApi.CreateInternetService(oscAuthClient).CreateInternetServiceRequest(internetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIds := []string{*internetServiceResponse.InternetService.InternetServiceId}
	err = tag.AddTag(oscAuthClient, "Name", internetServiceName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	internetService, ok := internetServiceResponse.GetInternetServiceOk()
	if !ok {
		return nil, errors.New("can not create internetService")
	}
	return internetService, nil
}

// DeleteInternetService delete the internet service.
func (s *Service) DeleteInternetService(internetServiceID string) error {
	deleteInternetServiceRequest := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.InternetServiceApi.DeleteInternetService(oscAuthClient).DeleteInternetServiceRequest(deleteInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// LinkInternetService attach the internet service to the net.
func (s *Service) LinkInternetService(internetServiceID string, netID string) error {
	linkInternetServiceRequest := osc.LinkInternetServiceRequest{
		InternetServiceId: internetServiceID,
		NetId:             netID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.InternetServiceApi.LinkInternetService(oscAuthClient).LinkInternetServiceRequest(linkInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// UnlinkInternetService detach the internet service from the net.
func (s *Service) UnlinkInternetService(internetServiceID string, netID string) error {
	unlinkInternetServiceRequest := osc.UnlinkInternetServiceRequest{
		InternetServiceId: internetServiceID,
		NetId:             netID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.InternetServiceApi.UnlinkInternetService(oscAuthClient).UnlinkInternetServiceRequest(unlinkInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}

	return nil
}

// GetInternetService retrieve internet service object using internet service id.
func (s *Service) GetInternetService(internetServiceID string) (*osc.InternetService, error) {
	readInternetServiceRequest := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			InternetServiceIds: &[]string{internetServiceID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readInternetServiceResponse, httpRes, err := oscAPIClient.InternetServiceApi.ReadInternetServices(oscAuthClient).ReadInternetServicesRequest(readInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	internetServices, ok := readInternetServiceResponse.GetInternetServicesOk()
	if !ok {
		return nil, errors.New("can not read internetService")
	}
	if len(*internetServices) == 0 {
		return nil, nil
	}
	internetService := *internetServices
	return &internetService[0], nil
}
