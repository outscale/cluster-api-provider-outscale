/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

//go:generate ../../../bin/mockgen -destination mock_net/internetservice_mock.go -package mock_net -source ./internetservice.go
type OscInternetServiceInterface interface {
	CreateInternetService(ctx context.Context, internetServiceName, clusterID string) (*osc.InternetService, error)
	DeleteInternetService(ctx context.Context, internetServiceId string) error
	LinkInternetService(ctx context.Context, internetServiceId, netId string) error
	UnlinkInternetService(ctx context.Context, internetServiceId, netId string) error
	GetInternetService(ctx context.Context, internetServiceId string) (*osc.InternetService, error)
	GetInternetServiceForNet(ctx context.Context, netId string) (*osc.InternetService, error)
}

// CreateInternetService launch the internet service
func (s *Service) CreateInternetService(ctx context.Context, internetServiceName, clusterID string) (*osc.InternetService, error) {
	req := osc.CreateInternetServiceRequest{}

	resp, err := s.tenant.Client().CreateInternetService(ctx, req)
	resourceIds := []string{resp.InternetService.InternetServiceId}
	internetServiceTag := osc.ResourceTag{
		Key:   "Name",
		Value: internetServiceName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	internetServiceTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{internetServiceTag, clusterTag},
	}
	err = tag.AddTag(ctx, internetServiceTagRequest, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}
	return resp.InternetService, nil
}

// DeleteInternetService deletes an internet service.
func (s *Service) DeleteInternetService(ctx context.Context, internetServiceId string) error {
	req := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceId}
	_, err := s.tenant.Client().DeleteInternetService(ctx, req)
	return err
}

// LinkInternetService attaches an internet service to a net.
func (s *Service) LinkInternetService(ctx context.Context, internetServiceId string, netId string) error {
	req := osc.LinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	_, err := s.tenant.Client().LinkInternetService(ctx, req)
	return err
}

// UnlinkInternetService detaches n internet service from a net.
func (s *Service) UnlinkInternetService(ctx context.Context, internetServiceId string, netId string) error {
	req := osc.UnlinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	_, err := s.tenant.Client().UnlinkInternetService(ctx, req)
	return err
}

// GetInternetService retrieve internet service object using internet service id
func (s *Service) GetInternetService(ctx context.Context, internetServiceId string) (*osc.InternetService, error) {
	req := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			InternetServiceIds: &[]string{internetServiceId},
		},
	}
	resp, err := s.tenant.Client().ReadInternetServices(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(*resp.InternetServices) == 0 {
		return nil, nil
	} else {
		internetService := *resp.InternetServices
		return &internetService[0], nil
	}
}

// GetInternetServiceForNet retrieve internet service object using internet service id
func (s *Service) GetInternetServiceForNet(ctx context.Context, netId string) (*osc.InternetService, error) {
	req := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			LinkNetIds: &[]string{netId},
		},
	}
	resp, err := s.tenant.Client().ReadInternetServices(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(*resp.InternetServices) == 0 {
		return nil, nil
	} else {
		internetService := *resp.InternetServices
		return &internetService[0], nil
	}
}
