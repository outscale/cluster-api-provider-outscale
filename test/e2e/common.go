/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2" //nolint
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework"
)

const (
	KubernetesVersion = "KUBERNETES_VERSION"
)

// Create the namespace
func createNamespace(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, timeout string, interval string) *corev1.Namespace {
	By(fmt.Sprintf("Creating a namespace %s for bootstrap cluster", specName))
	namespace := framework.CreateNamespace(ctx, framework.CreateNamespaceInput{
		Creator: clusterProxy.GetClient(),
		Name:    specName,
	}, timeout, interval)
	return namespace
}
