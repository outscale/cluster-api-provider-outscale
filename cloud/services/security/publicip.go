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
	"net/http"
	"time"

	"github.com/benbjohnson/clock"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
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

// CreatePublicIp retrieve a publicip associated with you account
func (s *Service) CreatePublicIp(publicIpName string) (*osc.PublicIp, error) {
	publicIpRequest := osc.CreatePublicIpRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var publicIpResponse osc.CreatePublicIpResponse
	createPublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		publicIpResponse, httpRes, err = oscApiClient.PublicIpApi.CreatePublicIp(oscAuthClient).CreatePublicIpRequest(publicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", publicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createPublicIpCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	publicIpTag := osc.ResourceTag{
		Key:   "Name",
		Value: publicIpName,
	}
	resourceIds := []string{*publicIpResponse.PublicIp.PublicIpId}
	publicIpTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{publicIpTag},
	}

	err, httpRes := tag.AddTag(publicIpTagRequest, resourceIds, oscApiClient, oscAuthClient)
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
	deletePublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.PublicIpApi.DeletePublicIp(oscAuthClient).DeletePublicIpRequest(deletePublicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deletePublicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deletePublicIpCallBack)
	if waitErr != nil {
		return waitErr
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

	var readPublicIpsResponse osc.ReadPublicIpsResponse
	readPublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readPublicIpsResponse, httpRes, err = oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readPublicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readPublicIpCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	publicIps, ok := readPublicIpsResponse.GetPublicIpsOk()
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
	var readPublicIpsResponse osc.ReadPublicIpsResponse
	readPublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readPublicIpsResponse, httpRes, err = oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readPublicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readPublicIpCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	var validPublicIpIds []string
	publicIps, ok := readPublicIpsResponse.GetPublicIpsOk()
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
	var linkPublicIpResponse osc.LinkPublicIpResponse
	linkPublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		linkPublicIpResponse, httpRes, err = oscApiClient.PublicIpApi.LinkPublicIp(oscAuthClient).LinkPublicIpRequest(linkPublicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", linkPublicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, linkPublicIpCallBack)
	if waitErr != nil {
		return "", waitErr
	}
	linkPublicIpId, ok := linkPublicIpResponse.GetLinkPublicIpIdOk()
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
	unlinkPublicIpCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = oscApiClient.PublicIpApi.UnlinkPublicIp(oscAuthClient).UnlinkPublicIpRequest(unlinkPublicIpRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", unlinkPublicIpRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, unlinkPublicIpCallBack)
	if waitErr != nil {
		return waitErr
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
			break
		}
		time.Sleep(clockInsideLoop * time.Second)

		if clock_time.Now().After(currentTimeout) {
			return errors.New("PublicIp is still link")
		}
	}
	return nil
}
