/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"k8s.io/utils/ptr"
)

//go:generate ../../../bin/mockgen -destination mock_net/route_mock.go -package mock_net -source ./route.go
type OscRouteTableInterface interface {
	CreateRouteTable(ctx context.Context, netId string, clusterID string, routeTableName string) (*osc.RouteTable, error)
	CreateRoute(ctx context.Context, destinationIpRange string, routeTableId string, resourceId string, targetType infrastructurev1beta2.OscTargetType) (*osc.RouteTable, error)
	DeleteRouteTable(ctx context.Context, routeTableId string) error
	DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error
	GetRouteTable(ctx context.Context, routeTableId string) (*osc.RouteTable, error)
	GetRouteTableFromRoute(ctx context.Context, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
	LinkRouteTable(ctx context.Context, routeTableId string, subnetId string) (string, error)
	UnlinkRouteTable(ctx context.Context, linkRouteTableId string) error
	GetRouteTablesFromNet(ctx context.Context, netId string) ([]osc.RouteTable, error)
}

// CreateRouteTable create the routetable associated with the net
func (s *Service) CreateRouteTable(ctx context.Context, netId string, clusterID string, routeTableName string) (*osc.RouteTable, error) {
	req := osc.CreateRouteTableRequest{
		NetId: netId,
	}

	resp, err := s.tenant.Client().CreateRouteTable(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.RouteTable.RouteTableId}
	routeTableTag := osc.ResourceTag{
		Key:   "Name",
		Value: routeTableName,
	}
	clusterTag := osc.ResourceTag{
		Key:   tags.ClusterIDKey(clusterID),
		Value: "owned",
	}
	routeTableTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{routeTableTag, clusterTag},
	}
	err = tag.AddTag(ctx, routeTableTagRequest, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}

	return resp.RouteTable, nil
}

// CreateRoute create the route associated with the routetable and the net
func (s *Service) CreateRoute(ctx context.Context, destinationIpRange, routeTableId, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var routeRequest osc.CreateRouteRequest
	switch resourceType {
	case "gateway":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: destinationIpRange,
			RouteTableId:       routeTableId,
			GatewayId:          &resourceId,
		}
	case "nat":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: destinationIpRange,
			RouteTableId:       routeTableId,
			NatServiceId:       &resourceId,
		}
	case "netPeering":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: destinationIpRange,
			RouteTableId:       routeTableId,
			NetPeeringId:       &resourceId,
		}
	default:
		return nil, fmt.Errorf("invalid type %q", resourceType)
	}

	resp, err := s.tenant.Client().CreateRoute(ctx, routeRequest)
	if err != nil {
		return nil, err
	}
	return resp.RouteTable, nil
}

// DeleteRouteTable delete the route table
func (s *Service) DeleteRouteTable(ctx context.Context, routeTableId string) error {
	req := osc.DeleteRouteTableRequest{RouteTableId: routeTableId}
	_, err := s.tenant.Client().DeleteRouteTable(ctx, req)
	return err
}

// DeleteRoute delete the route associated with the routetable
func (s *Service) DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error {
	req := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIpRange,
		RouteTableId:       routeTableId,
	}

	_, err := s.tenant.Client().DeleteRoute(ctx, req)
	return err
}

// GetRouteTable retrieve routetable object from the route table id
func (s *Service) GetRouteTable(ctx context.Context, routeTableId string) (*osc.RouteTable, error) {
	req := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			RouteTableIds: &[]string{routeTableId},
		},
	}

	resp, err := s.tenant.Client().ReadRouteTables(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.RouteTables) == 0:
		return nil, nil
	default:
		return &(*resp.RouteTables)[0], nil
	}
}

// GetRouteTableFromRoute  retrieve the routetable object which the route are associated with  from the route table id, the resourceId and resourcetyp (gateway | nat-service)
func (s *Service) GetRouteTableFromRoute(ctx context.Context, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var req osc.ReadRouteTablesRequest
	switch {
	case resourceType == "gateway":
		req = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:   &[]string{routeTableId},
				RouteGatewayIds: &[]string{resourceId},
			},
		}
	case resourceType == "nat":
		req = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:      &[]string{routeTableId},
				RouteNatServiceIds: &[]string{resourceId},
			},
		}
	default:
		return nil, errors.New("Invalid Type")
	}

	resp, err := s.tenant.Client().ReadRouteTables(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.RouteTables) == 0:
		return nil, nil
	default:
		return &(*resp.RouteTables)[0], nil
	}
}

// LinkRouteTable associate the routetable with the subnet
func (s *Service) LinkRouteTable(ctx context.Context, routeTableId string, subnetId string) (string, error) {
	req := osc.LinkRouteTableRequest{
		RouteTableId: routeTableId,
		SubnetId:     subnetId,
	}

	resp, err := s.tenant.Client().LinkRouteTable(ctx, req)
	if err != nil {
		return "", err
	}
	return *resp.LinkRouteTableId, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable
func (s *Service) UnlinkRouteTable(ctx context.Context, linkRouteTableId string) error {
	req := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableId,
	}

	_, err := s.tenant.Client().UnlinkRouteTable(ctx, req)
	return err
}

// GetRouteTablesFromNet returns all non main route tables present in a net.
func (s *Service) GetRouteTablesFromNet(ctx context.Context, netId string) ([]osc.RouteTable, error) {
	req := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds:             &[]string{netId},
			LinkRouteTableMain: ptr.To(false),
		},
	}

	resp, err := s.tenant.Client().ReadRouteTables(ctx, req)
	if err != nil {
		return nil, err
	}
	return *resp.RouteTables, nil
}
