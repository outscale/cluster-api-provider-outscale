package net

import (
	"fmt"

	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//CreatePublicIp retrieve a publicip associated with you account
func (s *Service) CreatePublicIp(tagValue string) (*osc.PublicIp, error) {
	publicIpRequest := osc.CreatePublicIpRequest{}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	publicIpResponse, httpRes, err := OscApiClient.PublicIpApi.CreatePublicIp(OscAuthClient).CreatePublicIpRequest(publicIpRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	publicIpName, err := tag.ValidateTagNameValue(tagValue)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*publicIpResponse.PublicIp.PublicIpId}
	err = tag.AddTag("Name", publicIpName, resourceIds, OscApiClient, OscAuthClient)
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return publicIpResponse.PublicIp, nil
}

// DeletePublicIp release the public ip
func (s *Service) DeletePublicIp(publicIpId string) error {
	deletePublicIpRequest := osc.DeletePublicIpRequest{
		PublicIpId: &publicIpId,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.PublicIpApi.DeletePublicIp(OscAuthClient).DeletePublicIpRequest(deletePublicIpRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetPublicIp get a public ip object using a public ip id
func (s *Service) GetPublicIp(publicIpId []string) (*osc.PublicIp, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIpId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readPublicIp, httpRes, err := OscApiClient.PublicIpApi.ReadPublicIps(OscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var publicip []osc.PublicIp
	publicips := *readPublicIp.PublicIps
	if len(publicips) == 0 {
		return nil, nil
	} else {
		publicip = append(publicip, publicips...)
		return &publicip[0], nil
	}
}

// ValidatePublicIpIds validate the list of id by checking each public ip resource and return only  public ip resource id that currently exist.
func (s *Service) ValidatePublicIpIds(publicIpId []string) ([]string, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIpId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readPublicIp, httpRes, err := OscApiClient.PublicIpApi.ReadPublicIps(OscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var publicipIds []string
	publicips := *readPublicIp.PublicIps
	if len(publicips) != 0 {
		for _, publicip := range publicips {
			publicipId := *publicip.PublicIpId
			publicipIds = append(publicipIds, publicipId)
		}
	}
	return publicipIds, nil
}
