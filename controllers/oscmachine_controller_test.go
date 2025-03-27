package controllers_test

import (
	"context"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func runMachineTest(t *testing.T, tc testcase) {
	c, oc := loadClusterSpecs(t, tc.clusterSpec, tc.clusterBaseSpec)
	m, om := loadMachineSpecs(t, tc.machineSpec, tc.machineBaseSpec)
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
	rec := controllers.OscMachineReconciler{
		Client: client,
		Tracker: &controllers.MachineResourceTracker{
			Cloud: cs,
		},
		ClusterTracker: &controllers.ClusterResourceTracker{
			Cloud: cs,
		},
		Cloud: cs,
	}
	nsn := types.NamespacedName{
		Namespace: om.Namespace,
		Name:      om.Name,
	}
	step := &tc
	for step != nil {
		for _, fn := range step.mockFuncs {
			fn(cs)
		}
		res, err := rec.Reconcile(context.TODO(), controllerruntime.Request{NamespacedName: nsn})
		if step.hasError {
			require.Error(t, err)
			assert.Zero(t, res)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tc.requeue, res.RequeueAfter > 0 || res.Requeue)
		}
		var out v1beta1.OscMachine
		err = client.Get(context.TODO(), nsn, &out)
		switch {
		case step.assertDeleted:
			require.True(t, apierrors.IsNotFound(err), "resource must have been deleted")
		default:
			require.NoError(t, err, "resource was not found")
			for _, fn := range step.machineAsserts {
				fn(t, &out)
			}
		}
		step = step.next
	}
}

func TestReconcileOSCMachine_Create(t *testing.T) {
	tcs := []testcase{
		// Worker node, with 3 reconciliation loops
		{
			name:        "Creating a worker with base parameters, vm is pending",
			clusterSpec: "ready-0.4", machineSpec: "base-worker",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-1555ea91", []string{"sg-a093d014", "sg-0cd1f87e"}, []string{}, "cluster-api-test-worker", "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
			},
			hasError: true,
			machineAsserts: []assertOSCMachineFunc{
				assertHasMachineFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
			next: &testcase{
				name:        "worker has been created, but vm is still pending",
				clusterSpec: "ready-0.4", machineSpec: "base-worker",
				machinePatches: []patchOSCMachineFunc{patchVmExists("i-foo", v1beta1.VmStatePending, false)},
				mockFuncs: []mockFunc{
					mockGetVm("i-foo", "pending"),
				},
				hasError:       true,
				machineAsserts: []assertOSCMachineFunc{assertVmExists("i-foo", v1beta1.VmStatePending, false)},
				next: &testcase{
					name:        "worker has been created, and vm is now running",
					clusterSpec: "ready-0.4", machineSpec: "base-worker",
					machinePatches: []patchOSCMachineFunc{patchVmExists("i-foo", v1beta1.VmStatePending, false)},
					mockFuncs: []mockFunc{
						mockGetVm("i-foo", "running"),
						mockVmReadCCMTag(false), mockVmSetCCMTag("i-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520"),
					},
					machineAsserts: []assertOSCMachineFunc{
						assertVmExists("i-foo", v1beta1.VmStateRunning, true),
						assertVolumesAreConfigured(),
					},
				},
			},
		},

		// Control plane node
		{
			name:        "Creating a controlplane with base parameters, vm is running & security groups are ok",
			clusterSpec: "ready-0.4", machineSpec: "base-controlplane",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-c1a282b0", []string{"sg-750ae810", "sg-0cd1f87e"}, []string{}, "cluster-api-test-controlplane", "uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertHasMachineFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
			hasError: true,
			next: &testcase{
				mockFuncs: []mockFunc{
					mockGetVm("i-foo", "running"),
					mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
					mockLinkLoadBalancer("i-foo", "test-cluster-api-k8s"),
					mockVmReadCCMTag(false), mockVmSetCCMTag("i-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				},
				machineAsserts: []assertOSCMachineFunc{
					assertHasMachineFinalizer(),
					assertVmExists("i-foo", v1beta1.VmStateRunning, true),
				},
			},
		},
		{
			name:        "Creating a controlplane with base parameters, vm is running & security groups are not ok",
			clusterSpec: "ready-0.4", machineSpec: "base-controlplane",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-c1a282b0", []string{"sg-750ae810", "sg-0cd1f87e"}, []string{}, "cluster-api-test-controlplane", "uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
			},
			hasError: true,
			machineAsserts: []assertOSCMachineFunc{
				assertHasMachineFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
			next: &testcase{
				mockFuncs: []mockFunc{
					mockGetVm("i-foo", "running"),
					mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-9e1db9c4-bf0a-4583-8999-203ec002c520"),
					mockLinkLoadBalancer("i-foo", "test-cluster-api-k8s"),
					mockVmReadCCMTag(false), mockVmSetCCMTag("i-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				},
				machineAsserts: []assertOSCMachineFunc{
					assertHasMachineFinalizer(),
					assertVmExists("i-foo", v1beta1.VmStateRunning, true),
				},
			},
		},

		// Volumes
		{
			name:        "Creating a vm with an additional volume",
			clusterSpec: "ready-0.4", machineSpec: "base-worker-volumes",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateVmWithVolumes("i-foo", []infrastructurev1beta1.OscVolume{{
					Name:       "data",
					Size:       15,
					VolumeType: "io1",
					Iops:       500,
					Device:     "/dev/sdb",
				}}, "vol-bar"),
			},
			hasError: true,
			machineAsserts: []assertOSCMachineFunc{
				assertHasMachineFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
			next: &testcase{
				mockFuncs: []mockFunc{
					mockGetVm("i-foo", "running"),
					mockVmReadCCMTag(false), mockVmSetCCMTag("i-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				},
				machineAsserts: []assertOSCMachineFunc{
					assertHasMachineFinalizer(),
					assertVmExists("i-foo", v1beta1.VmStateRunning, true),
					assertVolumesAreConfigured("/dev/sdb", "vol-bar"),
				},
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
			name:        "worker has been moved by clusterctl move, status is updated",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker",
			machinePatches: []patchOSCMachineFunc{patchMoveMachine()},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockVmReadCCMTag(true),
			},
			machineAsserts: []assertOSCMachineFunc{},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runMachineTest(t, tc)
		})
	}
}
