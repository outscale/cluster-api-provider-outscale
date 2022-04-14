package net

import (
	"fmt"

	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// CreateNatService create the nat in the public subnet of the net
func (s *Service) CreateNatService(publicIpId string, subnetId string, natServiceName string) (*osc.NatService, error) {
	natServiceRequest := osc.CreateNatServiceRequest{
		PublicIpId: publicIpId,
		SubnetId:   subnetId,
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	natServiceResponse, httpRes, err := oscApiClient.NatServiceApi.CreateNatService(oscAuthClient).CreateNatServiceRequest(natServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*natServiceResponse.NatService.NatServiceId}
	err = tag.AddTag("Name", natServiceName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return natServiceResponse.NatService, nil
}

// DeleteNatService  delete the nat
func (s *Service) DeleteNatService(natServiceId string) error {
	deleteNatServiceRequest := osc.DeleteNatServiceRequest{NatServiceId: natServiceId}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.NatServiceApi.DeleteNatService(oscAuthClient).DeleteNatServiceRequest(deleteNatServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatService(natServiceId string) (*osc.NatService, error) {
	readNatServiceRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &[]string{natServiceId},
		},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readNatServiceResponse, httpRes, err := oscApiClient.NatServiceApi.ReadNatServices(oscAuthClient).ReadNatServicesRequest(readNatServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	natServices := *readNatServiceResponse.NatServices
	if len(natServices) == 0 {
		return nil, nil
	} else {
		return &natServices[0], nil
	}
}
