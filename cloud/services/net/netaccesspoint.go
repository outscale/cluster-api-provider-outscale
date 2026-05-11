/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"
	"fmt"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

//go:generate ../../../bin/mockgen -destination mock_net/netaccesspoint_mock.go -package mock_net -source ./netaccesspoint.go
type OscNetAccessPointInterface interface {
	CreateNetAccessPoint(ctx context.Context, netId, region, service string, rtblIds []string, clusterID string) (*osc.NetAccessPoint, error)
	DeleteNetAccessPoint(ctx context.Context, netAccessPointId string) error
	ListNetAccessPoints(ctx context.Context, netId string) ([]osc.NetAccessPoint, error)
	GetNetAccessPoint(ctx context.Context, netAccessPointId string) (*osc.NetAccessPoint, error)
	GetNetAccessPointFor(ctx context.Context, netId, region, service string) (*osc.NetAccessPoint, error)
}

func netAccessPointServiceName(region, service string) string {
	return fmt.Sprintf("com.outscale.%s.%s", region, service)
}

// CreateNetAccessPoint launch the net access point
func (s *Service) CreateNetAccessPoint(ctx context.Context, netId, region, service string, rtblIds []string, clusterID string) (*osc.NetAccessPoint, error) {
	req := osc.CreateNetAccessPointRequest{
		NetId:         netId,
		ServiceName:   netAccessPointServiceName(region, service),
		RouteTableIds: &rtblIds,
	}

	resp, err := s.tenant.Client().CreateNetAccessPoint(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.NetAccessPoint.NetAccessPointId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netAccessPointTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err = tag.AddTag(ctx, netAccessPointTagRequest, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}
	return resp.NetAccessPoint, nil
}

// DeleteNetAccessPoint deletes an net access point.
func (s *Service) DeleteNetAccessPoint(ctx context.Context, netAccessPointId string) error {
	req := osc.DeleteNetAccessPointRequest{NetAccessPointId: netAccessPointId}

	_, err := s.tenant.Client().DeleteNetAccessPoint(ctx, req)
	return err
}

// ListNetAccessPoint lists all net access points for a net.
func (s *Service) ListNetAccessPoints(ctx context.Context, netId string) ([]osc.NetAccessPoint, error) {
	req := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetIds: &[]string{netId},
			States: &[]osc.NetAccessPointState{osc.NetAccessPointStatePending, osc.NetAccessPointStateAvailable},
		},
	}

	resp, err := s.tenant.Client().ReadNetAccessPoints(ctx, req)
	if err != nil {
		return nil, err
	}
	return *resp.NetAccessPoints, nil
}

// GetNetAccessPoint fetches a net access point by id
func (s *Service) GetNetAccessPoint(ctx context.Context, netAccessPointId string) (*osc.NetAccessPoint, error) {
	req := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetAccessPointIds: &[]string{netAccessPointId},
		},
	}

	resp, err := s.tenant.Client().ReadNetAccessPoints(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NetAccessPoints) == 0:
		return nil, nil
	default:
		return &(*resp.NetAccessPoints)[0], nil
	}
}

// GetNetAccessPointFor fetches a net access point for a net and a service.
func (s *Service) GetNetAccessPointFor(ctx context.Context, netId, region, service string) (*osc.NetAccessPoint, error) {
	req := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetIds:       &[]string{netId},
			ServiceNames: &[]string{netAccessPointServiceName(region, service)},
		},
	}

	resp, err := s.tenant.Client().ReadNetAccessPoints(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NetAccessPoints) == 0:
		return nil, nil
	default:
		return &(*resp.NetAccessPoints)[0], nil
	}
}
