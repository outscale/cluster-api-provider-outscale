package utils

import (
	"fmt"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type DaemonSetInput struct {
	Getter    client.Client
	Name      string
	Namespace string
}

type CreateDaemonSetInput struct {
	Getter    client.Client
	Name      string
	Namespace string
	Image     string
	SecretName string
	SecretKey string
	Port      int32
}

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
	By(fmt.Sprintf("Find DaemonSet %s", input.Name))
	return true
}

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
	By(fmt.Sprintf("Delete DaemonSet %s", input.Name))
	return true
}

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
	By(fmt.Sprintf("Create DaemonSet %s", input.Name))
	return true
}

func WaitForDaemonSetAvailable(ctx context.Context, input DaemonSetInput) {
	By(fmt.Sprintf("Waiting for daemonset %s to be available", input.Name))
	Eventually(func() bool {
		isDaemonSetAvailable := GetDaemonSet(ctx, input)
		return isDaemonSetAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find deployment %s", input.Name)
}

func WaitForCreateDaemonSetAvailable(ctx context.Context, input CreateDaemonSetInput) {
	By(fmt.Sprintf("Wait for daemonSet %s to be created and be available", input.Name))
	Eventually(func() bool {
		isCreateDaemonSetAvailable := CreateDaemonSet(ctx, input)
		return isCreateDaemonSetAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create daemonSet %s", input.Name)
}

func WaitForDeleteDaemonSetAvailable(ctx context.Context, input DaemonSetInput) {
	By(fmt.Sprintf("Wait for daemonset M%s to be deleted", input.Name))
	Eventually(func() bool {
		isDeleteDaemonSetAvailable := DeleteDaemonSet(ctx, input)
		return isDeleteDaemonSetAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete daemonset %s", input.Name)
}
