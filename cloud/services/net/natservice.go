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
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	natServiceResponse, httpRes, err := OscApiClient.NatServiceApi.CreateNatService(OscAuthClient).CreateNatServiceRequest(natServiceRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*natServiceResponse.NatService.NatServiceId}
	err = tag.AddTag("Name", natServiceName, resourceIds, OscApiClient, OscAuthClient)
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return natServiceResponse.NatService, nil
}

// DeleteNatService  delete the nat
func (s *Service) DeleteNatService(natServiceId string) error {
	deleteNatServiceRequest := osc.DeleteNatServiceRequest{NatServiceId: natServiceId}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.NatServiceApi.DeleteNatService(OscAuthClient).DeleteNatServiceRequest(deleteNatServiceRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetNatService retrieve nat service object using nat service id
func (s *Service) GetNatService(natServiceId []string) (*osc.NatService, error) {
	readNatServiceRequest := osc.ReadNatServicesRequest{
		Filters: &osc.FiltersNatService{
			NatServiceIds: &natServiceId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readNatServiceResponse, httpRes, err := OscApiClient.NatServiceApi.ReadNatServices(OscAuthClient).ReadNatServicesRequest(readNatServiceRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var natservice []osc.NatService
	natservices := *readNatServiceResponse.NatServices
	if len(natservices) == 0 {
		return nil, nil
	} else {
		natservice = append(natservice, natservices...)
		return &natservice[0], nil
	}
}
