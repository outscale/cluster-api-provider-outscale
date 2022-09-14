package e2e

import (
	"context"
	. "github.com/onsi/ginkgo"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("[conformance] Test kubernetes conformance", func() {
	ctx := context.TODO()

	Context("Run the k8s conformance", func() {
		capi_e2e.K8SConformanceSpec(ctx, func() capi_e2e.K8SConformanceSpecInput {
			return capi_e2e.K8SConformanceSpecInput{
				E2EConfig:             e2eConfig,
				ClusterctlConfigPath:  clusterctlConfigPath,
				BootstrapClusterProxy: bootstrapClusterProxy,
				ArtifactFolder:        artifactFolder,
				SkipCleanup:           skipCleanup,
				Flavor:                "with-clusterclass",
			}
		})
	})
})
