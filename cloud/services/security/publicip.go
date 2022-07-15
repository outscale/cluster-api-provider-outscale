package security

import (
	"fmt"

	"errors"
	"github.com/benbjohnson/clock"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"time"
)

//go:generate ../../../bin/mockgen -destination mock_security/publicip_mock.go -package mock_security -source ./publicip.go
type OscPublicIpInterface interface {
	CreatePublicIp(publicIpName string) (*osc.PublicIp, error)
	DeletePublicIp(publicIpId string) error
	GetPublicIp(publicIpId string) (*osc.PublicIp, error)
	LinkPublicIp(publicIpId string, vmId string) (string, error)
	UnlinkPublicIp(linkPublicIpId string) error
	CheckPublicIpUnlink(clockInsideLoop time.Duration, clockLoop time.Duration, publicIpId string) error
	ValidatePublicIpIds(publicIpIds []string) ([]string, error)
}

//CreatePublicIp retrieve a publicip associated with you account
func (s *Service) CreatePublicIp(publicIpName string) (*osc.PublicIp, error) {
	publicIpRequest := osc.CreatePublicIpRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	publicIpResponse, httpRes, err := oscApiClient.PublicIpApi.CreatePublicIp(oscAuthClient).CreatePublicIpRequest(publicIpRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*publicIpResponse.PublicIp.PublicIpId}
	err = tag.AddTag("Name", publicIpName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	publicIp, ok := publicIpResponse.GetPublicIpOk()
	if !ok {
		return nil, errors.New("Can not create publicIp")
	}
	return publicIp, nil
}

// DeletePublicIp release the public ip
func (s *Service) DeletePublicIp(publicIpId string) error {
	deletePublicIpRequest := osc.DeletePublicIpRequest{
		PublicIpId: &publicIpId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.PublicIpApi.DeletePublicIp(oscAuthClient).DeletePublicIpRequest(deletePublicIpRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetPublicIp get a public ip object using a public ip id
func (s *Service) GetPublicIp(publicIpId string) (*osc.PublicIp, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &[]string{publicIpId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readPublicIp, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	publicIps, ok := readPublicIp.GetPublicIpsOk()
	if !ok {
		return nil, errors.New("Can not get publicIp")
	}
	if len(*publicIps) == 0 {
		return nil, nil
	} else {
		publicIp := *publicIps
		return &publicIp[0], nil
	}
}

// ValidatePublicIpIds validate the list of id by checking each public ip resource and return only  public ip resource id that currently exist.
func (s *Service) ValidatePublicIpIds(publicIpIds []string) ([]string, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIpIds,
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readPublicIp, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var validPublicIpIds []string
	publicIps, ok := readPublicIp.GetPublicIpsOk()
	if !ok {
		return nil, errors.New("Can not get publicIp")
	}
	if len(*publicIps) != 0 {
		for _, publicIp := range *publicIps {
			publicIpId := publicIp.GetPublicIpId()
			validPublicIpIds = append(validPublicIpIds, publicIpId)
		}
	}
	return validPublicIpIds, nil
}

// LinkPublicIp link publicIp
func (s *Service) LinkPublicIp(publicIpId string, vmId string) (string, error) {
	linkPublicIpRequest := osc.LinkPublicIpRequest{
		PublicIpId: &publicIpId,
		VmId:       &vmId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	linkPublicIp, httpRes, err := oscApiClient.PublicIpApi.LinkPublicIp(oscAuthClient).LinkPublicIpRequest(linkPublicIpRequest).Execute()

	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return "", err
	}
	linkPublicIpId, ok := linkPublicIp.GetLinkPublicIpIdOk()
	if !ok {
		return "", errors.New("Can not get publicip")
	}
	return *linkPublicIpId, nil
}

// UnlinkPublicIp unlink publicIp
func (s *Service) UnlinkPublicIp(linkPublicIpId string) error {
	unlinkPublicIpRequest := osc.UnlinkPublicIpRequest{
		LinkPublicIpId: &linkPublicIpId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.PublicIpApi.UnlinkPublicIp(oscAuthClient).UnlinkPublicIpRequest(unlinkPublicIpRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// CheckPublicIpUnlink check publicIp is unlinked
func (s *Service) CheckPublicIpUnlink(clockInsideLoop time.Duration, clockLoop time.Duration, publicIpId string) error {
	clock_time := clock.New()
	currentTimeout := clock_time.Now().Add(time.Second * clockLoop)
	var getPublicIpUnlink = false
	for !getPublicIpUnlink {
		publicIp, err := s.GetPublicIp(publicIpId)
		if err != nil {
			return err
		}
		_, ok := publicIp.GetLinkPublicIpIdOk()
		if !ok {
			return nil
		}
		time.Sleep(clockInsideLoop * time.Second)

		if clock_time.Now().After(currentTimeout) {
			return errors.New("PublicIp is still link")
		}
	}
	return nil
}
