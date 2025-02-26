package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("[ugrade][all] Running the Cluster API upgrade tests", func() {
	ctx := context.TODO()

	BeforeEach(func() {
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.CNIPath))
		Expect(e2eConfig.Variables).To(HaveKey(CCMPath))
		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeFrom))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.CPMachineTemplateUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.WorkersMachineTemplateUpgradeTo))
	})

	Context("Running the workload cluster upgrade spec", func() {
		capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
			return capi_e2e.ClusterUpgradeConformanceSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
				SkipConformanceTests:   true,
			}
		})
	})

	Context("Running KCP upgrade in a HA cluster", func() {
		capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
			return capi_e2e.ClusterUpgradeConformanceSpecInput{
				E2EConfig:                e2eConfig,
				ClusterctlConfigPath:     clusterctlConfigPath,
				InfrastructureProvider:   &infraProvider,
				BootstrapClusterProxy:    bootstrapClusterProxy,
				ArtifactFolder:           artifactFolder,
				ControlPlaneMachineCount: ptr.To[int64](3),
				WorkerMachineCount:       ptr.To[int64](1),
				SkipCleanup:              skipCleanup,
				SkipConformanceTests:     true,
			}
		})
	})
})
