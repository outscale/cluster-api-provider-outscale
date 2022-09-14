package security

import (
	"fmt"

	"errors"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/route_mock.go -package mock_security -source ./route.go
type OscRouteTableInterface interface {
	CreateRouteTable(netId string, clusterName string, routeTableName string) (*osc.RouteTable, error)
	CreateRoute(destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
	DeleteRouteTable(routeTableId string) error
	DeleteRoute(destinationIpRange string, routeTableId string) error
	GetRouteTable(routeTableId []string) (*osc.RouteTable, error)
	GetRouteTableFromRoute(routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
	LinkRouteTable(routeTableId string, subnetId string) (string, error)
	UnlinkRouteTable(linkRouteTableId string) error
	GetRouteTableIdsFromNetIds(netId string) ([]string, error)
}

// CreateRouteTable create the routetable associated with the net
func (s *Service) CreateRouteTable(netId string, clusterName string, routeTableName string) (*osc.RouteTable, error) {
	routeTableRequest := osc.CreateRouteTableRequest{
		NetId: netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	routeTableResponse, httpRes, err := oscApiClient.RouteTableApi.CreateRouteTable(oscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*routeTableResponse.RouteTable.RouteTableId}
	err = tag.AddTag("Name", routeTableName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	err = tag.AddTag("OscK8sClusterID/"+clusterName, "owned", resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	routeTable, ok := routeTableResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("Can not create route table")
	}
	return routeTable, nil
}

// CreateRoute create the route associated with the routetable and the net
func (s *Service) CreateRoute(destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
	var routeRequest osc.CreateRouteRequest
	valideDestinationIpRange, err := infrastructurev1beta1.ValidateCidr(destinationIpRange)
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
		return nil, errors.New("Invalid Type")
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	routeResponse, httpRes, err := oscApiClient.RouteApi.CreateRoute(oscAuthClient).CreateRouteRequest(routeRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	route, ok := routeResponse.GetRouteTableOk()
	if !ok {
		return nil, errors.New("can not create route")
	}
	return route, nil
}

// DeleteRouteTable delete the route table
func (s *Service) DeleteRouteTable(routeTableId string) error {
	deleteRouteTableRequest := osc.DeleteRouteTableRequest{RouteTableId: routeTableId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.RouteTableApi.DeleteRouteTable(oscAuthClient).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.RouteApi.DeleteRoute(oscAuthClient).DeleteRouteRequest(deleteRouteRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
func (s *Service) GetRouteTableFromRoute(routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
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
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
func (s *Service) LinkRouteTable(routeTableId string, subnetId string) (string, error) {
	linkRouteTableRequest := osc.LinkRouteTableRequest{
		RouteTableId: routeTableId,
		SubnetId:     subnetId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	linkRouteTableResponse, httpRes, err := oscApiClient.RouteTableApi.LinkRouteTable(oscAuthClient).LinkRouteTableRequest(linkRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return "", err
	}
	routeTable, ok := linkRouteTableResponse.GetLinkRouteTableIdOk()
	if !ok {
		return "", errors.New("Can not link routetable")
	}
	return *routeTable, nil
}

// UnlinkRouteTable diassociate the subnet from the routetable
func (s *Service) UnlinkRouteTable(linkRouteTableId string) error {
	unlinkRouteTableRequest := osc.UnlinkRouteTableRequest{
		LinkRouteTableId: linkRouteTableId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.RouteTableApi.UnlinkRouteTable(oscAuthClient).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetRouteTableIdsFromNetIds return the routeTable id resource that exist from the net id
func (s *Service) GetRouteTableIdsFromNetIds(netId string) ([]string, error) {
	readRouteTableRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var routetableIds []string
	routetables, ok := readRouteTablesResponse.GetRouteTablesOk()
	if !ok {
		return nil, errors.New("Can not get routeTable")
	}
	if len(*routetables) != 0 {
		for _, routetable := range *routetables {
			routetableId := routetable.GetRouteTableId()
			routetableIds = append(routetableIds, routetableId)
		}
	}
	return routetableIds, nil
}
