package controllers_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func runMachineTest(t *testing.T, tc testcase) {
	c, oc := loadClusterSpecs(t, tc.clusterSpec)
	m, om := loadMachineSpecs(t, tc.machineSpec)
	om.Labels = map[string]string{clusterv1.ClusterNameLabel: oc.Name}
	om.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       "Machine",
		Name:       m.Name,
	}}
	for _, fn := range tc.machinePatches {
		fn(om)
	}
	om.Spec.Node.Vm.SetDefaultValue()
	fakeScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(fakeScheme)
	_ = clusterv1.AddToScheme(fakeScheme)
	_ = apiextensionsv1.AddToScheme(fakeScheme)
	_ = infrastructurev1beta1.AddToScheme(fakeScheme)
	client := fake.NewClientBuilder().WithScheme(fakeScheme).
		WithStatusSubresource(om).WithObjects(c, oc, m, om).Build()
	mockCtrl := gomock.NewController(t)
	cs := newMockCloudServices(mockCtrl)
	for _, fn := range tc.mockFuncs {
		fn(cs)
	}
	rec := controllers.OscMachineReconciler{
		Client: client,
		Cloud:  cs,
	}
	nsn := types.NamespacedName{
		Namespace: om.Namespace,
		Name:      om.Name,
	}
	res, err := rec.Reconcile(context.TODO(), controllerruntime.Request{NamespacedName: nsn})
	if tc.hasError {
		require.Error(t, err)
		assert.Zero(t, res)
	} else {
		require.NoError(t, err)
		assert.Equal(t, tc.requeue, res.RequeueAfter > 0 || res.Requeue)
	}
	var out v1beta1.OscMachine
	err = client.Get(context.TODO(), nsn, &out)
	require.NoError(t, err, "resource was not found")
	for _, fn := range tc.machineAsserts {
		fn(t, &out)
	}
}

func TestReconcileOSCMachine_Create(t *testing.T) {
	tcs := []testcase{
		// Worker node, with 3 reconciliation loops
		{
			name:        "Creating a worker with base parameters, vm is pending",
			clusterSpec: "ready", machineSpec: "base-worker",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockNoVmFoundByName("test-cluster-api-vm-kw-"),
				mockCreateVm("i-foo", "subnet-1555ea91", []string{"sg-a093d014", "sg-0cd1f87e"}, []string{}, "test-cluster-api-vm-kw-", nil),
				mockGetVm("i-foo"), mockGetVmState("i-foo", "pending"),
			},
			hasError: true,
			machineAsserts: []assertOSCMachineFunc{
				assertHasFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
		},
		{
			name:        "worker has been created, but vm is still pending",
			clusterSpec: "ready", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{patchVmExists("i-foo", v1beta1.VmStatePending, false)},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockGetVm("i-foo"), mockGetVmState("i-foo", "pending"),
			},
			hasError:       true,
			machineAsserts: []assertOSCMachineFunc{assertVmExists("i-foo", v1beta1.VmStatePending, false)},
		},
		{
			name:        "worker has been created, and vm is now running",
			clusterSpec: "ready", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{patchVmExists("i-foo", v1beta1.VmStatePending, false)},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockGetVm("i-foo"), mockGetVmState("i-foo", "running"),
				mockGetVm("i-foo"), mockVmReadEmptyCCMTag(), mockVmSetCCMTag("i-foo", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			machineAsserts: []assertOSCMachineFunc{assertVmExists("i-foo", v1beta1.VmStateRunning, true)},
		},
		// Control plane node
		{
			name:        "Creating a controlplane with base parameters, vm is running & security groups are ok",
			clusterSpec: "ready", machineSpec: "base-controlplane",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockNoVmFoundByName("test-cluster-api-vm-kcp-"),
				mockCreateVm("i-foo", "subnet-c1a282b0", []string{"sg-750ae810", "sg-0cd1f87e"}, []string{}, "test-cluster-api-vm-kcp-", nil),
				mockGetVm("i-foo"), mockGetVmState("i-foo", "running"),
				mockLinkLoadBalancer("i-foo", "test-cluster-api-k8s"),
				mockSecurityGroupHasRule("sg-7eb16ccb", "Outbound", "tcp", "", "sg-750ae810", 6443, 6443, true),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "", "sg-7eb16ccb", 6443, 6443, true),
				mockGetVm("i-foo"), mockVmReadEmptyCCMTag(), mockVmSetCCMTag("i-foo", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertHasFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStateRunning, true),
			},
		},
		{
			name:        "Creating a controlplane with base parameters, vm is running & security groups are not ok",
			clusterSpec: "ready", machineSpec: "base-controlplane",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockNoVmFoundByName("test-cluster-api-vm-kcp-"),
				mockCreateVm("i-foo", "subnet-c1a282b0", []string{"sg-750ae810", "sg-0cd1f87e"}, []string{}, "test-cluster-api-vm-kcp-", nil),
				mockGetVm("i-foo"), mockGetVmState("i-foo", "running"),
				mockLinkLoadBalancer("i-foo", "test-cluster-api-k8s"),
				mockSecurityGroupHasRule("sg-7eb16ccb", "Outbound", "tcp", "", "sg-750ae810", 6443, 6443, false),
				mockSecurityGroupCreateRule("sg-7eb16ccb", "Outbound", "tcp", "", "sg-750ae810", 6443, 6443),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "", "sg-7eb16ccb", 6443, 6443, false),
				mockSecurityGroupCreateRule("sg-750ae810", "Inbound", "tcp", "", "sg-7eb16ccb", 6443, 6443),
				mockGetVm("i-foo"), mockVmReadEmptyCCMTag(), mockVmSetCCMTag("i-foo", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertHasFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStateRunning, true),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runMachineTest(t, tc)
		})
	}
}

func TestReconcileOSCMachine_Update(t *testing.T) {
	tcs := []testcase{
		{
			// FIXME: this is a bug, Vm should be ok.
			name:        "worker has been moved by clusterctl move, status is updated",
			clusterSpec: "ready", machineSpec: "ready-worker",
			machinePatches: []patchOSCMachineFunc{patchMoveMachine()},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14"), mockKeyPairFound("cluster-api"),
				mockVmFoundByName("test-cluster-api-vm-kw-a009b2e2-2688-406c-a7db-a0b27a1082fd", "i-foo"),
				// mockGetVm("i-046f4bd0"), mockGetVmState("i-046f4bd0", "running"),
				// mockGetVm("i-046f4bd0"), mockVmReadEmptyCCMTag(), mockVmSetCCMTag("i-046f4bd0", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			hasError: true,
			// machineAsserts: []assertOSCMachineFunc{assertVmExists("i-046f4bd0", v1beta1.VmStateRunning, true)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runMachineTest(t, tc)
		})
	}
}
