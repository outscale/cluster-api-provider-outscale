/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("Node drain", func() {
	Context("[node-drain][all] Run node drain test", func() {
		capi_e2e.NodeDrainTimeoutSpec(context.TODO(), func() capi_e2e.NodeDrainTimeoutSpecInput {
			return capi_e2e.NodeDrainTimeoutSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
				Flavor:                 ptr.To("node-drain"),
			}
		})
	})
})
