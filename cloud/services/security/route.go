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
	"fmt"
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_security/route_mock.go -package mock_security -source ./route.go
type OscRouteTableInterface interface {
	CreateRouteTable(ctx context.Context, netId string, clusterName string, routeTableName string) (*osc.RouteTable, error)
	CreateRoute(ctx context.Context, destinationIpRange string, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
	DeleteRouteTable(ctx context.Context, routeTableId string) error
	DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error
	GetRouteTable(ctx context.Context, routeTableId []string) (*osc.RouteTable, error)
	GetRouteTableFromRoute(ctx context.Context, routeTableId string, resourceId string, resourceType string) (*osc.RouteTable, error)
	LinkRouteTable(ctx context.Context, routeTableId string, subnetId string) (string, error)
	UnlinkRouteTable(ctx context.Context, linkRouteTableId string) error
	GetRouteTableIdsFromNetIds(ctx context.Context, netId string) ([]string, error)
}

// CreateRouteTable create the routetable associated with the net
func (s *Service) CreateRouteTable(ctx context.Context, netId string, clusterName string, routeTableName string) (*osc.RouteTable, error) {
	routeTableRequest := osc.CreateRouteTableRequest{
		NetId: netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var routeTableResponse osc.CreateRouteTableResponse
	createRouteTableCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		routeTableResponse, httpRes, err = oscApiClient.RouteTableApi.CreateRouteTable(oscAuthClient).CreateRouteTableRequest(routeTableRequest).Execute()
		utils.LogAPICall(ctx, "CreateRouteTable", routeTableRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", routeTableRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createRouteTableCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	resourceIds := []string{*routeTableResponse.RouteTable.RouteTableId}
	routeTableTag := osc.ResourceTag{
		Key:   "Name",
		Value: routeTableName,
	}
	routeTableTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{routeTableTag},
	}
	err, httpRes := tag.AddTag(ctx, routeTableTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	clusterRouteTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err, httpRes = tag.AddTag(ctx, clusterRouteTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
	utils.LogAPICall(ctx, "CreateRoute", routeRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
	deleteRouteCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.RouteTableApi.DeleteRouteTable(oscAuthClient).DeleteRouteTableRequest(deleteRouteTableRequest).Execute()
		utils.LogAPICall(ctx, "DeleteRouteTable", deleteRouteTableRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteRouteTableRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteRouteCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// DeleteRoute delete the route associated with the routetable
func (s *Service) DeleteRoute(ctx context.Context, destinationIpRange string, routeTableId string) error {
	deleteRouteRequest := osc.DeleteRouteRequest{
		DestinationIpRange: destinationIpRange,
		RouteTableId:       routeTableId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteRouteCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.RouteApi.DeleteRoute(oscAuthClient).DeleteRouteRequest(deleteRouteRequest).Execute()
		utils.LogAPICall(ctx, "DeleteRoute", deleteRouteRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteRouteRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteRouteCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
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
	utils.LogAPICall(ctx, "ReadRouteTables", readRouteTableRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
	utils.LogAPICall(ctx, "ReadRouteTables", readRouteRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
		utils.LogAPICall(ctx, "LinkRouteTable", linkRouteTableRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", linkRouteTableRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
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

	unlinkRouteTableCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.RouteTableApi.UnlinkRouteTable(oscAuthClient).UnlinkRouteTableRequest(unlinkRouteTableRequest).Execute()
		utils.LogAPICall(ctx, "UnlinkRouteTable", unlinkRouteTableRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", unlinkRouteTableRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, err
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, unlinkRouteTableCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// GetRouteTableIdsFromNetIds return the routeTable id resource that exist from the net id
func (s *Service) GetRouteTableIdsFromNetIds(ctx context.Context, netId string) ([]string, error) {
	readRouteTablesRequest := osc.ReadRouteTablesRequest{
		Filters: &osc.FiltersRouteTable{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readRouteTablesResponse, httpRes, err := oscApiClient.RouteTableApi.ReadRouteTables(oscAuthClient).ReadRouteTablesRequest(readRouteTablesRequest).Execute()
	utils.LogAPICall(ctx, "ReadRouteTables", readRouteTablesRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
