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

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/natservice_mock.go -package mock_net -source ./natservice.go

type OscNatServiceInterface interface {
	CreateNatService(ctx context.Context, publicIpId, subnetId, clientToken, natServiceName, clusterID string) (*osc.NatService, error)
	DeleteNatService(ctx context.Context, natServiceId string) error
	GetNatService(ctx context.Context, natServiceId string) (*osc.NatService, error)
	GetNatServiceFromClientToken(ctx context.Context, clientToken string) (*osc.NatService, error)
	ListNatServices(tx context.Context, netId string) ([]osc.NatService, error)
}

// CreateNatService create the nat in the public subnet of the net
func (s *Service) CreateNatService(ctx context.Context, publicIpId, subnetId, clientToken, natServiceName, clusterID string) (*osc.NatService, error) {
	natServiceRequest := osc.CreateNatServiceRequest{
		PublicIpId:  publicIpId,
		ClientToken: &clientToken,
		SubnetId:    subnetId,
	}

	natServiceResponse, httpRes, err := s.tenant.Client().NatServiceApi.CreateNatService(s.tenant.ContextWithAuth(ctx)).CreateNatServiceRequest(natServiceRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateNatService", natServiceRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*natServiceResponse.NatService.NatServiceId}
	natServiceTag := osc.ResourceTag{
		Key:   "Name",
		Value: natServiceName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	natServiceTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{natServiceTag, clusterTag},
	}
	err = tag.AddTag(ctx, natServiceTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	natService, ok := natServiceResponse.GetNatServiceOk()
	if !ok {
		return nil, errors.New("cannot create natService")
	}
	return natService, nil
}

// DeleteNatService  delete the nat
func (s *Service) DeleteNatService(ctx context.Context, natServiceId string) error {
	deleteNatServiceRequest := osc.DeleteNatServiceRequest{NatServiceId: natServiceId}

	_, httpRes, err := s.tenant.Client().NatServiceApi.DeleteNatService(s.tenant.ContextWithAuth(ctx)).DeleteNatServiceRequest(deleteNatServiceRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteNatService", deleteNatServiceRequest, httpRes, err)
	return err
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatService(ctx context.Context, natServiceId string) (*osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &[]string{natServiceId},
		},
	}

	readNatServicesResponse, httpRes, err := s.tenant.Client().NatServiceApi.ReadNatServices(s.tenant.ContextWithAuth(ctx)).ReadNatServicesRequest(readNatServicesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNatServices", readNatServicesRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	natServices, ok := readNatServicesResponse.GetNatServicesOk()
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

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatServiceFromClientToken(ctx context.Context, clientToken string) (*osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			ClientTokens: &[]string{clientToken},
		},
	}

	readNatServicesResponse, httpRes, err := s.tenant.Client().NatServiceApi.ReadNatServices(s.tenant.ContextWithAuth(ctx)).ReadNatServicesRequest(readNatServicesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNatServices", readNatServicesRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	natServices, ok := readNatServicesResponse.GetNatServicesOk()
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

// ListNatServices lists all nat services in a net.
func (s *Service) ListNatServices(ctx context.Context, netId string) ([]osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NetIds: &[]string{netId},
		},
	}

	readNatServicesResponse, httpRes, err := s.tenant.Client().NatServiceApi.ReadNatServices(s.tenant.ContextWithAuth(ctx)).ReadNatServicesRequest(readNatServicesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNatServices", readNatServicesRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	natServices, ok := readNatServicesResponse.GetNatServicesOk()
	if !ok {
		return nil, errors.New("Can not get natService")
	}
	return *natServices, nil
}
