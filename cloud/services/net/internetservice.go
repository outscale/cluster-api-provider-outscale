package net

import (
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"

	"fmt"
)

// CreateInternetService launch the internet service
func (s *Service) CreateInternetService(internetServiceName string) (*osc.InternetService, error) {
	internetServiceRequest := osc.CreateInternetServiceRequest{}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	internetServiceResponse, httpRes, err := oscApiClient.InternetServiceApi.CreateInternetService(oscAuthClient).CreateInternetServiceRequest(internetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*internetServiceResponse.InternetService.InternetServiceId}
	err = tag.AddTag("Name", internetServiceName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return internetServiceResponse.InternetService, nil
}

// DeleteInternetService delete the internet service
func (s *Service) DeleteInternetService(internetServiceId string) error {
	deleteInternetServiceRequest := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceId}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.InternetServiceApi.DeleteInternetService(oscAuthClient).DeleteInternetServiceRequest(deleteInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// LinkInternetService attach the internet service to the net
func (s *Service) LinkInternetService(internetServiceId string, netId string) error {
	linkInternetServiceRequest := osc.LinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.InternetServiceApi.LinkInternetService(oscAuthClient).LinkInternetServiceRequest(linkInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

//UnlinkInternetService detach the internet service from the net
func (s *Service) UnlinkInternetService(internetServiceId string, netId string) error {
	unlinkInternetServiceRequest := osc.UnlinkInternetServiceRequest{
		InternetServiceId: internetServiceId,
		NetId:             netId,
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.InternetServiceApi.UnlinkInternetService(oscAuthClient).UnlinkInternetServiceRequest(unlinkInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
        
	return nil
}

// GetInternetService retrieve internet service object using internet service id
func (s *Service) GetInternetService(internetServiceIds []string) (*osc.InternetService, error) {
	readInternetServiceRequest := osc.ReadInternetServicesRequest{
		Filters: &osc.FiltersInternetService{
			InternetServiceIds: &internetServiceIds,
		},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readInternetServiceResponse, httpRes, err := oscApiClient.InternetServiceApi.ReadInternetServices(oscAuthClient).ReadInternetServicesRequest(readInternetServiceRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	internetServices := *readInternetServiceResponse.InternetServices
	if len(internetServices) == 0 {
		return nil, nil 
	} else {
		return &internetServices[0], nil
	}
}
