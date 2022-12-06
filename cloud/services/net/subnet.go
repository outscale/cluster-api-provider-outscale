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
	"fmt"

	"errors"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/subnet_mock.go -package mock_net -source ./subnet.go
type OscSubnetInterface interface {
	CreateSubnet(spec *infrastructurev1beta1.OscSubnet, netId string, clusterName string, subnetName string) (*osc.Subnet, error)
	DeleteSubnet(subnetId string) error
	GetSubnet(subnetId string) (*osc.Subnet, error)
	GetSubnetIdsFromNetIds(netId string) ([]string, error)
}

// CreateSubnet create the subnet associate to the net
func (s *Service) CreateSubnet(spec *infrastructurev1beta1.OscSubnet, netId string, clusterName string, subnetName string) (*osc.Subnet, error) {
	ipSubnetRange, err := infrastructurev1beta1.ValidateCidr(spec.IpSubnetRange)
	if err != nil {
		return nil, err
	}
	subnetRequest := osc.CreateSubnetRequest{
		IpRange: ipSubnetRange,
		NetId:   netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	subnetResponse, httpRes, err := oscApiClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	err = tag.AddTag("Name", subnetName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	err = tag.AddTag("OscK8sClusterID/"+clusterName, "owned", resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
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
func (s *Service) DeleteSubnet(subnetId string) error {
	deleteSubnetRequest := osc.DeleteSubnetRequest{SubnetId: subnetId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.SubnetApi.DeleteSubnet(oscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}
	return nil
}

// GetSubnet retrieve Subnet object from subnet Id
func (s *Service) GetSubnet(subnetId string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &[]string{subnetId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
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

// GetSubnetIdsFromNetIds return subnet id resource which eist from the net id
func (s *Service) GetSubnetIdsFromNetIds(netId string) ([]string, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpres %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	var subnetIds []string
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("Can not get Subnets")
	}
	if len(*subnets) != 0 {
		for _, subnet := range *subnets {
			subnetId := subnet.SubnetId
			subnetIds = append(subnetIds, *subnetId)
		}
	}
	return subnetIds, nil
}
