/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

//go:generate ../../../bin/mockgen -destination mock_security/route_mock.go -package mock_security -source ./route.go
type OscRouteTableInterface interface {
	CreateRouteTable(ctx context.Context, netId string, clusterID string, routeTableName string) (*osc.RouteTable, error)
	CreateRoute(ctx context.Context, destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
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
	routeTableRequest := osc.CreateRouteTableRequest{
		NetId: netId,
	}

	routeTableResponse, httpRes, err := s.tenant.Client().RouteTableApi.CreateRouteTable(s.tenant.ContextWithAuth(ctx)).CreateRouteTableRequest(routeTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateRouteTable", routeTableRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*routeTableResponse.RouteTable.RouteTableId}
	routeTableTag := osc.ResourceTag{
		Key:   "Name",
		Value: routeTableName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	routeTableTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{routeTableTag, clusterTag},
	}
	err = tag.AddTag(ctx, routeTableTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}

	routeTable, ok := routeTableResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("cannot create route table")
	}
	return routeTable, nil
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

	routeResponse, httpRes, err := s.tenant.Client().RouteApi.CreateRoute(s.tenant.ContextWithAuth(ctx)).CreateRouteRequest(routeRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateRoute", routeRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	route, ok := routeResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("can not create route")
	}
	return route, nil
}

// DeleteRouteTable delete the route table
func (s *Service) DeleteRouteTable(ctx context.Context, routeTableId string) error {
	deleteRouteTableRequest := osc.DeleteRouteTableRequest{RouteTableId: routeTableId}

	_, httpRes, err := s.tenant.Client().RouteTableApi.DeleteRouteTable(s.tenant.ContextWithAuth(ctx)).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteRouteTable", deleteRouteTableRequest, httpRes, err)
	return err
}

// DeleteRoute delete the route associated with the routetable
func (s *Service) DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error {
	deleteRouteRequest := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIpRange,
		RouteTableId:       routeTableId,
	}

	_, httpRes, err := s.tenant.Client().RouteApi.DeleteRoute(s.tenant.ContextWithAuth(ctx)).DeleteRouteRequest(deleteRouteRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteRoute", deleteRouteRequest, httpRes, err)
	return err
}

// GetRouteTable retrieve routetable object from the route table id
func (s *Service) GetRouteTable(ctx context.Context, routeTableId string) (*osc.RouteTable, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			RouteTableIds: &[]string{routeTableId},
		},
	}

	readRouteTablesResponse, httpRes, err := s.tenant.Client().RouteTableApi.ReadRouteTables(s.tenant.ContextWithAuth(ctx)).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteTableRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	var routetable []osc.RouteTable
	routetables, ok := readRouteTablesResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("cannot get routeTable")
	}
	if len(*routetables) == 0 {
		return nil, nil
	} else {
		routetable = append(routetable, *routetables...)
		return &routetable[0], nil
	}
}

// GetRouteTableFromRoute  retrieve the routetable object which the route are associated with  from the route table id, the resourceId and resourcetyp (gateway | nat-service)
func (s *Service) GetRouteTableFromRoute(ctx context.Context, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var readRouteRequest osc.ReadRouteTablesRequest
	switch {
	case resourceType == "gateway":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:   &[]string{routeTableId},
				RouteGatewayIds: &[]string{resourceId},
			},
		}
	case resourceType == "nat":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:      &[]string{routeTableId},
				RouteNatServiceIds: &[]string{resourceId},
			},
		}
	default:
		return nil, errors.New("Invalid Type")
	}

	readRouteResponse, httpRes, err := s.tenant.Client().RouteTableApi.ReadRouteTables(s.tenant.ContextWithAuth(ctx)).ReadRouteTablesRequest(readRouteRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	routetables, ok := readRouteResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("cannot get routeTable")
	}
	if len(*routetables) == 0 {
		return nil, nil
	} else {
		routetable := *routetables
		return &routetable[0], nil
	}
}

// LinkRouteTable associate the routetable with the subnet
func (s *Service) LinkRouteTable(ctx context.Context, routeTableId string, subnetId string) (string, error) {
	linkRouteTableRequest := osc.LinkRouteTableRequest{
		RouteTableId: routeTableId,
		SubnetId:     subnetId,
	}

	var linkRouteTableResponse osc.LinkRouteTableResponse
	linkRouteTableCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		linkRouteTableResponse, httpRes, err = s.tenant.Client().RouteTableApi.LinkRouteTable(s.tenant.ContextWithAuth(ctx)).LinkRouteTableRequest(linkRouteTableRequest).Execute()
		err = utils.LogAndExtractError(ctx, "LinkRouteTable", linkRouteTableRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, linkRouteTableCallBack)
	if waitErr != nil {
		return "", waitErr
	}
	routeTable, ok := linkRouteTableResponse.GetLinkRouteTableIdOk()
	if !ok {
		return "", errors.New("cannot link routetable")
	}
	return *routeTable, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable
func (s *Service) UnlinkRouteTable(ctx context.Context, linkRouteTableId string) error {
	unlinkRouteTableRequest := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableId,
	}

	_, httpRes, err := s.tenant.Client().RouteTableApi.UnlinkRouteTable(s.tenant.ContextWithAuth(ctx)).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "UnlinkRouteTable", unlinkRouteTableRequest, httpRes, err)
	return err
}

// GetRouteTablesFromNet returns all non main route tables present in a net.
func (s *Service) GetRouteTablesFromNet(ctx context.Context, netId string) ([]osc.RouteTable, error) {
	readRouteTablesRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds:             &[]string{netId},
			LinkRouteTableMain: ptr.To(false),
		},
	}

	readRouteTablesResponse, httpRes, err := s.tenant.Client().RouteTableApi.ReadRouteTables(s.tenant.ContextWithAuth(ctx)).ReadRouteTablesRequest(readRouteTablesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteTablesRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	return readRouteTablesResponse.GetRouteTables(), nil
}
