package net

import (
	"fmt"

	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

// CreateRouteTable create the routetable associated with the net
func (s *Service) CreateRouteTable(netId string, tagValue string) (*osc.RouteTable, error) {
	routeTableRequest := osc.CreateRouteTableRequest{
		NetId: netId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	routeTableResponse, httpRes, err := OscApiClient.RouteTableApi.CreateRouteTable(OscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	routeTableName, err := tag.ValidateTagNameValue(tagValue)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*routeTableResponse.RouteTable.RouteTableId}
	err = tag.AddTag("Name", routeTableName, resourceIds, OscApiClient, OscAuthClient)
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return routeTableResponse.RouteTable, nil
}

// CreateRoute create the route associated with the routetable and the net
func (s *Service) CreateRoute(destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var routeRequest osc.CreateRouteRequest
	valideDestinationIpRange, err := ValidateCidr(destinationIpRange)
	if err != nil {
		return nil, err
	}

	switch {
	case resourceType == "gateway":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: valideDestinationIpRange,
			RouteTableId:       routeTableId,
			GatewayId:          &resourceId,
		}
	case resourceType == "nat":
		routeRequest = osc.CreateRouteRequest{
			DestinationIpRange: valideDestinationIpRange,
			RouteTableId:       routeTableId,
			NatServiceId:       &resourceId,
		}
	default:
		errors.New("Invalid Type")
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	routeResponse, httpRes, err := OscApiClient.RouteApi.CreateRoute(OscAuthClient).CreateRouteRequest(routeRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return routeResponse.RouteTable, nil
}

// DeleteRouteTable delete the route table
func (s *Service) DeleteRouteTable(routeTableId string) error {
	deleteRouteTableRequest := osc.DeleteRouteTableRequest{RouteTableId: routeTableId}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.RouteTableApi.DeleteRouteTable(OscAuthClient).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// DeleteRoute delete the route associated with the routetable
func (s *Service) DeleteRoute(destinationIpRange string, routeTableId string) error {
	deleteRouteRequest := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIpRange,
		RouteTableId:       routeTableId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.RouteApi.DeleteRoute(OscAuthClient).DeleteRouteRequest(deleteRouteRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetRouteTable retrieve routetable object from the route table id
func (s *Service) GetRouteTable(routeTableId []string) (*osc.RouteTable, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			RouteTableIds: &routeTableId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readRouteTablesResponse, httpRes, err := OscApiClient.RouteTableApi.ReadRouteTables(OscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var routetable []osc.RouteTable
	routetables := *readRouteTablesResponse.RouteTables
	if len(routetables) == 0 {
		return nil, nil
	} else {
		routetable = append(routetable, routetables...)
		return &routetable[0], nil
	}
}

// GetRouteTableFromRoute  retrieve the routetable object which the route are associated with  from the route table id, the resourceId and resourcetyp (gateway | nat-service)
func (s *Service) GetRouteTableFromRoute(routeTableId []string, resourceId []string, resourceType string) (*osc.RouteTable, error) {
	var readRouteRequest osc.ReadRouteTablesRequest
	switch {
	case resourceType == "gateway":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:   &routeTableId,
				RouteGatewayIds: &resourceId,
			},
		}
	case resourceType == "nat":
		readRouteRequest = osc.ReadRouteTablesRequest{
			Filters: &osc.FiltersRouteTable{
				RouteTableIds:      &routeTableId,
				RouteNatServiceIds: &resourceId,
			},
		}
	default:
		errors.New("Invalid Type")
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readRouteResponse, httpRes, err := OscApiClient.RouteTableApi.ReadRouteTables(OscAuthClient).ReadRouteTablesRequest(readRouteRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var routetable []osc.RouteTable
	routetables := *readRouteResponse.RouteTables
	if len(routetables) == 0 {
		return nil, nil
	} else {
		routetable = append(routetable, routetables...)
		return &routetable[0], nil
	}
}
// LinkRouteTable associate the routetable with the subnet
func (s *Service) LinkRouteTable(routeTableId string, subnetId string) (string, error) {
	linkRouteTableRequest := osc.LinkRouteTableRequest{
		RouteTableId: routeTableId,
		SubnetId:     subnetId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	linkRouteTableResponse, httpRes, err := OscApiClient.RouteTableApi.LinkRouteTable(OscAuthClient).LinkRouteTableRequest(linkRouteTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return "", err
	}
	return *linkRouteTableResponse.LinkRouteTableId, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable
func (s *Service) UnlinkRouteTable(linkRouteTableId string) error {
	unlinkRouteTableRequest := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.RouteTableApi.UnlinkRouteTable(OscAuthClient).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetRouteTableIdsFromNetIds return the routeTable id resource that exist from the net id
func (s *Service) GetRouteTableIdsFromNetIds(netIds []string) ([]string, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds: &netIds,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readRouteTablesResponse, httpRes, err := OscApiClient.RouteTableApi.ReadRouteTables(OscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var routetableIds []string
	routetables := *readRouteTablesResponse.RouteTables
	if len(routetables) != 0 {
		for _, routetable := range routetables {
			routetableId := *routetable.RouteTableId
			routetableIds = append(routetableIds, routetableId)
		}
	}
	return routetableIds, nil
}
