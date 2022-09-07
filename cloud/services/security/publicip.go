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

package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/publicip_mock.go -package mock_security -source ./publicip.go
type OscPublicIPInterface interface {
	CreatePublicIP(publicIPName string) (*osc.PublicIp, error)
	DeletePublicIP(publicIPID string) error
	GetPublicIP(publicIPID string) (*osc.PublicIp, error)
	LinkPublicIP(publicIPID string, vmID string) (string, error)
	UnlinkPublicIP(linkPublicIPId string) error
	CheckPublicIPUnlink(clockInsideLoop time.Duration, clockLoop time.Duration, publicIPID string) error
	ValidatePublicIPIds(publicIPIds []string) ([]string, error)
}

// CreatePublicIP retrieve a publicip associated with you account.
func (s *Service) CreatePublicIP(publicIPName string) (*osc.PublicIp, error) {
	publicIPRequest := osc.CreatePublicIpRequest{}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	publicIPResponse, httpRes, err := oscAPIClient.PublicIpApi.CreatePublicIp(oscAuthClient).CreatePublicIpRequest(publicIPRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIds := []string{*publicIPResponse.PublicIp.PublicIpId}
	err = tag.AddTag(oscAuthClient, "Name", publicIPName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	publicIP, ok := publicIPResponse.GetPublicIpOk()
	if !ok {
		return nil, errors.New("can not create publicIP")
	}
	return publicIP, nil
}

// DeletePublicIP release the public ip.
func (s *Service) DeletePublicIP(publicIPID string) error {
	deletePublicIPRequest := osc.DeletePublicIpRequest{
		PublicIpId: &publicIPID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.PublicIpApi.DeletePublicIp(oscAuthClient).DeletePublicIpRequest(deletePublicIPRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetPublicIP get a public ip object using a public ip id.
func (s *Service) GetPublicIP(publicIPID string) (*osc.PublicIp, error) {
	readPublicIPRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &[]string{publicIPID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readPublicIP, httpRes, err := oscAPIClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIPRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	publicIPs, ok := readPublicIP.GetPublicIpsOk()
	if !ok {
		return nil, errors.New("can not get publicIP")
	}
	if len(*publicIPs) == 0 {
		return nil, nil
	}
	publicIP := *publicIPs
	return &publicIP[0], nil
}

// ValidatePublicIPIds validate the list of id by checking each public ip resource and return only  public ip resource id that currently exist.
func (s *Service) ValidatePublicIPIds(publicIPIds []string) ([]string, error) {
	readPublicIPRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIPIds,
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readPublicIP, httpRes, err := oscAPIClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIPRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	var validPublicIPIDs []string
	publicIPs, ok := readPublicIP.GetPublicIpsOk()
	if !ok {
		return nil, errors.New("can not get publicIP")
	}
	if len(*publicIPs) != 0 {
		for _, publicIP := range *publicIPs {
			publicIPID := publicIP.GetPublicIpId()
			validPublicIPIDs = append(validPublicIPIDs, publicIPID)
		}
	}
	return validPublicIPIDs, nil
}

// LinkPublicIP link publicIP.
func (s *Service) LinkPublicIP(publicIPID string, vmID string) (string, error) {
	linkPublicIPRequest := osc.LinkPublicIpRequest{
		PublicIpId: &publicIPID,
		VmId:       &vmID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	linkPublicIP, httpRes, err := oscAPIClient.PublicIpApi.LinkPublicIp(oscAuthClient).LinkPublicIpRequest(linkPublicIPRequest).Execute()

	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return "", err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	linkPublicIPID, ok := linkPublicIP.GetLinkPublicIpIdOk()
	if !ok {
		return "", errors.New("can not get publicip")
	}
	return *linkPublicIPID, nil
}

// UnlinkPublicIP unlink publicIP.
func (s *Service) UnlinkPublicIP(linkPublicIPID string) error {
	unlinkPublicIPRequest := osc.UnlinkPublicIpRequest{
		LinkPublicIpId: &linkPublicIPID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.PublicIpApi.UnlinkPublicIp(oscAuthClient).UnlinkPublicIpRequest(unlinkPublicIPRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// CheckPublicIPUnlink check publicIP is unlinked.
func (s *Service) CheckPublicIPUnlink(clockInsideLoop time.Duration, clockLoop time.Duration, publicIPID string) error {
	clocktime := clock.New()
	currentTimeout := clocktime.Now().Add(time.Second * clockLoop)
	var getPublicIPUnlink = false
	for !getPublicIPUnlink {
		publicIP, err := s.GetPublicIP(publicIPID)
		if err != nil {
			return err
		}
		_, ok := publicIP.GetLinkPublicIpIdOk()
		if !ok {
			break
		}
		time.Sleep(clockInsideLoop * time.Second)

		if clocktime.Now().After(currentTimeout) {
			return errors.New("PublicIp is still link")
		}
	}
	return nil
}
