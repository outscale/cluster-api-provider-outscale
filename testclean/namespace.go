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
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type NamespaceListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type NamespaceListDeleteInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

func GetNamespaceList(ctx context.Context, input NamespaceListInput) bool {
	NamespaceList := &v1.NamespaceList{}
	if err := input.Lister.List(ctx, NamespaceList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list NamespaceList %s", err))
		return false
	}
	for _, namespace := range NamespaceList.Items {
		By(fmt.Sprintf("Find namespaceList %s \n", namespace.Name))
	}
	return true
}

func DeleteNamespaceList(ctx context.Context, input NamespaceListDeleteInput) bool {
	NamespaceList := &v1.NamespaceList{}
	if err := input.Deleter.List(ctx, NamespaceList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list namespaceList %s\n", err))
		return false
	}
	var key client.ObjectKey
	var NamespaceGet *v1.Namespace
	for _, namespace := range NamespaceList.Items {
		By(fmt.Sprintf("Find Namespace %sto be delete", namespace.Name))
		NamespaceGet = &v1.Namespace{}
		key = client.ObjectKey{
			Name:      namespace.Name,
			Namespace: namespace.Name,
		}
		if err := input.Deleter.Get(ctx, key, NamespaceGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		time.Sleep(10 * time.Second)
		Eventually(func() error {
			return input.Deleter.Delete(ctx, NamespaceGet)
		}, 30*time.Second, 10*time.Second).Should(Succeed())
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait Namespace %s to be deleted \n", namespace.Name)
			return input.Deleter.Get(ctx, key, NamespaceGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())
	}
	return true
}

func WaitForNamespaceListAvailable(ctx context.Context, input NamespaceListInput) bool {
	By(fmt.Sprintf("Waiting for namespace selected by options to be ready"))
	Eventually(func() bool {
		isNamespaceAvailable := GetNamespaceList(ctx, input)
		return isNamespaceAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find NamespaceList")
	return false
}

func WaitForNamespaceListDelete(ctx context.Context, input NamespaceListDeleteInput) bool {
	By(fmt.Sprintf("Wait for namespace selected by options too be ready"))
	Eventually(func() bool {
		isNamespaceListDelete := DeleteNamespaceList(ctx, input)
		return isNamespaceListDelete
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find namespaceList")
	return false
}
