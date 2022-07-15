package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

type CapoClusterDeploymentSpecInput struct {
	E2EConfig             *clusterctl.E2EConfig
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool
	Flavor                string
}

type CreateClusterAndWaitInput struct {
	ClusterProxy            framework.ClusterProxy
	ConfigCluster           clusterctl.ConfigClusterInput
	WaitForClusterIntervals []interface{}
	Args                    []string
}

type CreateClusterAndWaitResult struct {
	Cluster *clusterv1.Cluster
}

// CreateClusterAndWait create cluster infrastructure and wait to be provisionned
func CreateClusterAndWait(ctx context.Context, input CreateClusterAndWaitInput, result *CreateClusterAndWaitResult) {
	Expect(ctx).NotTo(BeNil(), "ctx is required for CreateClusterAndWait")
	Expect(input.ClusterProxy).ToNot(BeNil(), "Invalid argument. input.ClusterProxy can't be nil when calling CreateClusterAndWait")
	Expect(result).ToNot(BeNil(), "Invalid argument. result can't be nil when calling CreateClusterTemplateAndWait")
	Expect(input.ConfigCluster.ControlPlaneMachineCount).ToNot(BeNil())
	Expect(input.ConfigCluster.WorkerMachineCount).ToNot(BeNil())

	log.Logf("Creating the workload cluster with name %q using the %q template (Kubernetes %s, %d control-plane machines, %d worker machines)",
		input.ConfigCluster.ClusterName, input.ConfigCluster.Flavor, input.ConfigCluster.KubernetesVersion, *input.ConfigCluster.ControlPlaneMachineCount, *input.ConfigCluster.WorkerMachineCount)
	log.Logf("Getting the cluster template yaml")
	workloadClusterTemplate := clusterctl.ConfigCluster(ctx, clusterctl.ConfigClusterInput{
		KubeconfigPath:           input.ConfigCluster.KubeconfigPath,
		ClusterctlConfigPath:     input.ConfigCluster.ClusterctlConfigPath,
		Flavor:                   input.ConfigCluster.Flavor,
		Namespace:                input.ConfigCluster.Namespace,
		ClusterName:              input.ConfigCluster.ClusterName,
		KubernetesVersion:        input.ConfigCluster.KubernetesVersion,
		ControlPlaneMachineCount: input.ConfigCluster.ControlPlaneMachineCount,
		WorkerMachineCount:       input.ConfigCluster.WorkerMachineCount,
		InfrastructureProvider:   input.ConfigCluster.InfrastructureProvider,
		LogFolder:                input.ConfigCluster.LogFolder,
	})
	Expect(workloadClusterTemplate).ToNot(BeNil(), "Failed to get the cluster template")
	log.Logf("Applying the cluster template yaml to the cluster")
	Eventually(func() error {
		return input.ClusterProxy.Apply(ctx, workloadClusterTemplate, input.Args...)
	}, 10*time.Second).Should(Succeed(), "Failed to apply the cluster template")

	log.Logf("Waiting for the cluster infrastructure to be provisioned")
	result.Cluster = framework.DiscoveryAndWaitForCluster(ctx, framework.DiscoveryAndWaitForClusterInput{
		Getter:    input.ClusterProxy.GetClient(),
		Namespace: input.ConfigCluster.Namespace,
		Name:      input.ConfigCluster.ClusterName,
	}, input.WaitForClusterIntervals...)
}

// CapoClusterDeploymentSpec create infrastructure cluster using its generated config and wait infrastructure cluster to be provisionned
func CapoClusterDeploymentSpec(ctx context.Context, inputGetter func() CapoClusterDeploymentSpecInput) {
	var (
		specName         = "capo"
		input            CapoClusterDeploymentSpecInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *CreateClusterAndWaitResult
		clusterName      string
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BoostrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))
		clusterResources = new(CreateClusterAndWaitResult)
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
	})

	It("Should sucessfully create an infrastructure cluster", func() {
		By("creating an infrastructure cluster")
		CreateClusterAndWait(ctx, CreateClusterAndWaitInput{
			ClusterProxy: input.BootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   input.Flavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(3),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals: input.E2EConfig.GetIntervals(specName, "wait-cluster"),
		}, clusterResources)
		By("Passed!")
	})

	AfterEach(func() {
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
