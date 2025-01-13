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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CapoMachineListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type CapoMachineListDeleteInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

type CapoMachineInput struct {
	Getter          client.Client
	Name, Namespace string
}

func GetCapoMachine(ctx context.Context, input CapoMachineInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetCapoMachine")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetCapoMachine")
	capoMachine := &clusterv1.Machine{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, capoMachine); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By("Find capoMachine " + input.Name)
	return true
}

// GetCapoMachineList get machine.
func GetCapoMachineList(ctx context.Context, input CapoMachineListInput) bool {
	capoMachineList := &clusterv1.MachineList{}
	if err := input.Lister.List(ctx, capoMachineList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list CapoMachine %s", err))
		return false
	}
	for _, capoMachine := range capoMachineList.Items {
		By(fmt.Sprintf("Find capoMachine %s in namespace %s \n", capoMachine.Name, capoMachine.Namespace))
	}
	return true
}

// DeleteCapoMachineList delete machine.
func DeleteCapoMachineList(ctx context.Context, input CapoMachineListDeleteInput) bool {
	capoMachineList := &clusterv1.MachineList{}
	if err := input.Deleter.List(ctx, capoMachineList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list CapoMachine %s", err))
		return false
	}
	var key client.ObjectKey
	var capoMachineGet *clusterv1.Machine
	for _, capoMachine := range capoMachineList.Items {
		By(fmt.Sprintf("Find capoMachine %s in namespace %s to be deleted \n", capoMachine.Name, capoMachine.Namespace))
		capoMachineGet = &clusterv1.Machine{}
		key = client.ObjectKey{
			Namespace: capoMachine.Namespace,
			Name:      capoMachine.Name,
		}
		if err := input.Deleter.Get(ctx, key, capoMachineGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		Eventually(func() error {
			return input.Deleter.Delete(ctx, capoMachineGet)
		}, 10*time.Minute, 3*time.Minute).Should(Succeed())
		fmt.Fprintf(GinkgoWriter, "Delete capoMachine pending \n")
		time.Sleep(15 * time.Second)
		if err := input.Deleter.Get(ctx, key, capoMachineGet); err != nil {
			By(fmt.Sprintf("Can not find %s, continue \n", err))
		} else {
			capoMachineGet.ObjectMeta.Finalizers = nil
			Expect(input.Deleter.Update(ctx, capoMachineGet)).Should(Succeed())
			fmt.Fprintf(GinkgoWriter, "Patch machine \n")
		}
		capoMachineGet = &clusterv1.Machine{}
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait capoMachine %s in namespace %s to be deleted \n", capoMachine.Name, capoMachine.Namespace)
			return input.Deleter.Get(ctx, key, capoMachineGet)
		}, 2*time.Minute, 5*time.Second).ShouldNot(Succeed())
	}

	return true
}

// WaitForCapoMachineAvailable wait machine to bee available.
func WaitForCapoMachineAvailable(ctx context.Context, input CapoMachineInput) bool {
	By(fmt.Sprintf("Wait for capoMachine %s to be available", input.Name))
	Eventually(func() bool {
		isCapoAvailable := GetCapoMachine(ctx, input)
		return isCapoAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoMachine %s", input.Name)
	return false
}

// WaitForCapoMachineListAvailable wait machine to be available.
func WaitForCapoMachineListAvailable(ctx context.Context, input CapoMachineListInput) bool {
	By("Waiting for capoMachine selected by options to be ready")
	Eventually(func() bool {
		isCapoMachineListAvailable := GetCapoMachineList(ctx, input)
		return isCapoMachineListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoMachineList")
	return false
}

// WaitForCapoMachineListDelete wait machine to be deleted.
func WaitForCapoMachineListDelete(ctx context.Context, input CapoMachineListDeleteInput) bool {
	By("Waiting for capoMachine selected by options to be deleted")
	Eventually(func() bool {
		isCapoMachineListDelete := DeleteCapoMachineList(ctx, input)
		return isCapoMachineListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find capoMachineList")
	return false
}
