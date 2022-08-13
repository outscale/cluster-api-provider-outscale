package utils

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type ConfigMapInput struct {
	Getter          client.Client
	Name, Namespace string
}

type CreateConfigMapInput struct {
	Getter                              client.Client
	Name, Namespace, DataKey, DataValue string
}

func GetConfigMap(ctx context.Context, input ConfigMapInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetConfigMap")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetConfigMap")
	configmap := &corev1.ConfigMap{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, configmap); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find configmap %s", input.Name))
	return true
}

func DeleteConfigMap(ctx context.Context, input ConfigMapInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in deleteConfigMap")
	Expect(input.Name).ToNot(BeNil(), "Need a name in DeleteConfigMap")
	configmap := &corev1.ConfigMap{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, configmap); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	if err := input.Getter.Delete(ctx, configmap); err != nil {
		By(fmt.Sprintf("Can not delete configmap %s", err))
		return false
	}
	By(fmt.Sprintf("Delete DaemonSet %s", input.Name))
	return true
}

func CreateConfigMap(ctx context.Context, input CreateConfigMapInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in CreateConfigMap")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreaeConfigMap")
	createConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
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
		Data: map[string]string{
			input.DataKey: input.DataValue,
		},
	}
	if err := input.Getter.Create(ctx, createConfigMap); err != nil {
		By(fmt.Sprintf("Can not create configmap %s", err))
		return false
	}
	By(fmt.Sprintf("Create configMap %s", input.Name))
	return true
}

func WaitForConfigMapsAvailable(ctx context.Context, input ConfigMapInput) {
	By(fmt.Sprintf("Waiting for configmap %s to be available", input.Name))
	Eventually(func() bool {
		isConfigMapAvailable := GetConfigMap(ctx, input)
		return isConfigMapAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find configMap %s", input.Name)
}

func WaitForCreateConfigMapAvailable(ctx context.Context, input CreateConfigMapInput) {
	By(fmt.Sprintf("Wait for configmap %s to be created and available", input.Name))
	Eventually(func() bool {
		isCreateConfigMapAvailable := CreateConfigMap(ctx, input)
		return isCreateConfigMapAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create configmap %s", input.Name)
}

func WaitForDeleteConfigMapAvailable(ctx context.Context, input ConfigMapInput) {
	By(fmt.Sprintf("Wait for configMap %s to be deleted", input.Name))
	Eventually(func() bool {
		isDeleteConfigMapAvailable := DeleteConfigMap(ctx, input)
		return isDeleteConfigMapAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete configmap %s", input.Name)
}
