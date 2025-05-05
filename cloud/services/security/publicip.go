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
	"context"
	"errors"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

const PublicIPPoolTag = "OscClusterIPPool"

//go:generate ../../../bin/mockgen -destination mock_security/publicip_mock.go -package mock_security -source ./publicip.go
type OscPublicIpInterface interface {
	CreatePublicIp(ctx context.Context, publicIpName, clusterID string) (*osc.PublicIp, error)
	DeletePublicIp(ctx context.Context, publicIpId string) error
	GetPublicIp(ctx context.Context, publicIpId string) (*osc.PublicIp, error)
	GetPublicIpByIp(ctx context.Context, publicIp string) (*osc.PublicIp, error)
	ListPublicIpsFromPool(ctx context.Context, pool string) ([]osc.PublicIp, error)
}

// CreatePublicIp retrieve a publicip associated with you account
func (s *Service) CreatePublicIp(ctx context.Context, publicIpName string, clusterID string) (*osc.PublicIp, error) {
	publicIpRequest := osc.CreatePublicIpRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	publicIpResponse, httpRes, err := oscApiClient.PublicIpApi.CreatePublicIp(oscAuthClient).CreatePublicIpRequest(publicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreatePublicIp", publicIpRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	publicIpTag := osc.ResourceTag{
		Key:   "Name",
		Value: publicIpName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	resourceIds := []string{*publicIpResponse.PublicIp.PublicIpId}
	publicIpTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{publicIpTag, clusterTag},
	}

	err = tag.AddTag(ctx, publicIpTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		return nil, err
	}
	publicIp, ok := publicIpResponse.GetPublicIpOk()
	if !ok {
		return nil, errors.New("Can not create publicIp")
	}
	return publicIp, nil
}

// DeletePublicIp release the public ip
func (s *Service) DeletePublicIp(ctx context.Context, publicIpId string) error {
	deletePublicIpRequest := osc.DeletePublicIpRequest{
		PublicIpId: &publicIpId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.PublicIpApi.DeletePublicIp(oscAuthClient).DeletePublicIpRequest(deletePublicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeletePublicIp", deletePublicIpRequest, httpRes, err)
	return err
}

// GetPublicIp get a public ip object using a public ip id
func (s *Service) GetPublicIp(ctx context.Context, publicIpId string) (*osc.PublicIp, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &[]string{publicIpId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var readPublicIpsResponse osc.ReadPublicIpsResponse
	readPublicIpsResponse, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadPublicIps", readPublicIpRequest, httpRes, err)
	if err != nil {
		return nil, err
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

// GetPublicIpByIp get a public ip object using a public ip
func (s *Service) GetPublicIpByIp(ctx context.Context, publicIp string) (*osc.PublicIp, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIps: &[]string{publicIp},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readPublicIpsResponse, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadPublicIps", readPublicIpRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	publicIps, ok := readPublicIpsResponse.GetPublicIpsOk()
	if !ok {
		return nil, errors.New("Cannot get publicIp")
	}
	if len(*publicIps) == 0 {
		return nil, nil
	} else {
		publicIp := *publicIps
		return &publicIp[0], nil
	}
}

func (s *Service) ListPublicIpsFromPool(ctx context.Context, pool string) ([]osc.PublicIp, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			TagKeys:   &[]string{PublicIPPoolTag},
			TagValues: &[]string{pool},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	readPublicIpsResponse, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadPublicIps", readPublicIpRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	return readPublicIpsResponse.GetPublicIps(), nil
}

// ValidatePublicIpIds validate the list of id by checking each public ip resource and return only  public ip resource id that currently exist.
func (s *Service) ValidatePublicIpIds(ctx context.Context, publicIpIds []string) ([]string, error) {
	readPublicIpRequest := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIpIds,
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readPublicIpsResponse, httpRes, err := oscApiClient.PublicIpApi.ReadPublicIps(oscAuthClient).ReadPublicIpsRequest(readPublicIpRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadPublicIps", readPublicIpRequest, httpRes, err)
	if err != nil {
		return nil, err
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
