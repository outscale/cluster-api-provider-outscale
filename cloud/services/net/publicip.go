/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package net

import (
	"context"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
)

const (
	PublicIPPoolTag = "OscK8sIPPool"
	NoDeleteTag     = "OscK8sNoDelete"
)

//go:generate ../../../bin/mockgen -destination mock_net/publicip_mock.go -package mock_net -source ./publicip.go
type OscPublicIpInterface interface {
	CreatePublicIp(ctx context.Context, publicIpName, clusterID string) (*osc.PublicIp, error)
	DeletePublicIp(ctx context.Context, publicIpId string) error
	GetPublicIp(ctx context.Context, publicIpId string) (*osc.PublicIp, error)
	GetPublicIpByIp(ctx context.Context, publicIp string) (*osc.PublicIp, error)
	ListPublicIpsFromPool(ctx context.Context, pool string) ([]osc.PublicIp, error)
}

// CreatePublicIp retrieve a publicip associated with you account
func (s *Service) CreatePublicIp(ctx context.Context, publicIpName string, clusterID string) (*osc.PublicIp, error) {
	resp, err := s.tenant.Client().CreatePublicIp(ctx, osc.CreatePublicIpRequest{})
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
	resourceIds := []string{resp.PublicIp.PublicIpId}
	req := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{publicIpTag, clusterTag},
	}

	err = tag.AddTag(ctx, req, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}
	return resp.PublicIp, nil
}

// DeletePublicIp release the public ip
func (s *Service) DeletePublicIp(ctx context.Context, publicIpId string) error {
	req := osc.DeletePublicIpRequest{
		PublicIpId: &publicIpId,
	}

	_, err := s.tenant.Client().DeletePublicIp(ctx, req)
	return err
}

// GetPublicIp get a public ip object using a public ip id
func (s *Service) GetPublicIp(ctx context.Context, publicIpId string) (*osc.PublicIp, error) {
	req := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &[]string{publicIpId},
		},
	}

	resp, err := s.tenant.Client().ReadPublicIps(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.PublicIps) == 0:
		return nil, nil
	default:
		return &(*resp.PublicIps)[0], nil
	}
}

// GetPublicIpByIp get a public ip object using a public ip
func (s *Service) GetPublicIpByIp(ctx context.Context, publicIp string) (*osc.PublicIp, error) {
	req := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIps: &[]string{publicIp},
		},
	}

	resp, err := s.tenant.Client().ReadPublicIps(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.PublicIps) == 0:
		return nil, nil
	default:
		return &(*resp.PublicIps)[0], nil
	}
}

func (s *Service) ListPublicIpsFromPool(ctx context.Context, pool string) ([]osc.PublicIp, error) {
	req := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			TagKeys:   &[]string{PublicIPPoolTag},
			TagValues: &[]string{pool},
		},
	}

	resp, err := s.tenant.Client().ReadPublicIps(ctx, req)
	if err != nil {
		return nil, err
	}
	return *resp.PublicIps, nil
}

// ValidatePublicIpIds validate the list of id by checking each public ip resource and return only  public ip resource id that currently exist.
func (s *Service) ValidatePublicIpIds(ctx context.Context, publicIpIds []string) ([]string, error) {
	req := osc.ReadPublicIpsRequest{
		Filters: &osc.FiltersPublicIp{
			PublicIpIds: &publicIpIds,
		},
	}

	resp, err := s.tenant.Client().ReadPublicIps(ctx, req)
	if err != nil {
		return nil, err
	}
	return lo.Map(*resp.PublicIps, func(ip osc.PublicIp, _ int) string {
		return ip.PublicIpId
	}), nil
}

// CanDelete returns the true if the IP can be deleted (no nodelete tag, no pool tag)
func CanDelete(ip *osc.PublicIp) bool {
	return !tags.Has(ip.Tags, tags.PublicIPPool) && !tags.Has(ip.Tags, NoDeleteTag)
}
