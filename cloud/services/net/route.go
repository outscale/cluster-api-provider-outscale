package net  

import(
    infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
    osc "github.com/outscale/osc-sdk-go/v2"
    tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
    "fmt"
    "github.com/pkg/errors"
)

func (s *Service) CreateRouteTable(spec *infrastructurev1beta1.OscRouteTable, netId string, tagValue string) (*osc.RouteTable, error) {
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
    err = tag.AddTag("Name", tagValue, resourceIds, OscApiClient, OscAuthClient)
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    return routeTableResponse.RouteTable, nil
}

func (s *Service) CreateRoute(destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error) {
    var routeRequest osc.CreateRouteRequest
    switch {
    case resourceType == "gateway": 
        routeRequest = osc.CreateRouteRequest{
            DestinationIpRange: destinationIpRange,
            RouteTableId: routeTableId,
            GatewayId: &resourceId,
        }
    case resourceType == "nat":
        routeRequest = osc.CreateRouteRequest{
            DestinationIpRange: destinationIpRange,
            RouteTableId: routeTableId,
            NatServiceId: &resourceId,
        }
    default:
        errors.New("Invalid Type") 
    } 
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    routeResponse, httpRes, err :=  OscApiClient.RouteApi.CreateRoute(OscAuthClient).CreateRouteRequest(routeRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil,err
    }
    return routeResponse.RouteTable, nil
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

func (s *Service) DeleteRoute(destinationIpRange string, routeTableId string) (error) {
    deleteRouteRequest := osc.DeleteRouteRequest{
        DestinationIpRange: destinationIpRange,
        RouteTableId: routeTableId,
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

func (s *Service) GetRouteTableFromRoute(routeTableId []string, resourceId []string, resourceType string) (*osc.RouteTable, error) {
    var readRouteRequest osc.ReadRouteTablesRequest
    switch {
    case resourceType == "gateway":
        readRouteRequest = osc.ReadRouteTablesRequest{
            Filters: &osc.FiltersRouteTable{
                RouteTableIds: &routeTableId,
                RouteGatewayIds: &resourceId,
            },
        }
    case resourceType == "nat": 
        readRouteRequest = osc.ReadRouteTablesRequest{
            Filters: &osc.FiltersRouteTable{
                RouteTableIds: &routeTableId,
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
        return nil,err
    }
    var routetable []osc.RouteTable
    routetables := *readRouteResponse.RouteTables
    if len(routetables) == 0 {
        return nil, nil
    } else {
        routetable = append(routetable, routetables...)
        return &routetable[0],nil
    }
}

func (s *Service) LinkRouteTable(routeTableId string, subnetId string) (string, error) {
    linkRouteTableRequest := osc.LinkRouteTableRequest{
        RouteTableId: routeTableId,
        SubnetId: subnetId,
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

func (s *Service) UnlinkRouteTable(linkRouteTableId string) (error) {
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
