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

//go:generate ../../../bin/mockgen -destination mock_net/natservice_mock.go -package mock_net -source ./natservice.go

type OscNatServiceInterface interface {
	CreateNatService(publicIPID string, subnetID string, natServiceName string) (*osc.NatService, error)
	DeleteNatService(natServiceID string) error
	GetNatService(natServiceID string) (*osc.NatService, error)
}

// CreateNatService create the nat in the public subnet of the net.
func (s *Service) CreateNatService(publicIPID string, subnetID string, natServiceName string) (*osc.NatService, error) {
	natServiceRequest := osc.CreateNatServiceRequest{
		PublicIpId: publicIPID,
		SubnetId:   subnetID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	natServiceResponse, httpRes, err := oscAPIClient.NatServiceApi.CreateNatService(oscAuthClient).CreateNatServiceRequest(natServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIds := []string{*natServiceResponse.NatService.NatServiceId}
	err = tag.AddTag(oscAuthClient, "Name", natServiceName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	natService, ok := natServiceResponse.GetNatServiceOk()
	if !ok {
		return nil, errors.New("can not create natSrvice")
	}
	return natService, nil
}

// DeleteNatService  delete the nat.
func (s *Service) DeleteNatService(natServiceID string) error {
	deleteNatServiceRequest := osc.DeleteNatServiceRequest{NatServiceId: natServiceID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.NatServiceApi.DeleteNatService(oscAuthClient).DeleteNatServiceRequest(deleteNatServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetNatService retrieve nat service object using nat service id.
func (s *Service) GetNatService(natServiceID string) (*osc.NatService, error) {
	readNatServiceRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &[]string{natServiceID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readNatServiceResponse, httpRes, err := oscAPIClient.NatServiceApi.ReadNatServices(oscAuthClient).ReadNatServicesRequest(readNatServiceRequest).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	natServices, ok := readNatServiceResponse.GetNatServicesOk()
	if !ok {
		return nil, errors.New("can not get natService")
	}
	if len(*natServices) == 0 {
		return nil, nil
	}
	natService := *natServices
	return &natService[0], nil
}
