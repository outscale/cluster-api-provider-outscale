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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DaemonSetInput struct {
	Getter    client.Client
	Name      string
	Namespace string
}

type CreateDaemonSetInput struct {
	Getter     client.Client
	Name       string
	Namespace  string
	Image      string
	SecretName string
	SecretKey  string
	Port       int32
}

// GetDaemonSet retrieve daemonset
func GetDaemonSet(ctx context.Context, input DaemonSetInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetDaemonSet")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetDaemonSet")
	daemonSet := &appsv1.DaemonSet{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, daemonSet); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By("Find DaemonSet " + input.Name)
	return true
}

// DeleteDaemonSet delete daemonset
func DeleteDaemonSet(ctx context.Context, input DaemonSetInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in DeleteDaemonSet")
	Expect(input.Name).ToNot(BeNil(), "Need a name in DeleteDaemonSet")
	daemonSet := &appsv1.DaemonSet{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, daemonSet); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	if err := input.Getter.Delete(ctx, daemonSet); err != nil {
		By(fmt.Sprintf("Can not delete daemonSet %s", err))
		return false
	}
	By("Delete DaemonSet " + input.Name)
	return true
}

// CreateDaemonSet create daemonset
func CreateDaemonSet(ctx context.Context, input CreateDaemonSetInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namesppace in CreateDaemonSet")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreateDaemonSet")
	createDaemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
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
		Spec: appsv1.DaemonSetSpec{
			MinReadySeconds: 0,
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.OnDeleteDaemonSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": input.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": input.Name,
					},
					Annotations: map[string]string{
						"app": input.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            input.Name,
							Image:           input.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:     "tcp",
									Protocol: corev1.ProtocolTCP, ContainerPort: input.Port,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: input.SecretKey,
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: input.SecretName,
											},
											Key: input.SecretKey,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := input.Getter.Create(ctx, createDaemonSet); err != nil {
		By(fmt.Sprintf("Can not create daemmonset %s", err))
		return false
	}
	By("Create DaemonSet " + input.Name)
	return true
}

// WaitForDaemonSetAvailable wait for daemonset to be available
func WaitForDaemonSetAvailable(ctx context.Context, input DaemonSetInput) {
	By(fmt.Sprintf("Waiting for daemonset %s to be available", input.Name))
	Eventually(func() bool {
		isDaemonSetAvailable := GetDaemonSet(ctx, input)
		return isDaemonSetAvailable
	}, 10*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to find deployment %s", input.Name)
}

// WaitForCreateDaemonSetAvailable  wait for daemonset to be created
func WaitForCreateDaemonSetAvailable(ctx context.Context, input CreateDaemonSetInput) {
	By(fmt.Sprintf("Wait for daemonSet %s to be created and be available", input.Name))
	Eventually(func() bool {
		isCreateDaemonSetAvailable := CreateDaemonSet(ctx, input)
		return isCreateDaemonSetAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create daemonSet %s", input.Name)
}

// WaitForDeleteDaemonSetAvailable wait for daemonset to be deleted
func WaitForDeleteDaemonSetAvailable(ctx context.Context, input DaemonSetInput) {
	By(fmt.Sprintf("Wait for daemonset M%s to be deleted", input.Name))
	Eventually(func() bool {
		isDeleteDaemonSetAvailable := DeleteDaemonSet(ctx, input)
		return isDeleteDaemonSetAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete daemonset %s", input.Name)
}
