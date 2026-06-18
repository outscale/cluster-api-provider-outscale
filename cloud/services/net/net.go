/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type NetInterface interface {
	CreateNet(ctx context.Context, spec infrastructurev1beta1.OscNet, clusterID, netName string) (*osc.Net, error)
	DeleteNet(ctx context.Context, netId string) error
	GetNet(ctx context.Context, netId string) (*osc.Net, error)
}

// CreateNet create the net from spec (in order to retrieve ip range)
func (s *Service) CreateNet(ctx context.Context, spec infrastructurev1beta1.OscNet, clusterID, netName string) (*osc.Net, error) {
	req := osc.CreateNetRequest{
		IpRange: spec.IpRange,
	}

	resp, err := s.tenant.Client().CreateNet(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.Net.NetId}
	netTag := osc.ResourceTag{
		Key:   "Name",
		Value: netName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{netTag, clusterTag},
	}

	err = s.tags.AddTag(ctx, netTagRequest, resourceIds)
	if err != nil {
		return nil, err
	}
	return resp.Net, nil
}

// DeleteNet delete the net
func (s *Service) DeleteNet(ctx context.Context, netId string) error {
	req := osc.DeleteNetRequest{NetId: netId}
	_, err := s.tenant.Client().DeleteNet(ctx, req)
	return err
}

// GetNet retrieve the net object using the net id
func (s *Service) GetNet(ctx context.Context, netId string) (*osc.Net, error) {
	req := osc.ReadNetsRequest{
		Filters: &osc.FiltersNet{
			NetIds: &[]string{netId},
		},
	}

	resp, err := s.tenant.Client().ReadNets(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Nets) == 0:
		return nil, nil
	default:
		return &(*resp.Nets)[0], nil
	}
}
