package controllers_test

import (
	"context"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
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
		Client:   client,
		Recorder: record.NewFakeRecorder(100),
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
			assert.Equal(t, step.requeue, res.RequeueAfter > 0 || res.Requeue)
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
			requeue: true,
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
				requeue:        true,
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
			name:        "Creating a controlplane with base parameters, vm is running & LB is ok",
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
			requeue: true,
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
			name:        "Creating a controlplane with base parameters, vm is running & LB is not found",
			clusterSpec: "ready-0.4", machineSpec: "base-controlplane",
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-c1a282b0", []string{"sg-750ae810", "sg-0cd1f87e"}, []string{}, "cluster-api-test-controlplane", "uster-api-test-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
			},
			requeue: true,
			machineAsserts: []assertOSCMachineFunc{
				assertHasMachineFinalizer(),
				assertVmExists("i-foo", v1beta1.VmStatePending, false),
			},
			next: &testcase{
				mockFuncs: []mockFunc{
					mockGetVm("i-foo", "running"),
					mockGetLoadBalancer("test-cluster-api-k8s", nil),
				},
				hasError: true,
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
			requeue: true,
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

		// Public IPs
		{
			name:        "Creating a vm with a dynamic public IP",
			clusterSpec: "ready-0.4", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{
				patchUsePublicIP(),
			},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreatePublicIp("cluster-api-test-worker", "9e1db9c4-bf0a-4583-8999-203ec002c520", "ipalloc-worker", "1.2.3.4"),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-1555ea91", []string{"sg-a093d014", "sg-0cd1f87e"}, []string{}, "cluster-api-test-worker", "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", map[string]string{
					compute.AutoAttachExternapIPTag: "1.2.3.4",
				}),
			},
			requeue: true,
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
					assertStatusMachineResources(infrastructurev1beta1.OscMachineResources{
						Image: map[string]string{
							"default": "ami-foo",
						},
						Vm: map[string]string{
							"default": "i-foo",
						},
						PublicIPs: map[string]string{
							"default": "ipalloc-worker",
						},
						Volumes: map[string]string{
							"/dev/sda1": "vol-foo",
						},
					}),
				},
			},
		},
		{
			name:        "Creating a vm with a public IP from a pool",
			clusterSpec: "ready-0.4", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{
				patchUsePublicIP("pool-foo"),
			},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockListPublicIpsFromPool("pool-foo", []osc.PublicIp{{PublicIpId: ptr.To("ipalloc-foo"), PublicIp: ptr.To("1.2.3.4")}}),
				mockCreateVmNoVolumes("i-foo", "ami-foo", "subnet-1555ea91", []string{"sg-a093d014", "sg-0cd1f87e"}, []string{}, "cluster-api-test-worker", "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", map[string]string{
					compute.AutoAttachExternapIPTag: "1.2.3.4",
				}),
			},
			requeue: true,
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
					assertStatusMachineResources(infrastructurev1beta1.OscMachineResources{
						Image: map[string]string{
							"default": "ami-foo",
						},
						Vm: map[string]string{
							"default": "i-foo",
						},
						PublicIPs: map[string]string{
							"default": "ipalloc-foo",
						},
						Volumes: map[string]string{
							"/dev/sda1": "vol-foo",
						},
					}),
				},
			},
		},
		{
			name:        "When the IP pool is empty, an error is returned",
			clusterSpec: "ready-0.4", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{
				patchUsePublicIP("pool-foo"),
			},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockListPublicIpsFromPool("pool-foo", []osc.PublicIp{}),
			},
			hasError: true,
		},
		{
			name:        "When the IP pool is fully used, an error is returned",
			clusterSpec: "ready-0.4", machineSpec: "base-worker",
			machinePatches: []patchOSCMachineFunc{
				patchUsePublicIP("pool-foo"),
			},
			mockFuncs: []mockFunc{
				mockImageFoundByName("ubuntu-2004-2004-kubernetes-v1.25.9-2023-04-14", "01234", "ami-foo"),
				mockGetVmFromClientToken("cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.VmResourceType, "cluster-api-test-worker-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockListPublicIpsFromPool("pool-foo", []osc.PublicIp{{LinkPublicIpId: ptr.To("ipassoc-foo"), PublicIp: ptr.To("1.2.3.4")}}),
			},
			hasError: true,
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
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.4",
			machineBaseSpec: "ready-worker",
			machinePatches:  []patchOSCMachineFunc{patchMoveMachine()},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockVmReadCCMTag(true),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertStatusMachineResources(infrastructurev1beta1.OscMachineResources{
					Vm: map[string]string{
						"default": "i-046f4bd0",
					},
					Volumes: map[string]string{
						"/dev/sda1": "vol-foo",
					},
				}),
			},
		},
		{
			name:        "0.5 worker has been moved by clusterctl move, status is updated",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.5",
			machineBaseSpec: "ready-worker",
			machinePatches:  []patchOSCMachineFunc{patchMoveMachine()},
			mockFuncs: []mockFunc{
				mockGetVmFromClientToken("luster-api-md-0-6p8qk-qgvhr-9e1db9c4-bf0a-4583-8999-203ec002c520", &osc.Vm{
					VmId:  ptr.To("i-worker"),
					State: ptr.To("running"),
				}),
				mockVmReadCCMTag(true),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertStatusMachineResources(infrastructurev1beta1.OscMachineResources{
					Vm: map[string]string{
						"default": "i-worker",
					},
					Volumes: map[string]string{
						"/dev/sda1": "vol-foo",
					},
				}),
			},
		},
		{
			name:        "0.5 worker has been moved by clusterctl move, status is updated",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.5",
			machineBaseSpec: "ready-worker",
			machinePatches: []patchOSCMachineFunc{
				patchMoveMachine(),
				patchUsePublicIP(),
			},
			mockFuncs: []mockFunc{
				mockGetVmFromClientToken("luster-api-md-0-6p8qk-qgvhr-9e1db9c4-bf0a-4583-8999-203ec002c520", &osc.Vm{
					VmId:     ptr.To("i-worker"),
					PublicIp: ptr.To("1.2.3.4"),
					State:    ptr.To("running"),
				}),
				mockGetPublicIpByIp("1.2.3.4", "ipalloc-worker"),
				mockVmReadCCMTag(true),
			},
			machineAsserts: []assertOSCMachineFunc{
				assertStatusMachineResources(infrastructurev1beta1.OscMachineResources{
					Vm: map[string]string{
						"default": "i-worker",
					},
					PublicIPs: map[string]string{
						"default": "ipalloc-worker",
					},
					Volumes: map[string]string{
						"/dev/sda1": "vol-foo",
					},
				}),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runMachineTest(t, tc)
		})
	}
}

func TestReconcileOSCMachine_Delete(t *testing.T) {
	tcs := []testcase{
		{
			name:        "deleting a 0.4 machine",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.4",
			machineBaseSpec: "ready-worker",
			machinePatches:  []patchOSCMachineFunc{patchDeleteMachine()},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockDeleteVm("i-046f4bd0"),
			},
			assertDeleted: true,
		},
		{
			name:        "deleting a 0.5 machine",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.5",
			machineBaseSpec: "ready-worker",
			machinePatches:  []patchOSCMachineFunc{patchDeleteMachine()},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockDeleteVm("i-046f4bd0"),
			},
			assertDeleted: true,
		},
		{
			name:        "deleting a 0.5 machine with a public ip",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.5",
			machineBaseSpec: "ready-worker",
			machinePatches: []patchOSCMachineFunc{
				patchDeleteMachine(),
				patchUsePublicIP(),
				patchPublicIPStatus("ipalloc-worker"),
			},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockDeleteVm("i-046f4bd0"),
				mockPublicIpFound("ipalloc-worker"),
				mockDeletePublicIp("ipalloc-worker"),
			},
			assertDeleted: true,
		},
		{
			name:        "deleting a 0.5 machine with a public ip from a pool",
			clusterSpec: "ready-0.4", machineSpec: "ready-worker-0.5",
			machineBaseSpec: "ready-worker",
			machinePatches: []patchOSCMachineFunc{
				patchDeleteMachine(),
				patchUsePublicIP("pool-foo"),
				patchPublicIPStatus("ipalloc-worker"),
			},
			mockFuncs: []mockFunc{
				mockGetVm("i-046f4bd0", "running"),
				mockDeleteVm("i-046f4bd0"),
			},
			assertDeleted: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runMachineTest(t, tc)
		})
	}
}
