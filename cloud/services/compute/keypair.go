/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"

	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type KeypairInterface interface {
	CreateKeypair(ctx context.Context, name, clusterID string) (*osc.KeypairCreated, error)
	DeleteKeypair(ctx context.Context, name string) error
	GetKeypair(ctx context.Context, name string) (*osc.Keypair, error)
}

func (s *Service) CreateKeypair(ctx context.Context, name, clusterID string) (*osc.KeypairCreated, error) {
	resp, err := s.tenant.Client().CreateKeypair(ctx, osc.CreateKeypairRequest{
		KeypairName: name,
	})
	if err != nil {
		return nil, err
	}
	keypairID := *resp.Keypair.KeypairId
	resourceIds := []string{keypairID}
	clusterTag := osc.ResourceTag{
		Key:   tags.ClusterIDKey(clusterID),
		Value: string(tags.ResourceLifecycleOwned),
	}
	KeypairTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err = s.tags.AddTag(ctx, KeypairTagRequest, resourceIds)
	if err != nil {
		return nil, err
	}
	return resp.Keypair, nil
}

// DeleteKeypair delete machine Keypair
func (s *Service) DeleteKeypair(ctx context.Context, name string) error {
	req := osc.DeleteKeypairRequest{KeypairName: &name}
	_, err := s.tenant.Client().DeleteKeypair(ctx, req)
	return err
}

// GetKeypair retrieve Keypair from KeypairId
func (s *Service) GetKeypair(ctx context.Context, name string) (*osc.Keypair, error) {
	req := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{
			KeypairNames: &[]string{name},
		},
	}

	resp, err := s.tenant.Client().ReadKeypairs(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(ptr.From(resp.Keypairs)) == 0:
		return nil, nil
	default:
		return &ptr.From(resp.Keypairs)[0], nil
	}
}

var _ KeypairInterface = (*Service)(nil)
