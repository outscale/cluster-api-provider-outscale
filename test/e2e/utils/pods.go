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

package utils

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podutil "k8s.io/kubectl/pkg/util/podutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

// IsPodReady check if pod is ready
func IsPodReady(ctx context.Context, input PodListInput) bool {
	podList := &corev1.PodList{}
	if err := input.Lister.List(ctx, podList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list pod %s", err))
		return false
	}
	for _, pod := range podList.Items {
		By(fmt.Sprintf("Find pod %s in namespace %s \n", pod.Name, pod.Namespace))
		now := metav1.Now()
		isAvailable := podutil.IsPodAvailable(&pod, 0, now)
		By(fmt.Sprintf("Find Pod Ready %t\n", isAvailable))
		if !isAvailable {
			By(fmt.Sprintf("Pod %s in namespace %s is not ready\n", pod.Name, pod.Namespace))
			return false
		}
	}
	return true
}

// WaitForPodToBeReady wait for pod to be ready
func WaitForPodToBeReady(ctx context.Context, input PodListInput) {
	By("Waiting for pod selected by options to be ready")
	Eventually(func() bool {
		isPodAvailableAndReady := IsPodReady(ctx, input)
		return isPodAvailableAndReady
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to find selected pod")
}
