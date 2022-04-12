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
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	subnetResponse, httpRes, err := oscApiClient.SubnetApi.CreateSubnet(oscAuthClient).CreateSubnetRequest(subnetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*subnetResponse.Subnet.SubnetId}
	err = tag.AddTag("Name", subnetName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return subnetResponse.Subnet, nil
}

// DeleteSubnet delete the subnet
func (s *Service) DeleteSubnet(subnetId string) error {
	deleteSubnetRequest := osc.DeleteSubnetRequest{SubnetId: subnetId}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.SubnetApi.DeleteSubnet(oscAuthClient).DeleteSubnetRequest(deleteSubnetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetSubnet retrieve Subnet object from subnet Id
func (s *Service) GetSubnet(subnetIds []string) (*osc.Subnet, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &subnetIds,
		},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	subnets := *readSubnetsResponse.Subnets
	if len(subnets) == 0 {
		return nil, nil
	} else {
		return &subnets[0], nil
	}
}

// GetSubnetIdsFromNetIds return subnet id resource which eist from the net id
func (s *Service) GetSubnetIdsFromNetIds(netIds []string) ([]string, error) {
	readSubnetsRequest := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds: &netIds,
		},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readSubnetsResponse, httpRes, err := oscApiClient.SubnetApi.ReadSubnets(oscAuthClient).ReadSubnetsRequest(readSubnetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
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
