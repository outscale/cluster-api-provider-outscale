/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package tenant

import (
	"errors"
	"fmt"

	"github.com/outscale/goutils/k8s/log"
	"github.com/outscale/osc-sdk-go/v3/pkg/middleware"
	"github.com/outscale/osc-sdk-go/v3/pkg/options"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/outscale/osc-sdk-go/v3/pkg/profile"
)

type Tenant interface {
	Profile() *profile.Profile
	Region() string
	Client() *osc.Client
}

type tenant struct {
	profile *profile.Profile
	client  *osc.Client
}

func (t *tenant) Profile() *profile.Profile {
	return t.profile
}

func (t *tenant) Region() string {
	return t.profile.Region
}

func (t *tenant) Client() *osc.Client {
	return t.client
}

func FromProfile(prof *profile.Profile) (Tenant, error) {
	c, err := newSDKClient(prof)
	if err != nil {
		return nil, err
	}
	return &tenant{profile: prof, client: c}, nil
}

func newSDKClient(prof *profile.Profile) (*osc.Client, error) {
	if prof.AccessKey == "" || prof.SecretKey == "" {
		return nil, errors.New("OSC_ACCESS_KEY/OSC_SECRET_KEY are required")
	}
	lg := log.OAPILogger{}
	copts := []middleware.MiddlewareChainOption{options.WithUseragent(userAgent()), options.WithLogging(lg)}
	// if len(opts) > 0 {
	// 	opt := opts[0]
	// 	// no default is set on RetryCount, it might be valid to run without backoff.
	// 	// ratelimiter is always configured. 0 values will be replaced by defaults.
	// 	err = mergo.Merge(&opt, Options{
	// 		RateLimit:    DefaultRateLimit,
	// 		RetryWaitMin: DefaultRetryWaitMin,
	// 		RetryWaitMax: DefaultRetryWaitMax,
	// 	})
	// 	if err != nil {
	// 		return nil, nil, fmt.Errorf("unable to set OAPI SDK options: %w", err)
	// 	}
	// 	copts = append(copts, opt.middleware()...)
	// }
	client, err := osc.NewClient(prof, copts...)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize OAPI client: %w", err)
	}
	return client, nil
}
