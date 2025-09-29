/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("[conformance] Test kubernetes conformance", func() {
	Context("Run the k8s conformance", func() {
		capi_e2e.K8SConformanceSpec(context.TODO(), func() capi_e2e.K8SConformanceSpecInput {
			return capi_e2e.K8SConformanceSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
			}
		})
	})
})
