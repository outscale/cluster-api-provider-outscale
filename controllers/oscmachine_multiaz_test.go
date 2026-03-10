/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers_test

import (
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMultiAZAllocator(t *testing.T) {
	allocated := infrastructurev1beta1.OscMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo1",
			Namespace: "foo",
			Labels: map[string]string{
				clusterv1.MachineDeploymentNameLabel: "foo",
			},
		},
		Status: infrastructurev1beta1.OscMachineStatus{
			FailureDomain: ptr.To("eu-west-2a"),
		},
	}
	nonallocated1 := infrastructurev1beta1.OscMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo2",
			Namespace: "foo",
			Labels: map[string]string{
				clusterv1.MachineDeploymentNameLabel: "foo",
			},
		},
		Status: infrastructurev1beta1.OscMachineStatus{},
	}
	nonallocated2 := infrastructurev1beta1.OscMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo3",
			Namespace: "foo",
			Labels: map[string]string{
				clusterv1.MachineDeploymentNameLabel: "foo",
			},
		},
		Status: infrastructurev1beta1.OscMachineStatus{},
	}
	testClient := func() client.Client {
		oms := &infrastructurev1beta1.OscMachineList{
			Items: []infrastructurev1beta1.OscMachine{
				allocated,
				nonallocated1,
				nonallocated2,
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar1",
						Namespace: "foo",
						Labels: map[string]string{
							clusterv1.MachineDeploymentNameLabel: "bar",
						},
					},
					Status: infrastructurev1beta1.OscMachineStatus{
						FailureDomain: ptr.To("eu-west-2b"),
					},
				},
			},
		}
		fakeScheme := runtime.NewScheme()
		_ = clientgoscheme.AddToScheme(fakeScheme)
		_ = clusterv1.AddToScheme(fakeScheme)
		_ = apiextensionsv1.AddToScheme(fakeScheme)
		_ = infrastructurev1beta1.AddToScheme(fakeScheme)
		return fake.NewClientBuilder().WithScheme(fakeScheme).WithLists(oms).Build()
	}
	t.Run("If already configured, the allocated az is returned", func(t *testing.T) {
		m := allocated
		a := controllers.NewMultiAZAllocator(testClient())
		az, err := a.AllocateAZ(t.Context(), &m, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, *m.Status.FailureDomain, az)
	})
	t.Run("A non allocated machines is allocated to new azs", func(t *testing.T) {
		m := nonallocated1
		a := controllers.NewMultiAZAllocator(testClient())
		az, err := a.AllocateAZ(t.Context(), &m, []string{"eu-west-2a", "eu-west-2b"})
		require.NoError(t, err)
		assert.Equal(t, "eu-west-2b", az)
	})
	t.Run("Non allocated machiness are allocated to all new azs", func(t *testing.T) {
		a := controllers.NewMultiAZAllocator(testClient())
		m1 := nonallocated1
		az1, err := a.AllocateAZ(t.Context(), &m1, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		m2 := nonallocated2
		az2, err := a.AllocateAZ(t.Context(), &m2, []string{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.NotEqual(t, az1, az2)
		assert.Contains(t, []string{"eu-west-2b", "eu-west-2c"}, az1)
		assert.Contains(t, []string{"eu-west-2b", "eu-west-2c"}, az2)
	})
}
