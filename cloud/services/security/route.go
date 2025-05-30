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

package security

import (
	"context"
	"errors"
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
	GetRouteTable(ctx context.Context, routeTableId []string) (*osc.RouteTable, error)
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	routeTableResponse, httpRes, err := oscApiClient.RouteTableApi.CreateRouteTable(oscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
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
	err = tag.AddTag(ctx, routeTableTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		return nil, err
	}

	routeTable, ok := routeTableResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("Can not create route table")
	}
	return routeTable, nil
}

// CreateRoute create the route associated with the routetable and the net
func (s *Service) CreateRoute(ctx context.Context, destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var routeRequest osc.CreateRouteRequest
	switch {
	case resourceType == "gateway":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: destinationIpRange,
			RouteTableId:       routeTableId,
			GatewayId:          &resourceId,
		}
	case resourceType == "nat":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: destinationIpRange,
			RouteTableId:       routeTableId,
			NatServiceId:       &resourceId,
		}
	default:
		return nil, errors.New("Invalid Type")
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	routeResponse, httpRes, err := oscApiClient.RouteApi.CreateRoute(oscAuthClient).CreateRouteRequest(routeRequest).Execute()
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.RouteTableApi.DeleteRouteTable(oscAuthClient).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteRouteTable", deleteRouteTableRequest, httpRes, err)
	return err
}

// DeleteRoute delete the route associated with the routetable
func (s *Service) DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error {
	deleteRouteRequest := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIpRange,
		RouteTableId:       routeTableId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.RouteApi.DeleteRoute(oscAuthClient).DeleteRouteRequest(deleteRouteRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteRoute", deleteRouteRequest, httpRes, err)
	return err
}

// GetRouteTable retrieve routetable object from the route table id
func (s *Service) GetRouteTable(ctx context.Context, routeTableId []string) (*osc.RouteTable, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			RouteTableIds: &routeTableId,
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteTableRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	var routetable []osc.RouteTable
	routetables, ok := readRouteTablesResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("Can not get routeTable")
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	routetables, ok := readRouteResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("Can not get routeTable")
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var linkRouteTableResponse osc.LinkRouteTableResponse
	linkRouteTableCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		linkRouteTableResponse, httpRes, err = oscApiClient.RouteTableApi.LinkRouteTable(oscAuthClient).LinkRouteTableRequest(linkRouteTableRequest).Execute()
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
		return "", errors.New("Can not link routetable")
	}
	return *routeTable, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable
func (s *Service) UnlinkRouteTable(ctx context.Context, linkRouteTableId string) error {
	unlinkRouteTableRequest := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	_, httpRes, err := oscApiClient.RouteTableApi.UnlinkRouteTable(oscAuthClient).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTablesRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadRouteTables", readRouteTablesRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	return readRouteTablesResponse.GetRouteTables(), nil
}
