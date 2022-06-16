package e2e

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("Running the Cluster API E2E tests", func() {
	BeforeEach(func() {
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeFrom))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.KubernetesVersionUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.CPMachineTemplateUpgradeTo))
		Expect(e2eConfig.Variables).To(HaveKey(capi_e2e.WorkersMachineTemplateUpgradeTo))
	})

	Context("Running the capo cluster deployment spec", func() {
		CapoClusterDeploymentSpec(context.TODO(), func() CapoClusterDeploymentSpecInput {
			return CapoClusterDeploymentSpecInput{
				E2EConfig:             e2eConfig,
				ClusterctlConfigPath:  clusterctlConfigPath,
				BootstrapClusterProxy: bootstrapClusterProxy,
				ArtifactFolder:        artifactFolder,
				SkipCleanup:           skipCleanup,
				Flavor:                "",
			}
		})
	})

})
