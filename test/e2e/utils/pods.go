package utils

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
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
	By(fmt.Sprintf("Waiting for pod selected by options to be ready"))
	Eventually(func() bool {
		isPodAvailableAndReady := IsPodReady(ctx, input)
		return isPodAvailableAndReady
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to find selected pod")
}
