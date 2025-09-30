/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	"github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getTenant(ctx context.Context, cl client.Client, c services.Servicer, cluster *infrastructurev1beta1.OscCluster) (tenant.Tenant, error) {
	switch {
	case cluster.Spec.Credentials.FromFile != "":
		return tenant.TenantFromFile(cluster.Spec.Credentials.FromFile, cluster.Spec.Credentials.Profile)
	case cluster.Spec.Credentials.FromSecret != "":
		return getTenantFromSecret(ctx, cl, cluster.Spec.Credentials.FromSecret, cluster.Namespace)
	default:
		return c.DefaultTenant()
	}
}

func getMgmtTenant(ctx context.Context, cl client.Client, c services.Servicer, cluster *infrastructurev1beta1.OscCluster) (tenant.Tenant, error) {
	creds := cluster.Spec.Network.NetPeering.ManagementCredentials
	switch {
	case creds.FromFile != "":
		return tenant.TenantFromFile(creds.FromFile, creds.Profile)
	case creds.FromSecret != "":
		return getTenantFromSecret(ctx, cl, creds.FromSecret, cluster.Namespace)
	default:
		return c.DefaultTenant()
	}
}

func getTenantFromSecret(ctx context.Context, cl client.Client, name, ns string) (tenant.Tenant, error) {
	var secret corev1.Secret
	err := cl.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: ns,
	}, &secret)
	if err != nil {
		return nil, fmt.Errorf("tenant from secret: %w", err)
	}
	return tenant.TenantFromConfigEnv(&osc.ConfigEnv{
		AccessKey: ptr.To(string(secret.Data["access_key"])),
		SecretKey: ptr.To(string(secret.Data["secret_key"])),
		Region:    ptr.To(string(secret.Data["region"])),
	})
}
