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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type OscInfraMachineTemplateInput struct {
	Getter          client.Client
	Name, Namespace string
}

type OscInfraMachineTemplateListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type OscInfraMachineTemplateListDeleteInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetOscInfraMachineTemplate get oscMachineTemplate.
func GetOscInfraMachineTemplate(ctx context.Context, input OscInfraMachineTemplateInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetOscInfraMachineTemplate")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetOscInfraMachineTemplate")
	oscInfraMachineTemplate := &infrastructurev1beta1.OscMachineTemplate{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, oscInfraMachineTemplate); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find oscInfraMachineTemplate %s", input.Name))
	return true
}

// GetOscInfraMachineTemplateList get oscMachineTemplate
func GetOscInfraMachineTemplateList(ctx context.Context, input OscInfraMachineTemplateListInput) bool {
	oscInfraMachineTemplateList := &infrastructurev1beta1.OscMachineTemplateList{}
	if err := input.Lister.List(ctx, oscInfraMachineTemplateList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list OscInfraMachineTemplateList %s\n", err))
		return false
	}
	for _, oscInfraMachineTemplate := range oscInfraMachineTemplateList.Items {
		By(fmt.Sprintf("Find oscInfraMachineTemplate %s in namespace %s \n", oscInfraMachineTemplate.Name, oscInfraMachineTemplate.Namespace))
	}
	return true
}

// DeleteOscInfraMachineTemplateList delete oscMachineTemplate
func DeleteOscInfraMachineTemplateList(ctx context.Context, input OscInfraMachineTemplateListDeleteInput) bool {
	oscInfraMachineTemplateList := &infrastructurev1beta1.OscMachineTemplateList{}
	if err := input.Deleter.List(ctx, oscInfraMachineTemplateList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list infraMachine %s", err))
		return false
	}
	var key client.ObjectKey
	var oscInfraMachineTemplateGet *infrastructurev1beta1.OscMachineTemplate
	for _, oscInfraMachineTemplate := range oscInfraMachineTemplateList.Items {
		By(fmt.Sprintf("Find oscInfraMachineTemplate %s in namespaace %s to be deleted \n", oscInfraMachineTemplate.Name, oscInfraMachineTemplate.Namespace))
		oscInfraMachineTemplateGet = &infrastructurev1beta1.OscMachineTemplate{}
		key = client.ObjectKey{
			Namespace: oscInfraMachineTemplate.Namespace,
			Name:      oscInfraMachineTemplate.Name,
		}
		if err := input.Deleter.Get(ctx, key, oscInfraMachineTemplateGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		Eventually(func() error {
			return input.Deleter.Delete(ctx, oscInfraMachineTemplateGet)
		}, 60*time.Second, 5*time.Second).Should(Succeed())

		fmt.Fprintf(GinkgoWriter, "Delete OscMachineTemplate pending \n")
		time.Sleep(10 * time.Second)
		if err := input.Deleter.Get(ctx, key, oscInfraMachineTemplateGet); err != nil {
			By(fmt.Sprintf("Can not find %s, continue \n", err))
		} else {
			oscInfraMachineTemplateGet.ObjectMeta.Finalizers = nil
			Expect(input.Deleter.Update(ctx, oscInfraMachineTemplateGet)).Should(Succeed())
			fmt.Fprintf(GinkgoWriter, "Patch machineTemplate \n")
		}
		oscInfraMachineTemplateGet = &infrastructurev1beta1.OscMachineTemplate{}
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait OscInfraMachineTemplate %s in namespace %s to be deleted \n", oscInfraMachineTemplate.Name, oscInfraMachineTemplate.Namespace)
			return input.Deleter.Get(ctx, key, oscInfraMachineTemplateGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())
	}
	return true
}

// WaitForOscInfraMachineTemplateAvailable wait for oscMachineTemplate to be available
func WaitForOscInfraMachineTemplateAvailable(ctx context.Context, input OscInfraMachineTemplateInput) bool {
	By(fmt.Sprintf("Wait for OscInfraMachineTemplate %s to be available", input.Name))
	Eventually(func() bool {
		isOscMachineTemplateAvailable := GetOscInfraMachineTemplate(ctx, input)
		return isOscMachineTemplateAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find oscInfraMachineTemplate %s", input.Name)
	return false
}

// WaitForOscInfraMachineTemplateListAvailable wait for oscmachinetemplate to be available
func WaitForOscInfraMachineTemplateListAvailable(ctx context.Context, input OscInfraMachineTemplateListInput) bool {
	By(fmt.Sprintf("Waiting for OscInfraMachineTemplate selected options to be ready"))
	Eventually(func() bool {
		isOscInfraMachineTemplateListAvailable := GetOscInfraMachineTemplateList(ctx, input)
		return isOscInfraMachineTemplateListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find OscInfraMachineTemplateList")
	return false
}

// WaitForOscInfraMachineTemplateListDelete wait for oscMachineTemplate to be deleted.
func WaitForOscInfraMachineTemplateListDelete(ctx context.Context, input OscInfraMachineTemplateListDeleteInput) bool {
	By(fmt.Sprintf("Waiting for OscInfraMachineTemplate selected by options to be deleted"))
	Eventually(func() bool {
		isOscInfraMachineTemplateListDelete := DeleteOscInfraMachineTemplateList(ctx, input)
		return isOscInfraMachineTemplateListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find oscInfraMachineTemplateList")
	return false
}
