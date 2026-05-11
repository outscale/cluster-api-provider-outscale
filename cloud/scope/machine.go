/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package scope

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams is a collection of input parameters to create a new scope
type MachineScopeParams struct {
	Client     client.Client
	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	OscCluster *infrastructurev1beta2.OscCluster
	OscMachine *infrastructurev1beta2.OscMachine
}

// NewMachineScope create new machineScope from parameters which is called at each reconciliation iteration
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.OscCluster == nil {
		return nil, errors.New("OscCluster is required when creating a MachineScope")
	}
	if params.OscMachine == nil {
		return nil, errors.New("OscMachine is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.OscMachine, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}
	return &MachineScope{
		client:      params.Client,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		OscCluster:  params.OscCluster,
		OscMachine:  params.OscMachine,
		patchHelper: helper,
	}, nil
}

// MachineScope is the basic context of the actuator that will be used
type MachineScope struct {
	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine
	OscCluster  *infrastructurev1beta2.OscCluster
	OscMachine  *infrastructurev1beta2.OscMachine
}

// Close closes the scope of the machine configuration and status
func (m *MachineScope) Close(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.OscMachine)
}

// GetName returns the name of the machine
func (m *MachineScope) GetName() string {
	return m.OscMachine.Name
}

func (m *MachineScope) GetClientToken(clusterScope *ClusterScope) string {
	ct := m.OscMachine.Name + "-" + clusterScope.GetUID()
	if len(ct) > 64 {
		ct = ct[len(ct)-64:]
	}
	return ct
}

// GetNamespace returns the namespace of the machine
func (m *MachineScope) GetNamespace() string {
	return m.OscMachine.Namespace
}

// GetUID returns the uid of the machine
func (m *MachineScope) GetUID() string {
	return string(m.Machine.UID)
}

// GetAdditionalVolumes returns the volume of the cluster
func (m *MachineScope) GetAdditionalVolumes() []infrastructurev1beta2.OscVolume {
	return m.OscMachine.Spec.Vm.AdditionalVolumes
}

// GetVm returns the vm
func (m *MachineScope) GetVm() infrastructurev1beta2.OscVm {
	return m.OscMachine.Spec.Vm
}

// GetImage returns the image
func (m *MachineScope) GetImage() *infrastructurev1beta2.OscImage {
	return &m.OscMachine.Spec.Vm.Image
}

// GetVmSecurityGroups returns the VM securityGroups
func (m *MachineScope) GetVmSecurityGroups() []infrastructurev1beta2.OscSecurityGroupElement {
	return m.GetVm().SecurityGroupNames
}

// GetPlacement returns the VM placement constraints.
func (m *MachineScope) GetPlacement() infrastructurev1beta2.OscPlacement {
	repulse := m.GetVm().Placement
	if m.IsControlPlane() || repulse.RepulseCluster != "" || repulse.RepulseServer != nil || repulse.AttractCluster != "" || repulse.AttractServer != "" {
		return repulse
	}

	hasRepulse := slices.ContainsFunc(lo.Keys(m.GetVm().Tags), func(k string) bool {
		return strings.HasPrefix(k, "osc.fcu.repulse") || strings.HasPrefix(k, "osc.fcu.attract")
	})
	if !hasRepulse {
		return infrastructurev1beta2.OscPlacement{
			RepulseServer: ptr.To(m.Machine.Labels[clusterv1.MachineDeploymentNameLabel]),
		}
	}

	return repulse
}

// IsControlPlane check if it is control plane
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// GetProviderID return the providerID
func (m *MachineScope) GetProviderID() string {
	if m.OscMachine.Spec.ProviderID != nil {
		return *m.OscMachine.Spec.ProviderID
	}
	return ""
}

// GetInstanceID return the instanceID
func (m *MachineScope) GetInstanceID() string {
	parsed, err := NewProviderID(m.GetProviderID())
	if err != nil {
		return ""
	}
	return parsed.ID()
}

// SetProviderID set the instanceID
func (m *MachineScope) SetProviderID(subregionName string, vmId string) {
	pid := fmt.Sprintf("aws:///%s/%s", subregionName, vmId)
	m.OscMachine.Spec.ProviderID = ptr.To(pid)
}

// GetVmState return the vmState
func (m *MachineScope) GetVmState() *osc.VmState {
	return m.OscMachine.Status.VmState
}

// SetVmState set vmstate
func (m *MachineScope) SetVmState(v osc.VmState) {
	m.OscMachine.Status.VmState = &v
}

// SetReady set machine status ready
func (m *MachineScope) SetReady() {
	m.OscMachine.Status.Initialization.Provisioned = new(true)
}

// SetAddresses set node address
func (m *MachineScope) SetAddresses(addrs []clusterv1.MachineAddress) {
	m.OscMachine.Status.Addresses = addrs
}

// SetFailureDomain set failure domain.
func (m *MachineScope) SetFailureDomain(subregion string) {
	m.OscMachine.Status.FailureDomain = subregion
}

// GetResources returns the resource list from the OscCluster status.
func (s *MachineScope) GetResources() *infrastructurev1beta2.OscMachineResources {
	return &s.OscMachine.Status.Resources
}

// NeedReconciliation returns true if a reconciler needs to run.
func (s *MachineScope) NeedReconciliation(reconciler infrastructurev1beta2.Reconciler) bool {
	if s.OscMachine.Status.ReconcilerGeneration == nil {
		return true
	}
	if s.OscMachine.Status.ReconcilerGeneration[reconciler] < s.OscMachine.Generation {
		return true
	}
	r := s.OscMachine.Spec.ReconciliationRule
	if r == nil {
		return false
	}
	switch r.Mode {
	case infrastructurev1beta2.ReconciliationModeAlways:
		return true
	case infrastructurev1beta2.ReconciliationModeRandom:
		return Rand() < r.ReconciliationChance
	default:
		return false
	}
}

// SetReconciliationGeneration marks a reconciler as having finished its job for a specific cluster generation.
func (s *MachineScope) SetReconciliationGeneration(reconciler infrastructurev1beta2.Reconciler) {
	if s.OscMachine.Status.ReconcilerGeneration == nil {
		s.OscMachine.Status.ReconcilerGeneration = map[infrastructurev1beta2.Reconciler]int64{}
	}
	s.OscMachine.Status.ReconcilerGeneration[reconciler] = s.OscMachine.Generation
}

// PatchObject keep the machine configuration and status
func (m *MachineScope) PatchObject(ctx context.Context) error {
	applicableConditions := []clusterv1.ConditionType{
		infrastructurev1beta2.VmReadyCondition,
	}
	conditions.SetSummary(m.OscMachine,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(m.OscMachine.ObjectMeta.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)
	return m.patchHelper.Patch(
		ctx,
		m.OscMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrastructurev1beta2.VmReadyCondition,
		}})
}

// GetBootstrapData return bootstrapData
func (m *MachineScope) GetBootstrapData(ctx context.Context) (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: DataSecretName is not set")
	}
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.GetNamespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("failed to retrieve bootstrap data secret: %w", err)
	}
	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return string(value), nil
}
