package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("[quickstart][fast] Running the Cluster API quick start tests", func() {
	ctx := context.TODO()

	BeforeEach(func() {
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.CNIPath))
		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeFrom))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.CPMachineTemplateUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.WorkersMachineTemplateUpgradeTo))
	})

	Context("Running the quick-start spec", func() {
		capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
			return capi_e2e.QuickStartSpecInput{
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
