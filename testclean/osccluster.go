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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OscInfraClusterInput struct {
	Getter          client.Client
	Name, Namespace string
}

type OscInfraClusterListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type OscInfraClusterDeleteListInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetOscInfraCluster get osccluster.
func GetOscInfraCluster(ctx context.Context, input OscInfraClusterInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetOscInfraCluster")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetOscInfraCluster")
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, oscInfraCluster); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By("Find OscClusterMachine " + input.Name)
	return true
}

// GetOscInfraClusterList get oscCluster.
func GetOscInfraClusterList(ctx context.Context, input OscInfraClusterListInput) bool {
	oscInfraClusterList := &infrastructurev1beta1.OscClusterList{}
	if err := input.Lister.List(ctx, oscInfraClusterList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list OscInfraClusterList %s\n", err))
		return false
	}
	for _, oscInfraCluster := range oscInfraClusterList.Items {
		By(fmt.Sprintf("Find oscInfraCluster %s in namespace %s \n", oscInfraCluster.Name, oscInfraCluster.Namespace))
	}
	return true
}

// DeleteOscInfraClusterList delete oscCluster.
func DeleteOscInfraClusterList(ctx context.Context, input OscInfraClusterDeleteListInput) bool {
	oscInfraClusterList := &infrastructurev1beta1.OscClusterList{}
	if err := input.Deleter.List(ctx, oscInfraClusterList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list oscInfraClusterList %s", err))
		return false
	}
	var key client.ObjectKey
	var oscInfraClusterGet *infrastructurev1beta1.OscCluster
	for _, oscInfraCluster := range oscInfraClusterList.Items {
		By(fmt.Sprintf("Find oscInfraCluster %s in namespace to be deleted %s\n", oscInfraCluster.Name, oscInfraCluster.Namespace))
		oscInfraClusterGet = &infrastructurev1beta1.OscCluster{}
		key = client.ObjectKey{
			Namespace: oscInfraCluster.Namespace,
			Name:      oscInfraCluster.Name,
		}
		if err := input.Deleter.Get(ctx, key, oscInfraClusterGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		Eventually(func() error {
			return input.Deleter.Delete(ctx, oscInfraClusterGet)
		}, 20*time.Minute, 8*time.Minute).Should(Succeed())
		time.Sleep(2 * time.Minute)
		if err := input.Deleter.Get(ctx, key, oscInfraClusterGet); err != nil {
			By(fmt.Sprintf("Can not find %s, continue \n", err))
		} else {
			time.Sleep(30 * time.Second)
			oscInfraClusterGet.ObjectMeta.Finalizers = nil
			Expect(input.Deleter.Update(ctx, oscInfraClusterGet)).Should(Succeed())
			fmt.Fprintf(GinkgoWriter, "Patch oscCluster \n")
		}
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait oscInfraCluster %s in namspace %s to be deleted  \n", oscInfraCluster.Name, oscInfraCluster.Namespace)
			return input.Deleter.Get(ctx, key, oscInfraClusterGet)
		}, 5*time.Minute, 10*time.Second).ShouldNot(Succeed())
	}
	return true
}

// WaitForOscInfraClusterAvailable wait osccluster to be available.
func WaitForOscInfraClusterAvailable(ctx context.Context, input OscInfraClusterInput) bool {
	By(fmt.Sprintf("Wait for oscInfraCluster %s to be available", input.Name))
	Eventually(func() bool {
		isOscClusterAvailable := GetOscInfraCluster(ctx, input)
		return isOscClusterAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find oscInfraCluster %s", input.Name)
	return false
}

// WaitForOscInfraClusterListAvailable wait oscCluster to be available.
func WaitForOscInfraClusterListAvailable(ctx context.Context, input OscInfraClusterListInput) bool {
	By("Waiting for oscInfraCluster selected options to be ready")
	Eventually(func() bool {
		isOscInfraClusterListAvailable := GetOscInfraClusterList(ctx, input)
		return isOscInfraClusterListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoClusterList")
	return false
}

// WaitForOscInfraClusterListDelete wait oscclustet to be deleted.
func WaitForOscInfraClusterListDelete(ctx context.Context, input OscInfraClusterDeleteListInput) bool {
	By("Wait for oscInfraCluster selected by options to be deleted")
	Eventually(func() bool {
		isOscInfraClusterListDelete := DeleteOscInfraClusterList(ctx, input)
		return isOscInfraClusterListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find oscInfraClusterListDelete")
	return false
}
