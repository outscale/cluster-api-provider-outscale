package utils

import (
	"fmt"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ServiceInput struct {
	Getter          client.Client
	Name, Namespace string
}

type CreateServiceInput struct {
	Getter          client.Client
	Name, Namespace string
	Port            int32
	TargetPort      int
}

// GetService retrieve service
func GetService(ctx context.Context, input ServiceInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in getService")
	Expect(input.Name).ToNot(BeNil(), "Need a name in getService")
	service := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, service); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find Service %s", input.Name))
	return true
}

// DeleteService delete service
func DeleteService(ctx context.Context, input ServiceInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in deleteService")
	Expect(input.Name).ToNot(BeNil(), "Need a name in deleteService")
	service := &corev1.Service{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, service); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	if err := input.Getter.Delete(ctx, service); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Delete Service %s", input.Name))
	return true
}

// CreateService create service
func CreateService(ctx context.Context, input CreateServiceInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in CreateService")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreateService")
	createService := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
			Labels: map[string]string{
				"app": input.Name,
			},
			Annotations: map[string]string{
				"app": input.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": input.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "tcp",
					Protocol:   corev1.ProtocolTCP,
					Port:       input.Port,
					TargetPort: intstr.FromInt(input.TargetPort),
				},
			},
		},
	}
	if err := input.Getter.Create(ctx, createService); err != nil {
		By(fmt.Sprintf("Can not create service %s", input.Name))
		return false
	}
	By(fmt.Sprintf("Create Service %s", input.Name))
	return true
}

// WaitForServiceAvailable wait for service to be available
func WaitForServiceAvailable(ctx context.Context, input ServiceInput) {
	By(fmt.Sprintf("Waiting for service %s to be available", input.Name))
	Eventually(func() bool {
		isServiceAvailable := GetService(ctx, input)
		return isServiceAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find service %s", input.Name)
}

// WaitForCreateServiceAvailable wait for service to be created
func WaitForCreateServiceAvailable(ctx context.Context, input CreateServiceInput) {
	By(fmt.Sprintf("Wait for secret %s to be created and available", input.Name))
	Eventually(func() bool {
		isCreateServiceAvailable := CreateService(ctx, input)
		return isCreateServiceAvailable
	}, 2*time.Minute, 3*time.Second).Should(BeTrue(), "Failed to create service %s", input.Name)
}

// WaitForDeleteServiceAvailable wait for service to be deleted
func WaitForDeleteServiceAvailable(ctx context.Context, input ServiceInput) {
	By(fmt.Sprintf("Wait for service %s to be deleted", input.Name))
	Eventually(func() bool {
		isDeleteServiceAvailable := DeleteService(ctx, input)
		return isDeleteServiceAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete service %s", input.Name)
}
