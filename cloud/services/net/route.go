package net  

import(
    infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
    osc "github.com/outscale/osc-sdk-go/v2"
    tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
    "fmt"
)

func (s *Service) CreateRouteTable(spec *infrastructurev1beta1.OscRouteTable, netId string) (*osc.RouteTable, error) {
    routeTableRequest  := osc.CreateRouteTableRequest{
        NetId: netId,
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    routeTableResponse, httpRes, err := OscApiClient.RouteTableApi.CreateRouteTable(OscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    } 
    resourceIds := []string{*routeTableResponse.RouteTable.RouteTableId}
    err = tag.AddTag("Name", "cluster-api-routetable", resourceIds, OscApiClient, OscAuthClient)
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    return routeTableResponse.RouteTable, nil
}

func (s *Service) DeleteRouteTable(routeTableId string) (error) {
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

func (s *Service) GetRouteTable(tagValue string, netId []string) (*osc.RouteTable, error) {
    readRouteTableRequest := osc.ReadRouteTablesRequest{
        Filters: &osc.FiltersRouteTable{
            NetIds: &netId,
            TagValues: &[]string{tagValue},
        },
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    readRouteTablesResponse, httpRes, err := OscApiClient.RouteTableApi.ReadRouteTables(OscAuthClient).ReadRouteTablesRequest(readRouteTableRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil,err
    }
    var routetable []osc.RouteTable
    routetables := *readRouteTablesResponse.RouteTables 
    if len(routetables) == 0 {
        return nil, nil
    } else {
        routetable = append(routetable, routetables...)
        return &routetable[0],nil
    }
}

