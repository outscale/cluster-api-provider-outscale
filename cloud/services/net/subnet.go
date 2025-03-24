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

package net

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/subnet_mock.go -package mock_net -source ./subnet.go
type OscSubnetInterface interface {
	CreateSubnet(ctx context.Context, spec *infrastructurev1beta1.OscSubnet, netId string, clusterName string, subnetName string) (*osc.Subnet, error)
	DeleteSubnet(ctx context.Context, subnetId string) error
	GetSubnet(ctx context.Context, subnetId string) (*osc.Subnet, error)
	GetSubnetIdsFromNetIds(ctx context.Context, netId string) ([]string, error)
}

// CreateSubnet create the subnet associate to the net
func (s *Service) CreateSubnet(ctx context.Context, spec *infrastructurev1beta1.OscSubnet, netId string, clusterName string, subnetName string) (*osc.Subnet, error) {
	err := infrastructurev1beta1.ValidateCidr(spec.IpSubnetRange)
	if err != nil {
		return nil, err
	}
	err = infrastructurev1beta1.ValidateSubregionName(spec.SubregionName)
	if err != nil {
		return nil, err
	}
	subnetRequest := osc.CreateSubnetRequest{
		IpRange:       spec.IpSubnetRange,
		NetId:         netId,
		SubregionName: &spec.SubregionName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	subnetResponse, httpRes, err := oscApiClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	utils.LogAPICall(ctx, "CreateSubnet", subnetRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}

	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	subnetTag := osc.ResourceTag{
		Key:   "Name",
		Value: subnetName,
	}
	subnetTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{subnetTag},
	}
	err, httpRes = tag.AddTag(ctx, subnetTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, utils.ExtractOAPIError(err, httpRes)
		} else {
			return nil, err
		}
	}
	subnetClusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	subnetClusterTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{subnetClusterTag},
	}
	err, httpRes = tag.AddTag(ctx, subnetClusterTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, utils.ExtractOAPIError(err, httpRes)
		} else {
			return nil, err
		}
	}

	subnet, ok := subnetResponse.GetSubnetOk()
	if !ok {
		return nil, errors.New("Can not create subnet")
	}
	return subnet, nil
}

// DeleteSubnet delete the subnet
func (s *Service) DeleteSubnet(ctx context.Context, subnetId string) error {
	deleteSubnetRequest := osc.DeleteSubnetRequest{SubnetId: subnetId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	_, httpRes, err := oscApiClient.SubnetApi.DeleteSubnet(oscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
	utils.LogAPICall(ctx, "DeleteSubnet", deleteSubnetRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}

// GetSubnet retrieve Subnet object from subnet Id
func (s *Service) GetSubnet(ctx context.Context, subnetId string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &[]string{subnetId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	utils.LogAPICall(ctx, "ReadSubnets", readSubnetsRequest, httpRes, err)
	if err != nil {
		return nil, fmt.Errorf("error %w httpres %s", err, httpRes.Status)
	}
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("cannot get Subnets")
	}
	if len(*subnets) == 0 {
		return nil, nil
	} else {
		subnet := *subnets
		return &subnet[0], nil
	}
}

// GetSubnetIdsFromNetIds return subnet id resource which eist from the net id
func (s *Service) GetSubnetIdsFromNetIds(ctx context.Context, netId string) ([]string, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	utils.LogAPICall(ctx, "ReadSubnets", readSubnetsRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("cannot get Subnets")
	}
	subnetIds := make([]string, 0, len(*subnets))
	for _, subnet := range *subnets {
		subnetId := subnet.SubnetId
		subnetIds = append(subnetIds, *subnetId)
	}
	return subnetIds, nil
}
