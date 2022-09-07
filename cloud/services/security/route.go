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
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/route_mock.go -package mock_security -source ./route.go
type OscRouteTableInterface interface {
	CreateRouteTable(netID string, routeTableName string) (*osc.RouteTable, error)
	CreateRoute(destinationIPRange string, routeTableID string, resourceID string, resourceType string) (*osc.RouteTable, error)
	DeleteRouteTable(routeTableID string) error
	DeleteRoute(destinationIPRange string, routeTableID string) error
	GetRouteTable(routeTableID []string) (*osc.RouteTable, error)
	GetRouteTableFromRoute(routeTableID string, resourceID string, resourceType string) (*osc.RouteTable, error)
	LinkRouteTable(routeTableID string, subnetID string) (string, error)
	UnlinkRouteTable(linkRouteTableID string) error
	GetRouteTableIdsFromNetIds(netID string) ([]string, error)
}

// CreateRouteTable create the routetable associated with the net.
func (s *Service) CreateRouteTable(netID string, routeTableName string) (*osc.RouteTable, error) {
	routeTableRequest := osc.CreateRouteTableRequest{
		NetId: netID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	routeTableResponse, httpRes, err := oscAPIClient.RouteTableApi.CreateRouteTable(oscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIDs := []string{*routeTableResponse.RouteTable.RouteTableId}
	err = tag.AddTag(oscAuthClient, "Name", routeTableName, resourceIDs, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	routeTable, ok := routeTableResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("can not create route table")
	}
	return routeTable, nil
}

// CreateRoute create the route associated with the routetable and the net.
func (s *Service) CreateRoute(destinationIPRange string, routeTableID string, resourceID string, resourceType string) (*osc.RouteTable, error) {
	var routeRequest osc.CreateRouteRequest
	valideDestinationIPRange, err := infrastructurev1beta1.ValidateCidr(destinationIPRange)
	if err != nil {
		return nil, err
	}

	switch {
	case resourceType == "gateway":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: valideDestinationIPRange,
			RouteTableId:       routeTableID,
			GatewayId:          &resourceID,
		}
	case resourceType == "nat":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: valideDestinationIPRange,
			RouteTableId:       routeTableID,
			NatServiceId:       &resourceID,
		}
	default:
		return nil, errors.New("invalid Type")
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	routeResponse, httpRes, err := oscAPIClient.RouteApi.CreateRoute(oscAuthClient).CreateRouteRequest(routeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	route, ok := routeResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("can not create route")
	}
	return route, nil
}

// DeleteRouteTable delete the route table.
func (s *Service) DeleteRouteTable(routeTableID string) error {
	deleteRouteTableRequest := osc.DeleteRouteTableRequest{RouteTableId: routeTableID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.RouteTableApi.DeleteRouteTable(oscAuthClient).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// DeleteRoute delete the route associated with the routetable.
func (s *Service) DeleteRoute(destinationIPRange string, routeTableID string) error {
	deleteRouteRequest := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIPRange,
		RouteTableId:       routeTableID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.RouteApi.DeleteRoute(oscAuthClient).DeleteRouteRequest(deleteRouteRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetRouteTable retrieve routetable object from the route table id.
func (s *Service) GetRouteTable(routeTableID []string) (*osc.RouteTable, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			RouteTableIds: &routeTableID,
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscAPIClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	var routetable []osc.RouteTable
	routetables, ok := readRouteTablesResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("can not get routeTable")
	}
	if len(*routetables) == 0 {
		return nil, nil
	}
	routetable = append(routetable, *routetables...)
	return &routetable[0], nil
}

// GetRouteTableFromRoute  retrieve the routetable object which the route are associated with  from the route table id, the resourceID and resourcetyp (gateway | nat-service).
func (s *Service) GetRouteTableFromRoute(routeTableID string, resourceID string, resourceType string) (*osc.RouteTable, error) {
	var readRouteRequest osc.ReadRouteTablesRequest
	switch {
	case resourceType == "gateway":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:   &[]string{routeTableID},
				RouteGatewayIds: &[]string{resourceID},
			},
		}
	case resourceType == "nat":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:      &[]string{routeTableID},
				RouteNatServiceIds: &[]string{resourceID},
			},
		}
	default:
		return nil, errors.New("invalid Type")
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readRouteResponse, httpRes, err := oscAPIClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	routetables, ok := readRouteResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("can not get routeTable")
	}
	if len(*routetables) == 0 {
		return nil, nil
	}
	routetable := *routetables
	return &routetable[0], nil
}

// LinkRouteTable associate the routetable with the subnet.
func (s *Service) LinkRouteTable(routeTableID string, subnetID string) (string, error) {
	linkRouteTableRequest := osc.LinkRouteTableRequest{
		RouteTableId: routeTableID,
		SubnetId:     subnetID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	linkRouteTableResponse, httpRes, err := oscAPIClient.RouteTableApi.LinkRouteTable(oscAuthClient).LinkRouteTableRequest(linkRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return "", err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	routeTable, ok := linkRouteTableResponse.GetLinkRouteTableIdOk()
	if !ok {
		return "", errors.New("can not link routetable")
	}
	return *routeTable, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable.
func (s *Service) UnlinkRouteTable(linkRouteTableID string) error {
	unlinkRouteTableRequest := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.RouteTableApi.UnlinkRouteTable(oscAuthClient).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetRouteTableIdsFromNetIds return the routeTable id resource that exist from the net id.
func (s *Service) GetRouteTableIdsFromNetIds(netID string) ([]string, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds: &[]string{netID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscAPIClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	var routetableIDs []string
	routetables, ok := readRouteTablesResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("can not get routeTable")
	}
	if len(*routetables) != 0 {
		for _, routetable := range *routetables {
			routetableID := routetable.GetRouteTableId()
			routetableIDs = append(routetableIDs, routetableID)
		}
	}
	return routetableIDs, nil
}
