package e2e

import (
	"context"
	. "github.com/onsi/ginkgo"
//	"os"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
  //      utils "github.com/outscale-dev/cluster-api-provider-outscale.git/test/e2e/utils"
)

var _ = Describe("Node life", func() {
	ctx := context.TODO()
/*	const oscAccessKeyEnvVar = "OSC_ACCESS_KEY"
	const oscSecretKeyEnvVar = "OSC_SECRET_KEY"
	accessKey := os.Getenv(oscAccessKeyEnvVar)
	secretKey := os.Getenv(oscSecretKeyEnvVar)
        k8sClient := bootstrapClusterProxy.GetClient()

	utils.WaitForCreateMultiSecretAvailable(ctx, utils.CreateMultiSecretInput{
		Getter: k8sClient,
                Name: "cluster-api-provider-outscale",
                Namespace: "cluster-api-provider-outscale-system",
                DataFirstKey: "access_key",
                DataFirstValue: accessKey,
                DataSecondKey: "secret_key",
                DataSecondValue: secretKey,
	})
*/
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
