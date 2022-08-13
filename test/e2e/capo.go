package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	utils "github.com/outscale-dev/cluster-api-provider-outscale.git/test/e2e/utils"
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

type CapoClusterMachineDeploymentSpecInput struct {
	E2EConfig             *clusterctl.E2EConfig
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool
	Flavor                string
	WorkerMachineCount    int64
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

func CapoClusterMachineDeploymentSpec(ctx context.Context, inputGetter func() CapoClusterMachineDeploymentSpecInput) {
	var (
		specName         = "capo"
		input            CapoClusterMachineDeploymentSpecInput
	//	namespace        *corev1.Namespace
	//	cancelWatches    context.CancelFunc
	//	clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
	//	clusterName      string
	)

	BeforeEach(func() {
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		Expect(input.ClusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. input.ClusterConfigPath must be an existing file when calling %s spec", specName)
		Expect(input.BootstrapClusterProxy).ToNot(BeNil(), "Invalid argument. input.BoostrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		Expect(input.E2EConfig.Variables).To(HaveKey(KubernetesVersion))
//		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
//		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
//		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
	})
	It("Should sucessfully create a cluster with one control planes", func() {
		By("Creating a workload cluster")
		ctx := context.Background()
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    k8sClient,
			Name:      "harbor-chartmuseum",
			Namespace: "harbor",
		})

		By("PASSED!")

		
	/*	clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
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
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)
		time.Sleep(10 * time.Minute)
*/
		By("Check configmap is ready")
//		clusterNamespace := namespace.Name
//		clusterName := clusterResources.Cluster.Name
			clusterNamespace := "default"
		clusterName := "hello-osc"
		workloadProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, clusterNamespace, clusterName)
		workloadClient := workloadProxy.GetClient()
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "kube-root-ca.crt",
			Namespace: "capi-kubeadm-bootstrap-system ",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "kube-root-ca.crt",	
			Namespace: "capi-kubeadm-control-plane-system",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "kube-root-ca.crt",
			Namespace: "capi-system",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "cluster-api-provider-outscale-manager-config  ",
			Namespace: "cluster-api-provider-outscale-system",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "controller-leader-election-capo",
			Namespace: "cluster-api-provider-outscale-system",
		})
		utils.WaitForConfigMapAvailable(ctx, utils.ConfigMapInput{
			Getter: k8sClient,
			Name: "kube-root-ca.crt",
			Namespace: "cluster-api-provider-outscale-system",
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kube-root-ca.crt",
			Namespace: "default",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kube-root-ca.crt",
			Namespace: "kube-node-lease",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "cluster-info",
			Namespace: "kube-public",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kube-root-ca.crt",
			Namespace: "kube-public",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "coredns",
			Namespace: "kube-system",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "extension-apiserver-authentication",
			Namespace: "kube-system",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kube-proxy",
			Namespace: "kube-system",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kube-root-ca.crt",
			Namespace: "kube-system",
		})

		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kubeadm-config",
			Namespace: "kube-system",
		})

		// based on version
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter:    workloadClient,
			Name:      "kubelet-config-1.22",
			Namespace: "kube-system",
		})
		utils.WaitForDeploymentAvailable(ctx, utils.DeploymentInput{
			Getter:    workloadClient,
			Name:      "calico-kube-controllers",
			Namespace: "kube-system",
		})
		utils.WaitForDaemonSetAvailable(ctx, utils.DaemonSetInput{
			Getter:    workloadClient,
			Name:      "calico-node",
			Namespace: "kube-system",
		})
		utils.WaitForDaemonSetAvailable(ctx, utils.DaemonSetInput{
			Getter:    workloadClient,
			Name:      "kube-proxy",
			Namespace: "kube-system",
		})
		utils.WaitForDaemonSetAvailable(ctx, utils.DaemonSetInput{
					Getter:    workloadClient,
					Name:      "osc-cloud-controller-manager",
					Namespace: "kube-system",
				})
	
		utils.WaitForServiceAvailable(ctx, utils.ServiceInput{
			Getter:    workloadClient,
			Name:      "kubernetes",
			Namespace: "default",
		})
		utils.WaitForServiceAvailable(ctx, utils.ServiceInput{
			Getter:    workloadClient,
			Name:      "kube-dns",
			Namespace: "kube-system",
		})
		utils.WaitForCreateSecretAvailable(ctx, utils.CreateSecretInput{
			Getter: workloadClient,	
			Name: "provisionner",
			Namespace: "default",
			DataKey: "provisionner",
			DataValue: "cluster-api",
		})
		utils.WaitForSecretsAvailable(ctx, utils.SecretInput{
			Getter: workloadClient,
			Name: "provisionner",
			Namespace: "default",
		})
		utils.WaitForCreateConfigMapAvailable(ctx, utils.CreateConfigMapInput{
			Getter: workloadClient,
			Name: "bootstrapper",
			Namespace: "default",
			DataKey: "bootstrapper",
			DataValue: "kubeadm",
		})
		utils.WaitForConfigMapsAvailable(ctx, utils.ConfigMapInput{
			Getter: workloadClient,
			Name: "bootstrapper",
			Namespace: "default",
		})
		utils.WaitForCreateDeploymentAvailable(ctx, utils.CreateDeploymentInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
			Image: "nginx:1.12",
			Port: 80,
			ConfigMapName: "bootstraper",
			ConfigMapKey: "bootstrapper",
		})
		utils.WaitForDeploymentAvailable(ctx, utils.DeploymentInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
		})
		utils.WaitForCreateDaemonSetAvailable(ctx, utils.CreateDaemonSetInput{
			Getter: workloadClient,
			Name: "nginx-daemonset",
			Namespace: "default",
			Image: "nginx:1.12",
			Port: 80,
			SecretName: "provisionner",
			SecretKey: "provisionner",
		})
		utils.WaitForDaemonSetAvailable(ctx, utils.DaemonSetInput{
			Getter: workloadClient,
			Name: "nginx-daemonset",
			Namespace: "default",
		})
		utils.WaitForCreateServiceAvailable(ctx, utils.CreateServiceInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
				Port:       80,
				TargetPort: 80,

		})
		utils.WaitForServiceAvailable(ctx, utils.ServiceInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
		})
			utils.WaitForCreateServiceAvailable(ctx, utils.CreateServiceInput{

			Getter: workloadClient,
			Name: "nginx-daemonset",
			Namespace: "default",
				Port:       80,
				TargetPort: 80,

		})
		utils.WaitForServiceAvailable(ctx, utils.ServiceInput{
			Getter: workloadClient,
			Name: "nginx-daemonset",
			Namespace: "default",
		})
		utils.WaitForDeleteServiceAvailable(ctx, utils.ServiceInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
		})
		utils.WaitForDeleteServiceAvailable(ctx, utils.ServiceInput{
			Getter: workloadClient,
			Name: "nginx-daemonset",
			Namespace: "default",
		})
		utils.WaitForDeleteDeploymentAvailable(ctx, utils.DeploymentInput{
			Getter: workloadClient,
			Name: "nginx-deployment",
			Namespace: "default",
		})
			utils.WaitForDeleteDaemonSetAvailable(ctx, utils.DaemonSetInput{
				Getter:    workloadClient,
				Name:      "nginx-daemonset",
				Namespace: "default",
			})
		utils.WaitForDeleteConfigMapAvailable(ctx, utils.ConfigMapInput{
			Getter: workloadClient,
			Name: "bootstrapper",
			Namespace: "default",
		})
		utils.WaitForDeleteSecretAvailable(ctx, utils.SecretInput{
			Getter: workloadClient,
			Name: "provisionner",
			Namespace: "default",
		})
		By("PASSED!")
	})
	AfterEach(func() {
//		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
