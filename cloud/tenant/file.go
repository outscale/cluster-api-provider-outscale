package tenant

import (
	"context"
	"fmt"

	osc "github.com/outscale/osc-sdk-go/v2"
)

func TenantFromFile(path, profile string) (Tenant, error) {
	if profile == "" {
		profile = "default"
	}
	cfg, err := osc.LoadConfigFile(&path)
	if err != nil {
		return nil, fmt.Errorf("from file: %w", err)
	}
	return &fileTenant{
		cfg:     cfg,
		profile: profile,
	}, nil
}

type fileTenant struct {
	cfg     *osc.ConfigFile
	profile string
}

func (t *fileTenant) Region() string {
	cfg, err := t.cfg.Configuration(t.profile)
	if err != nil {
		panic(err) // should never occur, as TenantFromFile has checked that Configuration() does not return an error
	}
	return cfg.Servers[0].Variables["region"].DefaultValue
}

func (t *fileTenant) ContextWithAuth(ctx context.Context) context.Context {
	ctx, err := t.cfg.Context(ctx, t.profile)
	if err != nil {
		panic(err) // should never occur, as TenantFromFile has checked that Context() does not return an error
	}
	return ctx
}

func (t *fileTenant) Client() *osc.APIClient {
	cfg, err := t.cfg.Configuration(t.profile)
	if err != nil {
		panic(err) // should never occur, as TenantFromFile has checked that Configuration() does not return an error
	}
	return osc.NewAPIClient(cfg)
}

var _ Tenant = (*fileTenant)(nil)
