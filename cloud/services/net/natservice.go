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
	"fmt"

	"errors"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/natservice_mock.go -package mock_net -source ./natservice.go

type OscNatServiceInterface interface {
	CreateNatService(publicIpId string, subnetId string, natServiceName string, clusterName string) (*osc.NatService, error)
	DeleteNatService(natServiceId string) error
	GetNatService(natServiceId string) (*osc.NatService, error)
}

// CreateNatService create the nat in the public subnet of the net
func (s *Service) CreateNatService(publicIpId string, subnetId string, natServiceName string, clusterName string) (*osc.NatService, error) {
	natServiceRequest := osc.CreateNatServiceRequest{
		PublicIpId: publicIpId,
		SubnetId:   subnetId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	natServiceResponse, httpRes, err := oscApiClient.NatServiceApi.CreateNatService(oscAuthClient).CreateNatServiceRequest(natServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	resourceIds := []string{*natServiceResponse.NatService.NatServiceId}
	err = tag.AddTag("Name", natServiceName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	subnet, err := s.GetSubnet(subnetId)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	resourceIds = []string{*subnet.SubnetId}
	err = tag.AddTag("OscK8sClusterID/"+clusterName, "owned", resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	natService, ok := natServiceResponse.GetNatServiceOk()
	if !ok {
		return nil, errors.New("Can not create natSrvice")
	}
	return natService, nil
}

// DeleteNatService  delete the nat
func (s *Service) DeleteNatService(natServiceId string) error {
	deleteNatServiceRequest := osc.DeleteNatServiceRequest{NatServiceId: natServiceId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.NatServiceApi.DeleteNatService(oscAuthClient).DeleteNatServiceRequest(deleteNatServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}
	return nil
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatService(natServiceId string) (*osc.NatService, error) {
	readNatServiceRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &[]string{natServiceId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readNatServiceResponse, httpRes, err := oscApiClient.NatServiceApi.ReadNatServices(oscAuthClient).ReadNatServicesRequest(readNatServiceRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	natServices, ok := readNatServiceResponse.GetNatServicesOk()
	if !ok {
		return nil, errors.New("Can not get natService")
	}
	if len(*natServices) == 0 {
		return nil, nil
	} else {
		natService := *natServices
		return &natService[0], nil
	}
}
