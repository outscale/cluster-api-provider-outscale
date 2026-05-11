/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

//go:generate ../../../bin/mockgen -destination mock_net/netpeering_mock.go -package mock_net -source ./netpeering.go

type OscNetPeeringInterface interface {
	CreateNetPeering(ctx context.Context, netID, mgmtNetID, mgmtAccountID, clusterID string) (*osc.NetPeering, error)
	AcceptNetPeering(ctx context.Context, netPeeringID string) error
	DeleteNetPeering(ctx context.Context, netPeeringID string) error
	GetNetPeering(ctx context.Context, netPeeringID string) (*osc.NetPeering, error)
	GetNetPeeringFromNet(ctx context.Context, netID, mgmtNetID, mgmtAccountID string) (*osc.NetPeering, error)
	ListNetPeerings(ctx context.Context, netId string) ([]osc.NetPeering, error)
}

// CreateNetPeering creates a net peering
func (s *Service) CreateNetPeering(ctx context.Context, netID, mgmtNetID, mgmtAccountID, clusterID string) (*osc.NetPeering, error) {
	req := osc.CreateNetPeeringRequest{
		SourceNetId:     netID,
		AccepterNetId:   mgmtNetID,
		AccepterOwnerId: &mgmtAccountID,
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
	err = tag.AddTag(ctx, netPeeringTagRequest, resourceIds, s.tenant.Client())
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

// ListNetPeerings lists all net peerings in a net.
func (s *Service) ListNetPeerings(ctx context.Context, netId string) ([]osc.NetPeering, error) {
	req := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			SourceNetNetIds: &[]string{netId},
		},
	}

	resp, err := s.tenant.Client().ReadNetPeerings(ctx, req)
	if err != nil {
		return nil, err
	}
	netPeerings := *resp.NetPeerings

	req = osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			AccepterNetNetIds: &[]string{netId},
		},
	}

	resp, err = s.tenant.Client().ReadNetPeerings(ctx, req)
	if err != nil {
		return nil, err
	}
	netPeerings = append(netPeerings, *resp.NetPeerings...)

	return netPeerings, nil
}
