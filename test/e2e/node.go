package e2e

import (
	"context"
	. "github.com/onsi/ginkgo"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("Node life", func() {
	ctx := context.TODO()
	kcpFlavor := "kcp-remediation"
	mdFlavor := "md-remediation"
	Context("[node-remediation] Run machine Remediation", func() {
		capi_e2e.MachineRemediationSpec(ctx, func() capi_e2e.MachineRemediationSpecInput {
			return capi_e2e.MachineRemediationSpecInput{
				E2EConfig:             e2eConfig,
				ClusterctlConfigPath:  clusterctlConfigPath,
				BootstrapClusterProxy: bootstrapClusterProxy,
				ArtifactFolder:        artifactFolder,
				SkipCleanup:           skipCleanup,
				KCPFlavor:             &kcpFlavor,
				MDFlavor:              &mdFlavor,
			}
		})
	})
	flavor := "node-drain"
	Context("[node-drain] Run node drain timeout", func() {
		capi_e2e.NodeDrainTimeoutSpec(ctx, func() capi_e2e.NodeDrainTimeoutSpecInput {
			return capi_e2e.NodeDrainTimeoutSpecInput{
				E2EConfig:             e2eConfig,
				ClusterctlConfigPath:  clusterctlConfigPath,
				BootstrapClusterProxy: bootstrapClusterProxy,
				ArtifactFolder:        artifactFolder,
				SkipCleanup:           skipCleanup,
				Flavor:                &flavor,
			}
		})
	})
})
