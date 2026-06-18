/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type ImageInterface interface {
	GetImage(ctx context.Context, id string) (*osc.Image, error)
	GetImageByName(ctx context.Context, name, accountId string) (*osc.Image, error)
}

// GetImage retrieves an image by id
func (s *Service) GetImage(ctx context.Context, id string) (*osc.Image, error) {
	req := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageIds: &[]string{id}},
	}

	resp, err := s.tenant.Client().ReadImages(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Images) == 0:
		return nil, nil
	default:
		return &(*resp.Images)[0], nil
	}
}

// GetImageByName retrieves image by name/owner
func (s *Service) GetImageByName(ctx context.Context, name, accountId string) (*osc.Image, error) {
	req := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{
			ImageNames: &[]string{name},
		},
	}
	if accountId != "" {
		req.Filters.AccountIds = &[]string{accountId}
	}

	resp, err := s.tenant.Client().ReadImages(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Images) == 0:
		return nil, nil
	default:
		return &(*resp.Images)[0], nil
	}
}
