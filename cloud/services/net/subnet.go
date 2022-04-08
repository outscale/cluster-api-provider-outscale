package net

import (
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// CreateSubnet create the subnet associate to the net
func (s *Service) CreateSubnet(spec *infrastructurev1beta1.OscSubnet, netId string, subnetName string) (*osc.Subnet, error) {
	IpSubnetRange, err := ValidateCidr(spec.IpSubnetRange)
	if err != nil {
		return nil, err
	}
	subnetRequest := osc.CreateSubnetRequest{
		IpRange: IpSubnetRange,
		NetId:   netId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	subnetResponse, httpRes, err := OscApiClient.SubnetApi.CreateSubnet(OscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	err = tag.AddTag("Name", subnetName, resourceIds, OscApiClient, OscAuthClient)
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return subnetResponse.Subnet, nil
}

// DeleteSubnet delete the subnet
func (s *Service) DeleteSubnet(subnetId string) error {
	deleteSubnetRequest := osc.DeleteSubnetRequest{SubnetId: subnetId}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.SubnetApi.DeleteSubnet(OscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetSubnet retrieve Subnet object from subnet Id
func (s *Service) GetSubnet(subnetId []string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &subnetId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readSubnetsResponse, httpRes, err := OscApiClient.SubnetApi.ReadSubnets(OscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var subnet []osc.Subnet
	subnets := *readSubnetsResponse.Subnets
	if len(subnets) == 0 {
		return nil, nil
	} else {
		subnet = append(subnet, subnets...)
		return &subnet[0], nil
	}
}

// GetSubnetIdsFromNetIds return subnet id resource which eist from the net id
func (s *Service) GetSubnetIdsFromNetIds(netIds []string) ([]string, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds: &netIds,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readSubnetsResponse, httpRes, err := OscApiClient.SubnetApi.ReadSubnets(OscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var subnetIds []string
	subnets := *readSubnetsResponse.Subnets
	if len(subnets) != 0 {
		for _, subnet := range subnets {
			subnetId := *subnet.SubnetId
			subnetIds = append(subnetIds, subnetId)
		}
	}
	return subnetIds, nil
}
