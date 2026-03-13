package e2e

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/samber/lo"
	"k8s.io/utils/ptr"
	capi_e2e "sigs.k8s.io/cluster-api/test/e2e"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	Context("Running the quick-start multiaz spec", func() {
		capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
			return capi_e2e.QuickStartSpecInput{
				E2EConfig:                e2eConfig,
				ClusterctlConfigPath:     clusterctlConfigPath,
				InfrastructureProvider:   &infraProvider,
				BootstrapClusterProxy:    bootstrapClusterProxy,
				ArtifactFolder:           artifactFolder,
				SkipCleanup:              skipCleanup,
				Flavor:                   ptr.To("multiaz"),
				ControlPlaneMachineCount: ptr.To[int64](3),
				WorkerMachineCount:       ptr.To[int64](4),
				PostMachinesProvisioned: func(managementClusterProxy framework.ClusterProxy, workloadClusterNamespace, workloadClusterName string) {
					var ms infrastructurev1beta1.OscMachineList
					err := managementClusterProxy.GetClient().List(ctx, &ms, client.InNamespace(workloadClusterNamespace))
					Expect(err).To(Succeed())
					Expect(ms.Items).To(HaveLen(5))
					workers := map[string]int{}
					cps := map[string]int{}
					for _, m := range ms.Items {
						Expect(m.Status.FailureDomain).To(Not(BeNil()))
						if _, found := m.GetLabels()["cluster.x-k8s.io/control-plane"]; found {
							cps[*m.Status.FailureDomain]++
						} else {
							workers[*m.Status.FailureDomain]++
						}
					}
					Expect(lo.Keys(workers)).To(HaveLen(2))
					Expect(lo.Max(lo.Values(workers)) - lo.Min(lo.Values(workers))).To(BeNumerically("<=", 1))
					Expect(lo.Keys(cps)).To(HaveLen(2))
					Expect(lo.Max(lo.Values(cps)) - lo.Min(lo.Values(cps))).To(BeNumerically("<=", 1))
				},
			}
		})
	})

	Context("Running the quick-start airgap spec", func() {
		capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
			return capi_e2e.QuickStartSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
				Flavor:                 ptr.To("airgap"),
			}
		})
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

	Context("Running the quick-start topology spec", func() {
		capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
			return capi_e2e.QuickStartSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
				Flavor:                 ptr.To("topology"),
			}
		})
	})

	Context("Running the quick-start public spec", func() {
		capi_e2e.QuickStartSpec(ctx, func() capi_e2e.QuickStartSpecInput {
			return capi_e2e.QuickStartSpecInput{
				E2EConfig:              e2eConfig,
				ClusterctlConfigPath:   clusterctlConfigPath,
				InfrastructureProvider: &infraProvider,
				BootstrapClusterProxy:  bootstrapClusterProxy,
				ArtifactFolder:         artifactFolder,
				SkipCleanup:            skipCleanup,
				Flavor:                 ptr.To("public"),
			}
		})
	})
})
