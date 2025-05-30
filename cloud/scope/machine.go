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

package scope

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams is a collection of input parameters to create a new scope
type MachineScopeParams struct {
	OscClient  *cloud.OscClient
	Client     client.Client
	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	OscCluster *infrastructurev1beta1.OscCluster
	OscMachine *infrastructurev1beta1.OscMachine
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
		OscClient:   params.OscClient,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		OscCluster:  params.OscCluster,
		OscMachine:  params.OscMachine,
		patchHelper: helper,
	}, nil
}

// MachineScope is the basic context of the actuator that will be used
type MachineScope struct {
	OscClient   *cloud.OscClient
	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine
	OscCluster  *infrastructurev1beta1.OscCluster
	OscMachine  *infrastructurev1beta1.OscMachine
}

// Close closes the scope of the machine configuration and status
func (m *MachineScope) Close(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.OscMachine)
}

// GetName return the name of the machine
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

// GetNamespace return the namespace of the machine
func (m *MachineScope) GetNamespace() string {
	return m.OscMachine.Namespace
}

// GetUID return the uid of the machine
func (m *MachineScope) GetUID() string {
	return string(m.Machine.UID)
}

// GetAuth return outscale api context
func (m *MachineScope) GetAuth() context.Context {
	return m.OscClient.Auth
}

// GetApi return outscale api credential
func (m *MachineScope) GetApi() *osc.APIClient {
	return m.OscClient.API
}

// GetVolumes return the volume of the cluster
func (m *MachineScope) GetVolumes() []infrastructurev1beta1.OscVolume {
	return m.OscMachine.Spec.Node.Volumes
}

// GetVm return the vm
func (m *MachineScope) GetVm() infrastructurev1beta1.OscVm {
	return m.OscMachine.Spec.Node.Vm
}

// GetImage return the image
func (m *MachineScope) GetImage() *infrastructurev1beta1.OscImage {
	return &m.OscMachine.Spec.Node.Image
}

// SetImageId set ImageId
func (m *MachineScope) SetImageId(imageId string) {
	m.OscMachine.Spec.Node.Vm.ImageId = imageId
}

// GetImageId return ImageId
func (m *MachineScope) GetImageId() string {
	return m.GetVm().ImageId
}

// GetVmPrivateIps return the vm privateIps
func (m *MachineScope) GetVmPrivateIps() []infrastructurev1beta1.OscPrivateIpElement {
	return m.GetVm().PrivateIps
}

// GetVmSecurityGroups return the vm securityGroups
func (m *MachineScope) GetVmSecurityGroups() []infrastructurev1beta1.OscSecurityGroupElement {
	return m.GetVm().SecurityGroupNames
}

// GetLinkPublicIpRef get the status of linkPublicIpRef (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetLinkPublicIpRef() *infrastructurev1beta1.OscResourceReference {
	return &m.OscMachine.Status.Node.LinkPublicIpRef
}

// GetLinkPublicIpRef get the status of linkPublicIpRef (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetPublicIpIdRef() *infrastructurev1beta1.OscResourceReference {
	return &m.OscMachine.Status.Node.PublicIpIdRef
}

// GetVolumeRef get the status of volume (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetVolumeRef() *infrastructurev1beta1.OscResourceReference {
	ref := &m.OscMachine.Status.Node.VolumeRef
	if ref.ResourceMap == nil {
		ref.ResourceMap = make(map[string]string)
	}
	return ref
}

// GetVmRef get the status of vm (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetVmRef() *infrastructurev1beta1.OscResourceReference {
	return &m.OscMachine.Status.Node.VmRef
}

// GetImageRef get the status of image (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetImageRef() *infrastructurev1beta1.OscResourceReference {
	return &m.OscMachine.Status.Node.ImageRef
}

// GetKeyPairRef get the status of key pair (a Map with tag name with machine uid associate with resource response id)
func (m *MachineScope) GetKeypairRef() *infrastructurev1beta1.OscResourceReference {
	return &m.OscMachine.Status.Node.KeypairRef
}

// IsControlPlane check if it is control plane
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// GetNode return the node
func (m *MachineScope) GetNode() *infrastructurev1beta1.OscNode {
	return &m.OscMachine.Spec.Node
}

// GetRole return the role
func (m *MachineScope) GetRole() string {
	if m.IsControlPlane() {
		return infrastructurev1beta1.APIServerRoleTagValue
	}
	return infrastructurev1beta1.NodeRoleTagValue
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
func (m *MachineScope) GetVmState() *infrastructurev1beta1.VmState {
	return m.OscMachine.Status.VmState
}

// SetVmState set vmstate
func (m *MachineScope) SetVmState(v infrastructurev1beta1.VmState) {
	m.OscMachine.Status.VmState = &v
}

// SetReady set machine status ready
func (m *MachineScope) SetReady() {
	m.OscMachine.Status.Ready = true
}

// SetReady set machine status not ready
func (m *MachineScope) SetNotReady() {
	m.OscMachine.Status.Ready = false
}

// SetFailureMessage set failure message
func (m *MachineScope) SetFailureMessage(v error) {
	m.OscMachine.Status.FailureMessage = ptr.To(v.Error())
}

// SetFailureReason set failure reason
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.OscMachine.Status.FailureReason = &v
}

// SetAddresses set node address
func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.OscMachine.Status.Addresses = addrs
}

// SetFailureDomain set failure domain.
func (m *MachineScope) SetFailureDomain(subregion string) {
	m.OscMachine.Status.FailureDomain = &subregion
}

// GetResources returns the resource list from the OscCluster status.
func (s *MachineScope) GetResources() *infrastructurev1beta1.OscMachineResources {
	return &s.OscMachine.Status.Resources
}

// NeedReconciliation returns true if a reconciler needs to run.
func (s *MachineScope) NeedReconciliation(reconciler infrastructurev1beta1.Reconciler) bool {
	if s.OscMachine.Status.ReconcilerGeneration == nil {
		return true
	}
	return s.OscMachine.Status.ReconcilerGeneration[reconciler] < s.OscMachine.Generation
}

// SetReconciliationGeneration marks a reconciler as having finished its job for a specific cluster generation.
func (s *MachineScope) SetReconciliationGeneration(reconciler infrastructurev1beta1.Reconciler) {
	if s.OscMachine.Status.ReconcilerGeneration == nil {
		s.OscMachine.Status.ReconcilerGeneration = map[infrastructurev1beta1.Reconciler]int64{}
	}
	s.OscMachine.Status.ReconcilerGeneration[reconciler] = s.OscMachine.Generation
}

// PatchObject keep the machine configuration and status
func (m *MachineScope) PatchObject(ctx context.Context) error {
	applicableConditions := []clusterv1.ConditionType{
		infrastructurev1beta1.VmReadyCondition,
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
			infrastructurev1beta1.VmReadyCondition,
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
