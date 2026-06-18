/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type NatServiceInterface interface {
	CreateNatService(ctx context.Context, publicIpId, subnetId, clientToken, natServiceName, clusterID string) (*osc.NatService, error)
	DeleteNatService(ctx context.Context, natServiceId string) error
	GetNatService(ctx context.Context, natServiceId string) (*osc.NatService, error)
	GetNatServiceFromClientToken(ctx context.Context, clientToken string) (*osc.NatService, error)
	ListNatServices(tx context.Context, netId string) ([]osc.NatService, error)
}

// CreateNatService create the nat in the public subnet of the net
func (s *Service) CreateNatService(ctx context.Context, publicIpId, subnetId, clientToken, natServiceName, clusterID string) (*osc.NatService, error) {
	req := osc.CreateNatServiceRequest{
		PublicIpId:  publicIpId,
		ClientToken: &clientToken,
		SubnetId:    subnetId,
	}

	resp, err := s.tenant.Client().CreateNatService(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.NatService.NatServiceId}
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
	err = s.tags.AddTag(ctx, natServiceTagRequest, resourceIds)
	if err != nil {
		return nil, err
	}
	return resp.NatService, nil
}

// DeleteNatService  delete the nat
func (s *Service) DeleteNatService(ctx context.Context, natServiceId string) error {
	req := osc.DeleteNatServiceRequest{NatServiceId: natServiceId}
	_, err := s.tenant.Client().DeleteNatService(ctx, req)
	return err
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatService(ctx context.Context, natServiceId string) (*osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &[]string{natServiceId},
		},
	}
	resp, err := s.tenant.Client().ReadNatServices(ctx, readNatServicesRequest)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NatServices) == 0:
		return nil, nil
	default:
		return &(*resp.NatServices)[0], nil
	}
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatServiceFromClientToken(ctx context.Context, clientToken string) (*osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			ClientTokens: &[]string{clientToken},
		},
	}

	resp, err := s.tenant.Client().ReadNatServices(ctx, readNatServicesRequest)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NatServices) == 0:
		return nil, nil
	default:
		return &(*resp.NatServices)[0], nil
	}
}

// ListNatServices lists all nat services in a net.
func (s *Service) ListNatServices(ctx context.Context, netId string) ([]osc.NatService, error) {
	readNatServicesRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NetIds: &[]string{netId},
		},
	}

	resp, err := s.tenant.Client().ReadNatServices(ctx, readNatServicesRequest)
	if err != nil {
		return nil, err
	}
	return *resp.NatServices, nil
}
