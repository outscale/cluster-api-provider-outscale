/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"
	"errors"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
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
	netPeeringRequest := osc.CreateNetPeeringRequest{
		SourceNetId:     netID,
		AccepterNetId:   mgmtNetID,
		AccepterOwnerId: &mgmtAccountID,
	}

	netPeeringResponse, httpRes, err := s.tenant.Client().NetPeeringApi.CreateNetPeering(s.tenant.ContextWithAuth(ctx)).CreateNetPeeringRequest(netPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateNetPeering", netPeeringRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*netPeeringResponse.NetPeering.NetPeeringId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netPeeringTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err = tag.AddTag(ctx, netPeeringTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	netPeering, ok := netPeeringResponse.GetNetPeeringOk()
	if !ok {
		return nil, errors.New("cannot create netPeering")
	}
	return netPeering, nil
}

// AcceptNetPeering accepts a net peering
func (s *Service) AcceptNetPeering(ctx context.Context, netPeeringID string) error {
	acceptNetPeeringRequest := osc.AcceptNetPeeringRequest{NetPeeringId: netPeeringID}

	_, httpRes, err := s.tenant.Client().NetPeeringApi.AcceptNetPeering(s.tenant.ContextWithAuth(ctx)).AcceptNetPeeringRequest(acceptNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "AcceptNetPeering", acceptNetPeeringRequest, httpRes, err)
	return err
}

// DeleteNetPeering deletes a net peering
func (s *Service) DeleteNetPeering(ctx context.Context, netPeeringID string) error {
	deleteNetPeeringRequest := osc.DeleteNetPeeringRequest{NetPeeringId: netPeeringID}

	_, httpRes, err := s.tenant.Client().NetPeeringApi.DeleteNetPeering(s.tenant.ContextWithAuth(ctx)).DeleteNetPeeringRequest(deleteNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteNetPeering", deleteNetPeeringRequest, httpRes, err)
	return err
}

// GetNetPeering retrieves a net peering
func (s *Service) GetNetPeering(ctx context.Context, netPeeringID string) (*osc.NetPeering, error) {
	readNetPeeringRequest := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			NetPeeringIds: &[]string{netPeeringID},
		},
	}

	readNetPeeringResponse, httpRes, err := s.tenant.Client().NetPeeringApi.ReadNetPeerings(s.tenant.ContextWithAuth(ctx)).ReadNetPeeringsRequest(readNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetPeerings", readNetPeeringRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netPeerings, ok := readNetPeeringResponse.GetNetPeeringsOk()
	if !ok {
		return nil, errors.New("cannot get netPeering")
	}
	if len(*netPeerings) == 0 {
		return nil, nil
	} else {
		netPeering := *netPeerings
		return &netPeering[0], nil
	}
}

// GetNetPeeringFromNet retrieves a net peering from its info
func (s *Service) GetNetPeeringFromNet(ctx context.Context, netID, mgmtNetID, mgmtAccountID string) (*osc.NetPeering, error) {
	readNetPeeringRequest := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			SourceNetNetIds:       &[]string{netID},
			AccepterNetNetIds:     &[]string{mgmtNetID},
			AccepterNetAccountIds: &[]string{mgmtAccountID},
		},
	}

	readNetPeeringResponse, httpRes, err := s.tenant.Client().NetPeeringApi.ReadNetPeerings(s.tenant.ContextWithAuth(ctx)).ReadNetPeeringsRequest(readNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetPeerings", readNetPeeringRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netPeerings, ok := readNetPeeringResponse.GetNetPeeringsOk()
	if !ok {
		return nil, errors.New("cannot get netPeering")
	}
	if len(*netPeerings) == 0 {
		return nil, nil
	} else {
		netPeering := *netPeerings
		return &netPeering[0], nil
	}
}

// ListNetPeerings lists all net peerings in a net.
func (s *Service) ListNetPeerings(ctx context.Context, netId string) ([]osc.NetPeering, error) {
	var netPeerings []osc.NetPeering

	readNetPeeringRequest := osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			SourceNetNetIds: &[]string{netId},
		},
	}

	readNetPeeringResponse, httpRes, err := s.tenant.Client().NetPeeringApi.ReadNetPeerings(s.tenant.ContextWithAuth(ctx)).ReadNetPeeringsRequest(readNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetPeerings", readNetPeeringRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netPeerings = append(netPeerings, readNetPeeringResponse.GetNetPeerings()...)

	readNetPeeringRequest = osc.ReadNetPeeringsRequest{
		Filters: &osc.FiltersNetPeering{
			AccepterNetNetIds: &[]string{netId},
		},
	}

	readNetPeeringResponse, httpRes, err = s.tenant.Client().NetPeeringApi.ReadNetPeerings(s.tenant.ContextWithAuth(ctx)).ReadNetPeeringsRequest(readNetPeeringRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNetPeerings", readNetPeeringRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	netPeerings = append(netPeerings, readNetPeeringResponse.GetNetPeerings()...)

	return netPeerings, nil
}
