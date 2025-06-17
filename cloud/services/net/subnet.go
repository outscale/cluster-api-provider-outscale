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
	CreateSubnet(ctx context.Context, spec infrastructurev1beta1.OscSubnet, netId, clusterID, subnetName string) (*osc.Subnet, error)
	DeleteSubnet(ctx context.Context, subnetId string) error
	GetSubnet(ctx context.Context, subnetId string) (*osc.Subnet, error)
	GetSubnetFromNet(ctx context.Context, netId, ipRange string) (*osc.Subnet, error)
}

// CreateSubnet create the subnet associate to the net
func (s *Service) CreateSubnet(ctx context.Context, spec infrastructurev1beta1.OscSubnet, netId, clusterID, subnetName string) (*osc.Subnet, error) {
	subnetRequest := osc.CreateSubnetRequest{
		IpRange:       spec.IpSubnetRange,
		NetId:         netId,
		SubregionName: &spec.SubregionName,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	subnetResponse, httpRes, err := oscApiClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateSubnet", subnetRequest, httpRes, err)
	if err != nil {
		return nil, err
	}

	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	subnetTag := osc.ResourceTag{
		Key:   "Name",
		Value: subnetName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	subnetTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        append(utils.RoleTags(spec.Roles), subnetTag, clusterTag),
	}
	err = tag.AddTag(ctx, subnetTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		return nil, err
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
	err = utils.LogAndExtractError(ctx, "DeleteSubnet", deleteSubnetRequest, httpRes, err)
	return err
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
	err = utils.LogAndExtractError(ctx, "ReadSubnets", readSubnetsRequest, httpRes, err)
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

// GetSubnetFromNet finds the subnet having a specific range within a net.
func (s *Service) GetSubnetFromNet(ctx context.Context, netId, ipRange string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds:   &[]string{netId},
			IpRanges: &[]string{ipRange},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadSubnets", readSubnetsRequest, httpRes, err)
	if err != nil {
		return nil, err
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
