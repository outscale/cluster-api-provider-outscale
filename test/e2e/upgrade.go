package e2e

import (
	"context"
	. "github.com/onsi/ginkgo"
	"k8s.io/utils/pointer"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
)

var _ = Describe("[upgrade] Test Upgrade Kuberneted Version", func() {
	ctx := context.TODO()
	Context("[first_upgrade] Running upgrade of kubernetes with one simple control plane", func() {
		capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
			return capi_e2e.ClusterUpgradeConformanceSpecInput{
				E2EConfig:                e2eConfig,
				ClusterctlConfigPath:     clusterctlConfigPath,
				BootstrapClusterProxy:    bootstrapClusterProxy,
				ArtifactFolder:           artifactFolder,
				SkipCleanup:              skipCleanup,
				SkipConformanceTests:     true,
				ControlPlaneMachineCount: pointer.Int64(1),
				WorkerMachineCount:       pointer.Int64(1),
				Flavor:                   pointer.String("upgrade"),
			}
		})
	})
	Context("[upgrade] Upgrade only KCP on HA cluster", func() {
		capi_e2e.ClusterUpgradeConformanceSpec(context.TODO(), func() capi_e2e.ClusterUpgradeConformanceSpecInput {
			return capi_e2e.ClusterUpgradeConformanceSpecInput{
				E2EConfig:                e2eConfig,
				ClusterctlConfigPath:     clusterctlConfigPath,
				BootstrapClusterProxy:    bootstrapClusterProxy,
				ArtifactFolder:           artifactFolder,
				ControlPlaneMachineCount: pointer.Int64(3),
				WorkerMachineCount:       pointer.Int64(0),
				SkipCleanup:              skipCleanup,
				SkipConformanceTests:     true,
			}
		})
	})
	Context("[upgrade] Upgrade HA controlPlane and scale-in", func() {
		capi_e2e.ClusterUpgradeConformanceSpec(ctx, func() capi_e2e.ClusterUpgradeConformanceSpecInput {
			return capi_e2e.ClusterUpgradeConformanceSpecInput{
				E2EConfig:                e2eConfig,
				ClusterctlConfigPath:     clusterctlConfigPath,
				BootstrapClusterProxy:    bootstrapClusterProxy,
				ArtifactFolder:           artifactFolder,
				SkipCleanup:              skipCleanup,
				SkipConformanceTests:     true,
				ControlPlaneMachineCount: pointer.Int64(3),
				WorkerMachineCount:       pointer.Int64(1),
				Flavor:                   pointer.String("upgrade-scale-in"),
			}
		})
	})

})
