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
	"golang.org/x/net/context"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type CapoClusterInput struct {
	Getter          client.Client
	Name, Namespace string
}
type CapoClusterGetNamespaceInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type CapoClusterInputDeleteListInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetCapoCluster get capo cluster.
func GetCapoCluster(ctx context.Context, input CapoClusterInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetCapoCluster")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetCapoCluster")
	capoCluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, capoCluster); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find capoCluster %s", input.Name))
	return true
}

// DeleteCapoClusterList delete cluster list.
func DeleteCapoClusterList(ctx context.Context, input CapoClusterInputDeleteListInput) bool {
	capoClusterList := &clusterv1.ClusterList{}
	if err := input.Deleter.List(ctx, capoClusterList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list capoClusterList %s\n", err))
		return false
	}
	var key client.ObjectKey
	var capoClusterGet *clusterv1.Cluster
	for _, capoCluster := range capoClusterList.Items {
		By(fmt.Sprintf("Find CapoCluster %s in namespace to be deleted %s\n", capoCluster.Name, capoCluster.Namespace))
		capoClusterGet = &clusterv1.Cluster{}
		key = client.ObjectKey{
			Namespace: capoCluster.Namespace,
			Name:      capoCluster.Name,
		}
		if err := input.Deleter.Get(ctx, key, capoClusterGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		time.Sleep(10 * time.Second)
		Eventually(func() error {
			return input.Deleter.Delete(ctx, capoClusterGet)
		}, 30*time.Second, 10*time.Second).Should(Succeed())
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait capoCluster %s in namespace %s to be deleted \n", capoCluster.Name, capoCluster.Namespace)
			return input.Deleter.Get(ctx, key, capoClusterGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())
	}
	return true
}

// GetCapoClusterNamespace get capo cluster namespace.
func GetCapoClusterNamespace(ctx context.Context, input CapoClusterGetNamespaceInput) (string, bool) {
	capoClusterList := &clusterv1.ClusterList{}
	if err := input.Lister.List(ctx, capoClusterList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list capoClusterList %s\n", err))
		return "", false
	}
	for _, capoCluster := range capoClusterList.Items {
		By(fmt.Sprintf("Find CapoCluster %s in namespace %s\n", capoCluster.Name, capoCluster.Namespace))
		return capoCluster.Namespace, true
	}
	return "", false
}

// WaitForCapoClusterAvailable wait capoCluster to be available.
func WaitForCapoClusterAvailable(ctx context.Context, input CapoClusterInput) bool {
	By(fmt.Sprintf("Wait for capoCluster %s to be available", input.Name))
	Eventually(func() bool {
		isCapoAvailable := GetCapoCluster(ctx, input)
		return isCapoAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoCluster %s", input.Name)
	return false
}

// WaitForCapoClusterListDelete wait capocluster to be deleted.
func WaitForCapoClusterListDelete(ctx context.Context, input CapoClusterInputDeleteListInput) bool {
	By(fmt.Sprintf("Wait for capoCluster selected by options to be ready"))
	Eventually(func() bool {
		isCapoClusterListDelete := DeleteCapoClusterList(ctx, input)
		return isCapoClusterListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find capoClusterListDelete")
	return false
}

// WaitForFindingCapoNamespace find capo Namespace.
func WaitForFindingCapoNamespace(ctx context.Context, input CapoClusterGetNamespaceInput) string {
	By(fmt.Sprintf("Wait for capoCluster selected by options to be deleted"))
	Eventually(func() bool {
		_, isCapoClusterNamespace := GetCapoClusterNamespace(ctx, input)
		return isCapoClusterNamespace
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoClusterNamespace")
	capoNamespace, _ := GetCapoClusterNamespace(ctx, input)
	return capoNamespace
}
