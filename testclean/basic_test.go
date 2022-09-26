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

package test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/slices"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Basic] tools to check", func() {
	BeforeEach(func() {})
	AfterEach(func() {})
	Context("Check content", func() {
		It("should get configmap", func() {
			fmt.Printf("Hello %s", clusterToClean)
			ctx := context.Background()
			cluster_ccm_name, err := labels.Parse("ccm=" + clusterToClean + "-crs-ccm")
			Expect(err).ToNot(HaveOccurred())
			cluster_namespace := WaitForFindingCapoNamespace(ctx, CapoClusterGetNamespaceInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_ccm_name},
			})
			fmt.Printf("cluster_namespace: %s\n", cluster_namespace)
			namespace_name, err := labels.Parse("kubernetes.io/metadata.name=" + cluster_namespace)
			Expect(err).ToNot(HaveOccurred())
			WaitForNamespaceListAvailable(ctx, NamespaceListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: namespace_name},
			})

			cluster_name, err := labels.Parse("cluster.x-k8s.io/cluster-name=" + clusterToClean)
			Expect(err).ToNot(HaveOccurred())

			WaitForCapoMachineListAvailable(ctx, CapoMachineListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForOscInfraClusterListAvailable(ctx, OscInfraClusterListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForOscInfraMachineListAvailable(ctx, OscInfraMachineListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForCapoMachineDeploymentListAvailable(ctx, CapoMachineDeploymentListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})
			WaitForCapoKubeAdmControlPLaneListAvailable(ctx, CapoKubeAdmControlPlaneListInput{
				Lister:      k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForCapoMachineDeploymentListDelete(ctx, CapoMachineDeploymentDeleteListInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})
			WaitForCapoKubeAdmControlPlaneListDelete(ctx, CapoKubeAdmControlPlaneListDeleteInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForCapoMachineListDelete(ctx, CapoMachineListDeleteInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})
			WaitForOscInfraMachineListDelete(ctx, OscInfraMachineListDeleteInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})
			WaitForOscInfraClusterListDelete(ctx, OscInfraClusterDeleteListInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_name},
			})

			WaitForCapoClusterListDelete(ctx, CapoClusterInputDeleteListInput{
				Deleter:     k8sClient,
				ListOptions: &client.ListOptions{LabelSelector: cluster_ccm_name},
			})

			forbidden_namespace := []string{"default", "kube-system", "kube-public"}
			if !slices.Contains(forbidden_namespace, cluster_namespace) {
				WaitForNamespaceListDelete(ctx, NamespaceListDeleteInput{
					Deleter:     k8sClient,
					ListOptions: &client.ListOptions{LabelSelector: namespace_name},
				})
			}
		})
	})
})
