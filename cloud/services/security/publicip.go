/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"context"
	"errors"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

const PublicIPPoolTag = "OscK8sIPPool"

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

	publicIpResponse, httpRes, err := s.tenant.Client().PublicIpApi.CreatePublicIp(s.tenant.ContextWithAuth(ctx)).CreatePublicIpRequest(publicIpRequest).Execute()
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

	err = tag.AddTag(ctx, publicIpTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
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

	_, httpRes, err := s.tenant.Client().PublicIpApi.DeletePublicIp(s.tenant.ContextWithAuth(ctx)).DeletePublicIpRequest(deletePublicIpRequest).Execute()
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

	var readPublicIpsResponse osc.ReadPublicIpsResponse
	readPublicIpsResponse, httpRes, err := s.tenant.Client().PublicIpApi.ReadPublicIps(s.tenant.ContextWithAuth(ctx)).ReadPublicIpsRequest(readPublicIpRequest).Execute()
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

	readPublicIpsResponse, httpRes, err := s.tenant.Client().PublicIpApi.ReadPublicIps(s.tenant.ContextWithAuth(ctx)).ReadPublicIpsRequest(readPublicIpRequest).Execute()
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

	readPublicIpsResponse, httpRes, err := s.tenant.Client().PublicIpApi.ReadPublicIps(s.tenant.ContextWithAuth(ctx)).ReadPublicIpsRequest(readPublicIpRequest).Execute()
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

	readPublicIpsResponse, httpRes, err := s.tenant.Client().PublicIpApi.ReadPublicIps(s.tenant.ContextWithAuth(ctx)).ReadPublicIpsRequest(readPublicIpRequest).Execute()
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

// PoolName returns the pool name from a public IP or "" if none.
func PoolName(ip *osc.PublicIp) string {
	for _, t := range ip.GetTags() {
		if t.Key == PublicIPPoolTag {
			return t.GetValue()
		}
	}
	return ""
}
