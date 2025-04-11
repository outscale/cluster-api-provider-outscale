package helm_test

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/test/framework"
)

func getHelmSpecs(t *testing.T, vars ...string) []runtime.Object {
	// To avoid importing cert manager, we will not try to test certificate and issuer.
	vars = append(vars, "certificate.enable=false", "issuer.enable=false")
	args := []string{"template", "--debug"}
	if len(vars) > 0 {
		args = append(args, "--set", strings.Join(vars, ","))
	}
	args = append(args, "clusterapioutscale")
	cmd := exec.Command("helm", args...)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	err = cmd.Start()
	require.NoError(t, err)
	scheme := runtime.NewScheme()
	framework.TryAddDefaultSchemes(scheme)
	err = infrastructurev1beta1.AddToScheme(scheme)
	require.NoError(t, err)
	codecs := serializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode

	var specs []runtime.Object
	r := yaml.NewYAMLReader(bufio.NewReader(stdout))
	for {
		buf, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		spec, _, err := decode(buf, nil, nil)
		require.NoError(t, err)
		specs = append(specs, spec)
	}
	err = cmd.Wait()
	require.NoError(t, err)
	return specs
}

func TestHelmTemplate(t *testing.T) {
	t.Run("The chart contains the right objects", func(t *testing.T) {
		specs := getHelmSpecs(t)
		require.Len(t, specs, 17)
		objs := map[string]int{}
		for _, obj := range specs {
			objs[reflect.TypeOf(obj).String()]++
		}
		assert.Equal(t, map[string]int{
			"*v1.ServiceAccount":                 1,
			"*v1.ConfigMap":                      1,
			"*v1.ClusterRole":                    3,
			"*v1.CustomResourceDefinition":       3,
			"*v1.ClusterRoleBinding":             2,
			"*v1.Role":                           1,
			"*v1.RoleBinding":                    1,
			"*v1.Service":                        2,
			"*v1.Deployment":                     1,
			"*v1.MutatingWebhookConfiguration":   1,
			"*v1.ValidatingWebhookConfiguration": 1,
		}, objs)
	})
}

func TestHelmTemplate_Deployment(t *testing.T) {
	var getDeployment = func(t *testing.T, vars ...string) *appsv1.Deployment {
		specs := getHelmSpecs(t, vars...)
		for _, obj := range specs {
			if dep, ok := obj.(*appsv1.Deployment); ok {
				return dep
			}
		}
		return nil
	}
	t.Run("The deployment has the right defaults", func(t *testing.T) {
		dep := getDeployment(t)
		assert.Equal(t, ptr.To(int32(1)), dep.Spec.Replicas)
		assert.Equal(t, metav1.ObjectMeta{
			Labels: map[string]string{
				"chart":         "clusterapioutscale-1.0.0",
				"control-plane": "release-name-controller-manager",
				"release":       "release-name",
			},
			Annotations: map[string]string{
				"kubectl.kubernetes.io/default-container": "manager",
			},
		}, dep.Spec.Template.ObjectMeta)
		require.Len(t, dep.Spec.Template.Spec.Containers, 2)
		manager := dep.Spec.Template.Spec.Containers[1]
		assert.Equal(t, "registry.hub.docker.com/outscale/cluster-api-outscale-controllers:v0.1.0", manager.Image)
		assert.Equal(t, []string{
			"--health-probe-bind-address=:8081",
			"--metrics-bind-address=127.0.0.1:8080",
			"--leader-elect",
			"--zap-log-level=5",
		}, manager.Args)
		assert.Equal(t, []corev1.EnvVar{
			{Name: "OSC_ACCESS_KEY", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "cluster-api-provider-outscale",
					},
					Key:      "access_key",
					Optional: ptr.To(true),
				},
			}},
			{Name: "OSC_SECRET_KEY", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "cluster-api-provider-outscale",
					},
					Key:      "secret_key",
					Optional: ptr.To(true),
				},
			}},
			{Name: "OSC_REGION", ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "cluster-api-provider-outscale",
					},
					Key:      "region",
					Optional: ptr.To(true),
				},
			}},
			{Name: "BACKOFF_DURATION", Value: "1"},
			{Name: "BACKOFF_FACTOR", Value: "1.5"},
			{Name: "BACKOFF_STEPS", Value: "20"},
		}, manager.Env)
		assert.Equal(t, corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("128Mi"),
				"cpu":    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				"memory": resource.MustParse("128Mi"),
				"cpu":    resource.MustParse("100m"),
			},
		}, manager.Resources)
		assert.Equal(t, &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
		}, manager.SecurityContext)
	})
	t.Run("Resources can be set", func(t *testing.T) {
		dep := getDeployment(t,
			"deployment.resources.memory.limits=64Mi", "deployment.resources.cpu.limits=10m",
			"deployment.resources.memory.requests=96Mi", "deployment.resources.cpu.requests=20m")
		require.Len(t, dep.Spec.Template.Spec.Containers, 2)
		manager := dep.Spec.Template.Spec.Containers[1]
		assert.Equal(t, corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("96Mi"),
				"cpu":    resource.MustParse("20m"),
			},
			Limits: corev1.ResourceList{
				"memory": resource.MustParse("64Mi"),
				"cpu":    resource.MustParse("10m"),
			},
		}, manager.Resources)
	})
}
