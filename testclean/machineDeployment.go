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

type CapoMachineDeploymentListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type CapoMachineDeploymentDeleteListInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetCapoMachineDeploymentList get machineList.
func GetCapoMachineDeploymentList(ctx context.Context, input CapoMachineDeploymentListInput) bool {
	capoMachineDeploymentList := &clusterv1.MachineDeploymentList{}
	if err := input.Lister.List(ctx, capoMachineDeploymentList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list CapoMachineDeployment %s", err))
		return false
	}
	for _, capoMachineDeployment := range capoMachineDeploymentList.Items {
		By(fmt.Sprintf("Find capoMachineList %s in namespace %s \n", capoMachineDeployment.Name, capoMachineDeployment.Namespace))
	}
	return true
}

// DeleteCapoMachineDeploymentList delete machine.
func DeleteCapoMachineDeploymentList(ctx context.Context, input CapoMachineDeploymentDeleteListInput) bool {
	capoMachineDeploymentList := &clusterv1.MachineDeploymentList{}
	if err := input.Deleter.List(ctx, capoMachineDeploymentList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list capoMachineDeployment %s", err))
		return false
	}
	var key client.ObjectKey
	var capoMachineDeploymentGet *clusterv1.MachineDeployment
	for _, capoMachineDeployment := range capoMachineDeploymentList.Items {
		By(fmt.Sprintf("Find capoMachineDeployment %s in namespace %s to be deleted \n", capoMachineDeployment.Name, capoMachineDeployment.Namespace))
		capoMachineDeploymentGet = &clusterv1.MachineDeployment{}
		key = client.ObjectKey{
			Namespace: capoMachineDeployment.Namespace,
			Name:      capoMachineDeployment.Name,
		}
		if err := input.Deleter.Get(ctx, key, capoMachineDeploymentGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		time.Sleep(10 * time.Second)
		Eventually(func() error {
			return input.Deleter.Delete(ctx, capoMachineDeploymentGet)
		}, 30*time.Second, 10*time.Second).Should(Succeed())
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait capoMachineDeployment %s in namespace %s to be deleted \n", capoMachineDeployment.Name, capoMachineDeployment.Namespace)
			return input.Deleter.Get(ctx, key, capoMachineDeploymentGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())

	}
	return true
}

// WaitForCapoMachineDeploymentListAvailable wait machine to be available.
func WaitForCapoMachineDeploymentListAvailable(ctx context.Context, input CapoMachineDeploymentListInput) bool {
	By(fmt.Sprintf("Waiting for capoMachineDeployment selected by options to be ready"))
	Eventually(func() bool {
		isCapoMachineDeploymentListAvailable := GetCapoMachineDeploymentList(ctx, input)
		return isCapoMachineDeploymentListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoMachineDeploymentList")
	return false
}

// WaitForCapoMachineDeploymentListDelete  wait machine to be deleted.
func WaitForCapoMachineDeploymentListDelete(ctx context.Context, input CapoMachineDeploymentDeleteListInput) bool {
	By(fmt.Sprintf("Wait for capoMachineDeployment selected by options to be deleted"))
	Eventually(func() bool {
		isCapoMachineDeploymentListDelete := DeleteCapoMachineDeploymentList(ctx, input)
		return isCapoMachineDeploymentListDelete
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoMachineDeploymentList")
	return false
}
