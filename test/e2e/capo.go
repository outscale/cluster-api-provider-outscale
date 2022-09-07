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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	gomega "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/test/e2e/framework/log"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// CapoClusterDeploymentSpecInput is used with method CapoClusterDeploymentSpec.
type CapoClusterDeploymentSpecInput struct {
	E2EConfig             *clusterctl.E2EConfig
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool
	Flavor                string
}

// CreateClusterAndWaitInput is use with method CreateClusterAndWait.
type CreateClusterAndWaitInput struct {
	ClusterProxy            framework.ClusterProxy
	ConfigCluster           clusterctl.ConfigClusterInput
	WaitForClusterIntervals []interface{}
	Args                    []string
}

// CreateClusterAndWaitResult is composed of cluster..
type CreateClusterAndWaitResult struct {
	Cluster *clusterv1.Cluster
}

// CreateClusterAndWait create cluster infrastructure and wait to be provisionned.
func CreateClusterAndWait(ctx context.Context, input CreateClusterAndWaitInput, result *CreateClusterAndWaitResult) {
	gomega.Expect(ctx).NotTo(gomega.BeNil(), "ctx is required for CreateClusterAndWait")
	gomega.Expect(input.ClusterProxy).ToNot(gomega.BeNil(), "Invalid argument. input.ClusterProxy can't be nil when calling CreateClusterAndWait")
	gomega.Expect(result).ToNot(gomega.BeNil(), "Invalid argument. result can't be nil when calling CreateClusterTemplateAndWait")
	gomega.Expect(input.ConfigCluster.ControlPlaneMachineCount).ToNot(gomega.BeNil())
	gomega.Expect(input.ConfigCluster.WorkerMachineCount).ToNot(gomega.BeNil())

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
	gomega.Expect(workloadClusterTemplate).ToNot(gomega.BeNil(), "Failed to get the cluster template")
	log.Logf("Applying the cluster template yaml to the cluster")
	gomega.Eventually(func() error {
		return input.ClusterProxy.Apply(ctx, workloadClusterTemplate, input.Args...)
	}, 10*time.Second).Should(gomega.Succeed(), "Failed to apply the cluster template")

	log.Logf("Waiting for the cluster infrastructure to be provisioned")
	result.Cluster = framework.DiscoveryAndWaitForCluster(ctx, framework.DiscoveryAndWaitForClusterInput{
		Getter:    input.ClusterProxy.GetClient(),
		Namespace: input.ConfigCluster.Namespace,
		Name:      input.ConfigCluster.ClusterName,
	}, input.WaitForClusterIntervals...)
}

// CapoClusterDeploymentSpec create infrastructure cluster using its generated config and wait infrastructure cluster to be provisionned.
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
		gomega.Expect(ctx).NotTo(gomega.BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		gomega.Expect(input.E2EConfig).ToNot(gomega.BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		gomega.Expect(input.ClusterctlConfigPath).To(gomega.BeAnExistingFile(), "Invalid argument. input.ClusterConfigPath must be an existing file when calling %s spec", specName)
		gomega.Expect(input.BootstrapClusterProxy).ToNot(gomega.BeNil(), "Invalid argument. input.BoostrapClusterProxy can't be nil when calling %s spec", specName)
		gomega.Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(gomega.Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		gomega.Expect(input.E2EConfig.Variables).To(gomega.HaveKey(KubernetesVersion))
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
