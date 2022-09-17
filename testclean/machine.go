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
	By(fmt.Sprintf("Find capoMachine %s", input.Name))
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
		}, 2*time.Minute, 10*time.Second).Should(Succeed())
		fmt.Fprintf(GinkgoWriter, "Delete capoMachine pending \n")
		time.Sleep(15 * time.Second)
		capoMachineGet.ObjectMeta.Finalizers = nil
		Expect(input.Deleter.Update(ctx, capoMachineGet)).Should(Succeed())
		fmt.Fprintf(GinkgoWriter, "Patch machine \n")
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
	By(fmt.Sprintf("Waiting for capoMachine selected by options to be ready"))
	Eventually(func() bool {
		isCapoMachineListAvailable := GetCapoMachineList(ctx, input)
		return isCapoMachineListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoMachineList")
	return false
}

// WaitForCapoMachineListDelete wait machine to be deleted.
func WaitForCapoMachineListDelete(ctx context.Context, input CapoMachineListDeleteInput) bool {
	By(fmt.Sprintf("Waiting for capoMachine selected by options to be deleted"))
	Eventually(func() bool {
		isCapoMachineListDelete := DeleteCapoMachineList(ctx, input)
		return isCapoMachineListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find capoMachineList")
	return false
}
