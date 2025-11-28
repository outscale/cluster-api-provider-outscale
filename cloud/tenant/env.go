/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"context"
	"errors"

	osc "github.com/outscale/osc-sdk-go/v2"
)

func TenantFromEnv() (Tenant, error) {
	return TenantFromConfigEnv(osc.NewConfigEnv())
}

func TenantFromConfigEnv(cfg *osc.ConfigEnv) (Tenant, error) {
	t := &envTenant{
		cfg: cfg,
	}
	// checking config
	_, err := t.cfg.Configuration()
	if err == nil {
		_, err = t.cfg.Context(context.TODO())
	}
	if err == nil && t.cfg.Region == nil {
		err = errors.New("OSC_REGION is not set")
	}
	return t, err
}

type envTenant struct {
	cfg *osc.ConfigEnv
}

func (t *envTenant) Region() string {
	return *t.cfg.Region
}

func (t *envTenant) ContextWithAuth(ctx context.Context) context.Context {
	ctx, err := t.cfg.Context(ctx)
	if err != nil {
		panic(err) // should never occur, as TenantFromEnv has checked that Context() does not return an error
	}
	return ctx
}

func (t *envTenant) Client() *osc.APIClient {
	cfg, err := t.cfg.Configuration()
	if err != nil {
		panic(err) // should never occur, as TenantFromEnv has checked that Configuration() does not return an error
	}
	cfg.UserAgent = userAgent()
	return osc.NewAPIClient(cfg)
}

var _ Tenant = (*envTenant)(nil)
