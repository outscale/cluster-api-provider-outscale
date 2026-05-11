/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"context"
	"fmt"

	"github.com/outscale/osc-sdk-go/v3/pkg/options"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/outscale/osc-sdk-go/v3/pkg/profile"
)

func TenantFromFile(path, prof string) (Tenant, error) {
	if prof == "" {
		prof = "default"
	}
	cfg, err := profile.New(profile.FromFile(path, prof))
	if err != nil {
		return nil, fmt.Errorf("from file: %w", err)
	}
	return &fileTenant{
		profile: cfg,
	}, nil
}

type fileTenant struct {
	profile *profile.Profile
}

func (t *fileTenant) Region() string {
	return t.profile.Region
}

func (t *fileTenant) ContextWithAuth(ctx context.Context) context.Context {
	ctx, err := t.cfg.Context(ctx, t.profile)
	if err != nil {
		panic(err) // should never occur, as TenantFromFile has checked that Context() does not return an error
	}
	return ctx
}

func (t *fileTenant) Client() *osc.Client {
	cfg, err := t.cfg.Configuration(t.profile)
	if err != nil {
		panic(err) // should never occur, as TenantFromFile has checked that Configuration() does not return an error
	}
	cfg.UserAgent = userAgent()
	return osc.NewClient(cfg, options.WithUseragent(userAgent()))
}

var _ Tenant = (*fileTenant)(nil)
