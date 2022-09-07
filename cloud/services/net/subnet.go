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
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/subnet_mock.go -package mock_net -source ./subnet.go
// OscSubnetInterface is the subnet interface.
type OscSubnetInterface interface {
	CreateSubnet(spec *infrastructurev1beta1.OscSubnet, netID string, subnetName string) (*osc.Subnet, error)
	DeleteSubnet(subnetID string) error
	GetSubnet(subnetID string) (*osc.Subnet, error)
	GetSubnetIDsFromNetIds(netID string) ([]string, error)
}

// CreateSubnet create the subnet associate to the net.
func (s *Service) CreateSubnet(spec *infrastructurev1beta1.OscSubnet, netID string, subnetName string) (*osc.Subnet, error) {
	ipSubnetRange, err := infrastructurev1beta1.ValidateCidr(spec.IPSubnetRange)
	if err != nil {
		return nil, err
	}
	subnetRequest := osc.CreateSubnetRequest{
		IpRange: ipSubnetRange,
		NetId:   netID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	subnetResponse, httpRes, err := oscAPIClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	err = tag.AddTag(oscAuthClient, "Name", subnetName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	subnet, ok := subnetResponse.GetSubnetOk()
	if !ok {
		return nil, errors.New("can not create subnet")
	}
	return subnet, nil
}

// DeleteSubnet delete the subnet.
func (s *Service) DeleteSubnet(subnetID string) error {
	deleteSubnetRequest := osc.DeleteSubnetRequest{SubnetId: subnetID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.SubnetApi.DeleteSubnet(oscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetSubnet retrieve Subnet object from subnet Id.
func (s *Service) GetSubnet(subnetID string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &[]string{subnetID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscAPIClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("can not get Subnets")
	}
	if len(*subnets) == 0 {
		return nil, nil
	}
	subnet := *subnets
	return &subnet[0], nil
}

// GetSubnetIDsFromNetIds return subnet id resource which eist from the net id.
func (s *Service) GetSubnetIDsFromNetIds(netID string) ([]string, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds: &[]string{netID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readSubnetsResponse, httpRes, err := oscAPIClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	var subnetIDs []string
	subnets, ok := readSubnetsResponse.GetSubnetsOk()
	if !ok {
		return nil, errors.New("can not get Subnets")
	}
	if len(*subnets) != 0 {
		for _, subnet := range *subnets {
			subnetID := subnet.SubnetId
			subnetIDs = append(subnetIDs, *subnetID)
		}
	}
	return subnetIDs, nil
}
