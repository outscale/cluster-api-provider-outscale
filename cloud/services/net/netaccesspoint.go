/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"
	"errors"
	"fmt"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
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
	netAccessPointRequest := osc.CreateNetAccessPointRequest{
		NetId:         netId,
		ServiceName:   netAccessPointServiceName(region, service),
		RouteTableIds: &rtblIds,
	}

	netAccessPointResponse, httpRes, err := s.tenant.Client().NetAccessPointApi.CreateNetAccessPoint(s.tenant.ContextWithAuth(ctx)).CreateNetAccessPointRequest(netAccessPointRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateNetAccessPoint", netAccessPointRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*netAccessPointResponse.NetAccessPoint.NetAccessPointId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netAccessPointTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err = tag.AddTag(ctx, netAccessPointTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	netAccessPoint, ok := netAccessPointResponse.GetNetAccessPointOk()
	if !ok {
		return nil, errors.New("cannot create netAccessPoint")
	}
	return netAccessPoint, nil
}

// DeleteNetAccessPoint deletes an net access point.
func (s *Service) DeleteNetAccessPoint(ctx context.Context, netAccessPointId string) error {
	deleteNetAccessPointRequest := osc.DeleteNetAccessPointRequest{NetAccessPointId: netAccessPointId}

	_, httpRes, err := s.tenant.Client().NetAccessPointApi.DeleteNetAccessPoint(s.tenant.ContextWithAuth(ctx)).DeleteNetAccessPointRequest(deleteNetAccessPointRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteNetAccessPoint", deleteNetAccessPointRequest, httpRes, err)
	return err
}

// ListNetAccessPoint lists all net access points for a net.
func (s *Service) ListNetAccessPoints(ctx context.Context, netId string) ([]osc.NetAccessPoint, error) {
	readNetAccessPointRequest := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetIds: &[]string{netId},
			States: &[]string{"pending", "available"},
		},
	}

	resp, httpRes, err := s.tenant.Client().NetAccessPointApi.ReadNetAccessPoints(s.tenant.ContextWithAuth(ctx)).ReadNetAccessPointsRequest(readNetAccessPointRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetAccessPoints", readNetAccessPointRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	return resp.GetNetAccessPoints(), nil
}

// GetNetAccessPoint fetches a net access point by id
func (s *Service) GetNetAccessPoint(ctx context.Context, netAccessPointId string) (*osc.NetAccessPoint, error) {
	readNetAccessPointRequest := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetAccessPointIds: &[]string{netAccessPointId},
		},
	}

	resp, httpRes, err := s.tenant.Client().NetAccessPointApi.ReadNetAccessPoints(s.tenant.ContextWithAuth(ctx)).ReadNetAccessPointsRequest(readNetAccessPointRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetAccessPoints", readNetAccessPointRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netAccessPoints, ok := resp.GetNetAccessPointsOk()
	if !ok {
		return nil, errors.New("cannot read netAccessPoint")
	}
	if len(*netAccessPoints) == 0 {
		return nil, nil
	} else {
		netAccessPoint := *netAccessPoints
		return &netAccessPoint[0], nil
	}
}

// GetNetAccessPointFor fetches a net access point for a net and a service.
func (s *Service) GetNetAccessPointFor(ctx context.Context, netId, region, service string) (*osc.NetAccessPoint, error) {
	readNetAccessPointRequest := osc.ReadNetAccessPointsRequest{
		Filters: &osc.FiltersNetAccessPoint{
			NetIds:       &[]string{netId},
			ServiceNames: &[]string{netAccessPointServiceName(region, service)},
		},
	}

	resp, httpRes, err := s.tenant.Client().NetAccessPointApi.ReadNetAccessPoints(s.tenant.ContextWithAuth(ctx)).ReadNetAccessPointsRequest(readNetAccessPointRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetAccessPoints", readNetAccessPointRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netAccessPoints, ok := resp.GetNetAccessPointsOk()
	if !ok {
		return nil, errors.New("cannot read netAccessPoint")
	}
	if len(*netAccessPoints) == 0 {
		return nil, nil
	} else {
		netAccessPoint := *netAccessPoints
		return &netAccessPoint[0], nil
	}
}
