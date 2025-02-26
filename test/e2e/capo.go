/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	utils "github.com/outscale/cluster-api-provider-outscale/test/e2e/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	// Apply the manifests using the Kubernetes client from the ClusterProxy
	k8sClient := input.ClusterProxy.GetClient()

	reader := bytes.NewReader(workloadClusterTemplate)
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	for {
		obj := &unstructured.Unstructured{}
		err := decoder.Decode(obj)
		if errors.Is(err, io.EOF) {
			break
		}
		Expect(err).NotTo(HaveOccurred(), "Failed to decode cluster template")
		gvk := obj.GroupVersionKind()
		obj.SetGroupVersionKind(gvk)
		// Set the namespace for namespaced resources
		if obj.GetNamespace() == "" {
			obj.SetNamespace(input.ConfigCluster.Namespace)
		}
		// Use Create or Update to apply the resource
		err = k8sClient.Create(ctx, obj)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				Expect(err).NotTo(HaveOccurred(), "Failed to apply resource")
			}
		}
	}
	// Wait for the cluster to be ready
	Eventually(func() bool {
		cluster := &clusterv1.Cluster{}
		key := types.NamespacedName{Namespace: input.ConfigCluster.Namespace, Name: input.ConfigCluster.ClusterName}
		err := k8sClient.Get(ctx, key, cluster)
		if err != nil {
			return false
		}
		return conditions.IsTrue(cluster, clusterv1.ReadyCondition)
	}, 30*time.Minute, 30*time.Second).Should(BeTrue(), "Cluster did not become ready in time")
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

	It("Should successfully create an infrastructure cluster", func() {
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
				ControlPlaneMachineCount: ptr.To(int64(3)),
				WorkerMachineCount:       ptr.To(int64(1)),
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
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
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
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
	})
	It("Should successfully create a cluster with one control planes", func() {
		By("Creating a workload cluster")
		ctx := context.Background()

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
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
				ControlPlaneMachineCount: ptr.To(int64(1)),
				WorkerMachineCount:       ptr.To(int64(1)),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		By("Check workload cluster services")
		clusterNamespace := namespace.Name
		clusterName := clusterResources.Cluster.Name
		workloadProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, clusterNamespace, clusterName)
		workloadClient := workloadProxy.GetClient()

		utils.WaitForDaemonSetAvailable(ctx, utils.DaemonSetInput{
			Getter:    workloadClient,
			Name:      "kube-proxy",
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

		coreDns, err := labels.Parse("k8s-app=kube-dns")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: coreDns},
		})
		etcd, err := labels.Parse("component=etcd")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: etcd},
		})

		kube_apiserver, err := labels.Parse("component=kube-apiserver")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: kube_apiserver},
		})
		kube_controller_manager, err := labels.Parse("component=kube-controller-manager")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: kube_controller_manager},
		})
		kube_proxy, err := labels.Parse("k8s-app=kube-proxy")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: kube_proxy},
		})
		kube_scheduler, err := labels.Parse("component=kube-scheduler")
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForPodToBeReady(ctx, utils.PodListInput{
			Lister:      workloadClient,
			ListOptions: &client.ListOptions{LabelSelector: kube_scheduler},
		})
		By("Config PASSED!")
	})
	AfterEach(func() {
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
	})
}
