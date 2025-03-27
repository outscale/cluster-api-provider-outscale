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
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
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
	var subnetResponse osc.CreateSubnetResponse
	createSubnetCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		subnetResponse, httpRes, err = oscApiClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
		utils.LogAPICall(ctx, "CreateSubnet", subnetRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}
			requestStr := fmt.Sprintf("%v", subnetRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, createSubnetCallBack)
	if waitErr != nil {
		return nil, waitErr
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
		Tags:        []osc.ResourceTag{subnetTag, clusterTag},
	}
	err := tag.AddTag(ctx, subnetTagRequest, resourceIds, oscApiClient, oscAuthClient)
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

	deleteSubnetCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.SubnetApi.DeleteSubnet(oscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
		utils.LogAPICall(ctx, "DeleteSubnet", deleteSubnetRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}

			requestStr := fmt.Sprintf("%v", deleteSubnetRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, deleteSubnetCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
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
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpres %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("Can not get Subnets")
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
	var readSubnetsResponse osc.ReadSubnetsResponse
	readSubnetsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readSubnetsResponse, httpRes, err = oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
		utils.LogAPICall(ctx, "ReadSubnets", readSubnetsRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, utils.ExtractOAPIError(err, httpRes)
			}
			requestStr := fmt.Sprintf("%v", readSubnetsRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, readSubnetsCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("Can not get Subnets")
	}
	if len(*subnets) == 0 {
		return nil, nil
	} else {
		subnet := *subnets
		return &subnet[0], nil
	}
}
