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
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type SecretInput struct {
	Getter    client.Client
	Name      string
	Namespace string
}

type CreateSecretInput struct {
	Getter                              client.Client
	Name, Namespace, DataKey, DataValue string
}

type CreateMultiSecretInput struct {
	Getter                                                                                                      client.Client
	Name, Namespace, DataFirstKey, DataFirstValue, DataSecondKey, DataSecondValue, DataThirdKey, DataThirdValue string
}

// GetSecret retrieve secret
func GetSecret(ctx context.Context, input SecretInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetSecret")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetSecret")
	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}

	if err := input.Getter.Get(ctx, key, secret); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find Secret %s", input.Name))
	return true
}

// DeleteSecret delete secret
func DeleteSecret(ctx context.Context, input SecretInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in DeleteSecret")
	Expect(input.Name).ToNot(BeNil(), "Need a name in DeleteSecret")
	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, secret); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	if err := input.Getter.Delete(ctx, secret); err != nil {
		By(fmt.Sprintf("Can not delete secret %s", err))
		return false
	}
	By(fmt.Sprintf("Delete Secret %s", input.Name))
	return true
}

// CreateSecret create secret
func CreateSecret(ctx context.Context, input CreateSecretInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in CreateSecret")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreateSecret")
	createSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
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
		Type: "Opaque",
		Data: map[string][]byte{
			input.DataKey: []byte(input.DataValue),
		},
	}
	if err := input.Getter.Create(ctx, createSecret); err != nil {
		By(fmt.Sprintf("Can not create secret %s", err))
		return false
	}
	By(fmt.Sprintf("Create secret %s", input.Name))
	return true
}

// CreateMultiSecret create multi secret
func CreateMultiSecret(ctx context.Context, input CreateMultiSecretInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in CreateSecret")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreateSecret")
	createMultiSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
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
		Type: "Opaque",
		Data: map[string][]byte{
			input.DataFirstKey:  []byte(input.DataFirstValue),
			input.DataSecondKey: []byte(input.DataSecondValue),
			input.DataThirdKey:  []byte(input.DataThirdValue),
		},
	}
	if err := input.Getter.Create(ctx, createMultiSecret); err != nil {
		By(fmt.Sprintf("Can not create multi secret %s", err))
		return false
	}
	By(fmt.Sprintf("Create multi secret %s", input.Name))
	return true
}

// WaitForSecretsAvailable wait for secret to be available
func WaitForSecretsAvailable(ctx context.Context, input SecretInput) {
	By(fmt.Sprintf("Waiting for secret %s to be available", input.Name))
	Eventually(func() bool {
		isSecretAvailable := GetSecret(ctx, input)
		return isSecretAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find secret %s", input.Name)
}

// WaitForCreateSecretAvailable wait for secret to be created
func WaitForCreateSecretAvailable(ctx context.Context, input CreateSecretInput) {
	By(fmt.Sprintf("Wait for secret %s to be created and available", input.Name))
	Eventually(func() bool {
		isCreateSecretAvailable := CreateSecret(ctx, input)
		return isCreateSecretAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create secret %s", input.Name)
}

// WaitForCreateMultiSecretAvailable wait for multi secret to be available
func WaitForCreateMultiSecretAvailable(ctx context.Context, input CreateMultiSecretInput) {
	By(fmt.Sprintf("Wait for secret %s to be created and available", input.Name))
	Eventually(func() bool {
		isCreateMultiSecretAvailable := CreateMultiSecret(ctx, input)
		return isCreateMultiSecretAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create multi secret %s", input.Name)
}

// WaitForDeleteSecretAvailable wait for secret to be deleted
func WaitForDeleteSecretAvailable(ctx context.Context, input SecretInput) {
	By(fmt.Sprintf("Wait for secret %s to be delete", input.Name))
	Eventually(func() bool {
		isDeleteSecretAvailable := DeleteSecret(ctx, input)
		return isDeleteSecretAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete secret %s", input.Name)
}
