package test

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"golang.org/x/net/context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type OscInfraMachineInput struct {
	Getter          client.Client
	Name, Namespace string
}

type OscInfraMachineListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type OscInfraMachineListDeleteInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetOscInfraMachine get oscMachine.
func GetOscInfraMachine(ctx context.Context, input OscInfraMachineInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetOscInfraMachine")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetOscInfraMachine")
	oscInfraMachine := &infrastructurev1beta1.OscMachine{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, oscInfraMachine); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find oscInfraMachine %s", input.Name))
	return true
}

// GetOscInfraMachineList get oscMachine.
func GetOscInfraMachineList(ctx context.Context, input OscInfraMachineListInput) bool {
	oscInfraMachineList := &infrastructurev1beta1.OscMachineList{}
	if err := input.Lister.List(ctx, oscInfraMachineList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list OscInfraMachineList %s\n", err))
		return false
	}
	for _, oscInfraMachine := range oscInfraMachineList.Items {
		By(fmt.Sprintf("Find oscInfraMachine %s in namespace %s \n", oscInfraMachine.Name, oscInfraMachine.Namespace))
	}
	return true
}

// DeleteOscInfraMachineList delete oscMachine.
func DeleteOscInfraMachineList(ctx context.Context, input OscInfraMachineListDeleteInput) bool {
	oscInfraMachineList := &infrastructurev1beta1.OscMachineList{}
	if err := input.Deleter.List(ctx, oscInfraMachineList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list infraMachine %s", err))
		return false
	}
	var key client.ObjectKey
	var oscInfraMachineGet *infrastructurev1beta1.OscMachine
	for _, oscInfraMachine := range oscInfraMachineList.Items {
		By(fmt.Sprintf("Find oscInfraMachine %s in namespace %s to be deleted \n", oscInfraMachine.Name, oscInfraMachine.Namespace))
		oscInfraMachineGet = &infrastructurev1beta1.OscMachine{}
		key = client.ObjectKey{
			Namespace: oscInfraMachine.Namespace,
			Name:      oscInfraMachine.Name,
		}
		if err := input.Deleter.Get(ctx, key, oscInfraMachineGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		Eventually(func() error {
			return input.Deleter.Delete(ctx, oscInfraMachineGet)
		}, 30*time.Second, 10*time.Second).Should(Succeed())
		if !oscInfraMachine.Status.Ready {
			fmt.Fprintf(GinkgoWriter, "Delete OscMachine pending \n")
			time.Sleep(15 * time.Second)
			oscInfraMachineGet.ObjectMeta.Finalizers = nil
			Expect(input.Deleter.Update(ctx, oscInfraMachineGet)).Should(Succeed())
			fmt.Fprintf(GinkgoWriter, "Patch machine \n")
		}
		oscInfraMachineGet = &infrastructurev1beta1.OscMachine{}
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait OscInfraMachine %s in namespace %s to be deleted \n", oscInfraMachine.Name, oscInfraMachine.Namespace)
			return input.Deleter.Get(ctx, key, oscInfraMachineGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())
	}
	return true
}

// WaitForOscInfraMachineAvailable wait for oscMachine to be available.
func WaitForOscInfraMachineAvailable(ctx context.Context, input OscInfraMachineInput) bool {
	By(fmt.Sprintf("Wait for oscInfraMachine %s to be available", input.Name))
	Eventually(func() bool {
		isOscMachineAvailable := GetOscInfraMachine(ctx, input)
		return isOscMachineAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find oscInfraMachine %s", input.Name)
	return false
}

// WaitForOscInfraMachineListAvailable wait for oscmachne to be available.
func WaitForOscInfraMachineListAvailable(ctx context.Context, input OscInfraMachineListInput) bool {
	By(fmt.Sprintf("Waiting for OscInfraMachine selected options to be ready"))
	Eventually(func() bool {
		isOscInfraMachineListAvailable := GetOscInfraMachineList(ctx, input)
		return isOscInfraMachineListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find oscInfraMachineList")
	return false
}

// WaitForOscInfraMachineListDelete wait for oscMachine to be deleted.
func WaitForOscInfraMachineListDelete(ctx context.Context, input OscInfraMachineListDeleteInput) bool {
	By(fmt.Sprintf("Waiting for OscInfraMachine selected by options to be deleted"))
	Eventually(func() bool {
		isOscInfraMachineListDelete := DeleteOscInfraMachineList(ctx, input)
		return isOscInfraMachineListDelete
	}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to find oscInfraMachineList")
	return false
}
