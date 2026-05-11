/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

//go:generate ../../../bin/mockgen -destination mock_net/subnet_mock.go -package mock_net -source ./subnet.go
type OscSubnetInterface interface {
	CreateSubnet(ctx context.Context, spec infrastructurev1beta2.OscSubnet, netId, clusterID, subnetName string) (*osc.Subnet, error)
	DeleteSubnet(ctx context.Context, subnetId string) error
	GetSubnet(ctx context.Context, subnetId string) (*osc.Subnet, error)
	GetSubnetFromNet(ctx context.Context, netId, ipRange string) (*osc.Subnet, error)
}

// CreateSubnet create the subnet associate to the net
func (s *Service) CreateSubnet(ctx context.Context, spec infrastructurev1beta2.OscSubnet, netId, clusterID, subnetName string) (*osc.Subnet, error) {
	req := osc.CreateSubnetRequest{
		IpRange:       spec.IpRange,
		NetId:         netId,
		SubregionName: &spec.Subregion,
	}

	resp, err := s.tenant.Client().CreateSubnet(ctx, req)
	if err != nil {
		return nil, err
	}

	resourceIds := []string{resp.Subnet.SubnetId}
	subnetTag := osc.ResourceTag{
		Key:   "Name",
		Value: subnetName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	subnetTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        append(utils.RoleTags(spec.Roles), subnetTag, clusterTag),
	}
	err = tag.AddTag(ctx, subnetTagRequest, resourceIds, s.tenant.Client())
	if err != nil {
		return nil, err
	}

	return resp.Subnet, nil
}

// DeleteSubnet delete the subnet
func (s *Service) DeleteSubnet(ctx context.Context, subnetId string) error {
	req := osc.DeleteSubnetRequest{SubnetId: subnetId}

	_, err := s.tenant.Client().DeleteSubnet(ctx, req)
	return err
}

// GetSubnet retrieve Subnet object from subnet Id
func (s *Service) GetSubnet(ctx context.Context, subnetId string) (*osc.Subnet, error) {
	req := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			SubnetIds: &[]string{subnetId},
		},
	}

	resp, err := s.tenant.Client().ReadSubnets(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Subnets) == 0:
		return nil, nil
	default:
		return &(*resp.Subnets)[0], nil
	}
}

// GetSubnetFromNet finds the subnet having a specific range within a net.
func (s *Service) GetSubnetFromNet(ctx context.Context, netId, ipRange string) (*osc.Subnet, error) {
	req := osc.ReadSubnetsRequest{
		Filters: &osc.FiltersSubnet{
			NetIds:   &[]string{netId},
			IpRanges: &[]string{ipRange},
		},
	}

	resp, err := s.tenant.Client().ReadSubnets(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Subnets) == 0:
		return nil, nil
	default:
		return &(*resp.Subnets)[0], nil
	}
}
