/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
)

type NetPeeringInterface interface {
	CreateNetPeering(ctx context.Context, netID, mgmtNetID string, mgmtAccountID *string, clusterID string) (*osc.NetPeering, error)
	AcceptNetPeering(ctx context.Context, netPeeringID string) error
	DeleteNetPeering(ctx context.Context, netPeeringID string) error
	GetNetPeering(ctx context.Context, netPeeringID string) (*osc.NetPeering, error)
	GetNetPeeringFromNet(ctx context.Context, netID, mgmtNetID, mgmtAccountID string) (*osc.NetPeering, error)
	ListNetPeerings(ctx context.Context, netId, clusterID string) ([]osc.NetPeering, error)
}

// CreateNetPeering creates a net peering
func (s *Service) CreateNetPeering(ctx context.Context, hubNetID, spokeNetID string, spokeAccountID *string, clusterID string) (*osc.NetPeering, error) {
	req := osc.CreateNetPeeringRequest{
		SourceNetId:     hubNetID,
		AccepterNetId:   spokeNetID,
		AccepterOwnerId: spokeAccountID,
	}

	resp, err := s.tenant.Client().CreateNetPeering(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.NetPeering.NetPeeringId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netPeeringTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err = s.tags.AddTag(ctx, netPeeringTagRequest)
	if err != nil {
		return nil, err
	}
	return resp.NetPeering, nil
}

// AcceptNetPeering accepts a net peering
func (s *Service) AcceptNetPeering(ctx context.Context, netPeeringID string) error {
	req := osc.AcceptNetPeeringRequest{NetPeeringId: netPeeringID}
	_, err := s.tenant.Client().AcceptNetPeering(ctx, req)
	return err
}

// DeleteNetPeering deletes a net peering
func (s *Service) DeleteNetPeering(ctx context.Context, netPeeringID string) error {
	req := osc.DeleteNetPeeringRequest{NetPeeringId: netPeeringID}
	_, err := s.tenant.Client().DeleteNetPeering(ctx, req)
	return err
}

// GetNetPeering retrieves a net peering
func (s *Service) GetNetPeering(ctx context.Context, netPeeringID string) (*osc.NetPeering, error) {
	req := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			NetPeeringIds: &[]string{netPeeringID},
		},
	}

	resp, err := s.tenant.Client().ReadNetPeerings(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NetPeerings) == 0:
		return nil, nil
	default:
		return &(*resp.NetPeerings)[0], nil
	}
}

// GetNetPeeringFromNet retrieves a net peering from its info
func (s *Service) GetNetPeeringFromNet(ctx context.Context, netID, mgmtNetID, mgmtAccountID string) (*osc.NetPeering, error) {
	req := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			SourceNetNetIds:       &[]string{netID},
			AccepterNetNetIds:     &[]string{mgmtNetID},
			AccepterNetAccountIds: &[]string{mgmtAccountID},
		},
	}

	resp, err := s.tenant.Client().ReadNetPeerings(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.NetPeerings) == 0:
		return nil, nil
	default:
		return &(*resp.NetPeerings)[0], nil
	}
}

// ListNetPeerings lists all net peerings in a net, owned by a cluster.
func (s *Service) ListNetPeerings(ctx context.Context, netId, clusterID string) ([]osc.NetPeering, error) {
	req := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			Tags: &[]string{"OscK8sClusterID/" + clusterID + "=owned"},
		},
	}

	resp, err := s.tenant.Client().ReadNetPeerings(ctx, req)
	if err != nil {
		return nil, err
	}
	return lo.Filter(ptr.From(resp.NetPeerings), func(np osc.NetPeering, _ int) bool {
		return ptr.From(np.SourceNet.NetId) == netId || ptr.From(np.AccepterNet.NetId) == netId
	}), nil
}
