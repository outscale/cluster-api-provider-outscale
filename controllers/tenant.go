/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	"github.com/outscale/osc-sdk-go/v3/pkg/profile"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func getTenant(ctx context.Context, cl client.Client, c services.Servicer, cluster *infrastructurev1beta2.OscCluster) (tenant.Tenant, error) {
	logger := log.FromContext(ctx).V(4)
	switch {
	case cluster.Spec.Credentials.FromFile != "":
		logger.Info("Using tenant from file", "file", cluster.Spec.Credentials.FromFile, "profile", cluster.Spec.Credentials.Profile)
		return tenant.FromFile(cluster.Spec.Credentials.FromFile, cluster.Spec.Credentials.Profile)
	case cluster.Spec.Credentials.FromSecret != "":
		logger.Info("Using tenant from secret", "secret", cluster.Spec.Credentials.FromSecret)
		return getTenantFromSecret(ctx, cl, cluster.Spec.Credentials.FromSecret, cluster.Namespace)
	default:
		logger.Info("Using default tenant")
		return c.DefaultTenant()
	}
}

func getMgmtTenant(ctx context.Context, cl client.Client, c services.Servicer, cluster *infrastructurev1beta2.OscCluster) (tenant.Tenant, error) {
	logger := log.FromContext(ctx).V(4)
	creds := cluster.Spec.NetPeering.ManagementCredentials
	switch {
	case creds.FromFile != "":
		logger.Info("Using tenant from file for management cluster", "file", creds.FromFile, "profile", creds.Profile)
		return tenant.FromFile(creds.FromFile, creds.Profile)
	case creds.FromSecret != "":
		logger.Info("Using tenant from secret for management cluster", "secret", creds.FromSecret)
		return getTenantFromSecret(ctx, cl, creds.FromSecret, cluster.Namespace)
	default:
		logger.Info("Using default tenant for management cluster")
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
	return tenant.FromProfile(&profile.Profile{
		AccessKey: string(secret.Data["access_key"]),
		SecretKey: string(secret.Data["secret_key"]),
		Region:    string(secret.Data["region"]),
	})
}
