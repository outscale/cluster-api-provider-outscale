package utils

import (
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

type DeploymentInput struct {
	Getter    client.Client
	Name      string
	Namespace string
}

type CreateDeploymentInput struct {
	Getter    client.Client
	Name      string
	Namespace string
	Image     string
	ConfigMapName string
	ConfigMapKey string
	Port      int32
}

func CreateDeployment(ctx context.Context, input CreateDeploymentInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in CreateDeployment")
	Expect(input.Name).ToNot(BeNil(), "Need a name in CreateDeployment")
	createDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
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
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": input.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: input.Name,
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
									Name:          "tcp",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: input.Port,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: input.ConfigMapKey,
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: input.ConfigMapName,
											},
											Key: input.ConfigMapKey,
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
	if err := input.Getter.Create(ctx, createDeployment); err != nil {
		By(fmt.Sprintf("Can not create deployment %s", err))
		return false
	}
	By(fmt.Sprintf("Create Deployment %s", input.Name))
	return true
}

func GetDeployment(ctx context.Context, input DeploymentInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in GetDeployment")
	Expect(input.Name).ToNot(BeNil(), "Need a name in GetDeployment")
	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, deployment); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	By(fmt.Sprintf("Find Deployment %s", input.Name))
	return true
}

func DeleteDeployment(ctx context.Context, input DeploymentInput) bool {
	Expect(input.Namespace).ToNot(BeNil(), "Need a namespace in deleteDeployment")
	Expect(input.Name).ToNot(BeNil(), "Need a name in DeleteDeployment")
	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{
		Namespace: input.Namespace,
		Name:      input.Name,
	}
	if err := input.Getter.Get(ctx, key, deployment); err != nil {
		By(fmt.Sprintf("Can not find %s", err))
		return false
	}
	if err := input.Getter.Delete(ctx, deployment); err != nil {
		By(fmt.Sprintf("Can not delete deployment %s", err))
		return false
	}
	By(fmt.Sprintf("Delete Deployment %s", input.Name))
	return true
}

func WaitForDeploymentAvailable(ctx context.Context, input DeploymentInput) {
	By(fmt.Sprintf("Waiting for deployment %s to be available", input.Name))
	Eventually(func() bool {
		isDeploymentAvailable := GetDeployment(ctx, input)
		return isDeploymentAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find deployment %s", input.Name)
}

func WaitForCreateDeploymentAvailable(ctx context.Context, input CreateDeploymentInput) {
	By(fmt.Sprintf("Wait for deployment %s to be created and be available", input.Name))
	Eventually(func() bool {
		isCreateDeploymentAvailable := CreateDeployment(ctx, input)
		return isCreateDeploymentAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to create deployment %s", input.Name)
}

func WaitForDeleteDeploymentAvailable(ctx context.Context, input DeploymentInput) {
	By(fmt.Sprintf("Wait for deployment %s to be deleted", input.Name))
	Eventually(func() bool {
		isDeleteDeploymentAvailable := DeleteDeployment(ctx, input)
		return isDeleteDeploymentAvailable
	}, 2*time.Minute, 10*time.Second).Should(BeTrue(), "Failed to delete deployment %s", input.Name)
}
