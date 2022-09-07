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

	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/noderefutil"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams is a collection of input parameters to create a new scope.
type MachineScopeParams struct {
	OscClient  *OscClient
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	OscCluster *infrastructurev1beta1.OscCluster
	OscMachine *infrastructurev1beta1.OscMachine
}

// NewMachineScope create new machineScope from parameters which is called at each reconciliation iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("cluster is required when creating a MachineScope")
	}
	if params.OscCluster == nil {
		return nil, errors.New("oscCluster is required when creating a MachineScope")
	}
	if params.OscMachine == nil {
		return nil, errors.New("oscMachine is required when creating a MachineScope")
	}
	if params.Logger == (logr.Logger{}) {
		params.Logger = klogr.New()
	}

	client, err := newOscClient()

	if err != nil {
		return nil, fmt.Errorf("%w failed to create Osc Client", err)
	}

	if params.OscClient == nil {
		params.OscClient = client
	}
	if params.OscClient.api == nil {
		params.OscClient.api = client.api
	}
	if params.OscClient.auth == nil {
		params.OscClient.auth = client.auth
	}

	helper, err := patch.NewHelper(params.OscMachine, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %+v", err)
	}
	return &MachineScope{
		client:      params.Client,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		OscCluster:  params.OscCluster,
		OscMachine:  params.OscMachine,
		Logger:      params.Logger,
		patchHelper: helper,
	}, nil
}

// MachineScope is the basic context of the actuator that will be used.
type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine
	OscClient   *OscClient
	OscCluster  *infrastructurev1beta1.OscCluster
	OscMachine  *infrastructurev1beta1.OscMachine
}

// Close closes the scope of the machine configuration and status.
func (m *MachineScope) Close() error {
	return m.patchHelper.Patch(context.TODO(), m.OscMachine)
}

// GetName return the name of the machine.
func (m *MachineScope) GetName() string {
	return m.OscMachine.Name
}

// GetNamespace return the namespace of the machine.
func (m *MachineScope) GetNamespace() string {
	return m.OscMachine.Namespace
}

// GetUID return the uid of the machine.
func (m *MachineScope) GetUID() string {
	return string(m.Machine.UID)
}

// GetAuth return outscale api context.
func (m *MachineScope) GetAuth() context.Context {
	return m.OscClient.auth
}

// GetAPI return outscale api credential.
func (m *MachineScope) GetAPI() *osc.APIClient {
	return m.OscClient.api
}

// GetVolume return the volume of the cluster.
func (m *MachineScope) GetVolume() []*infrastructurev1beta1.OscVolume {
	return m.OscMachine.Spec.Node.Volumes
}

// GetVolumeSubregionName return the volume subregionName.
func (m *MachineScope) GetVolumeSubregionName(name string) string {
	volumes := m.OscMachine.Spec.Node.Volumes
	for _, volume := range volumes {
		if volume.Name == name {
			return volume.SubregionName
		}
	}
	return ""
}

// GetVM return the vm.
func (m *MachineScope) GetVM() *infrastructurev1beta1.OscVM {
	return &m.OscMachine.Spec.Node.VM
}

// GetVMPrivateIPS return the vm privateIps.
func (m *MachineScope) GetVMPrivateIPS() *[]infrastructurev1beta1.OscPrivateIPElement {
	return &m.GetVM().PrivateIPS
}

// GetVMSecurityGroups return the vm securityGroups.
func (m *MachineScope) GetVMSecurityGroups() *[]infrastructurev1beta1.OscSecurityGroupElement {
	return &m.GetVM().SecurityGroupNames
}

// GetLinkPublicIPRef get the status of linkPublicIpRef (a Map with tag name with machine uid associate with resource response id).
func (m *MachineScope) GetLinkPublicIPRef() *infrastructurev1beta1.OscResourceMapReference {
	return &m.OscMachine.Status.Node.LinkPublicIPRef
}

// GetVolumeRef get the status of volume (a Map with tag name with machine uid associate with resource response id).
func (m *MachineScope) GetVolumeRef() *infrastructurev1beta1.OscResourceMapReference {
	return &m.OscMachine.Status.Node.VolumeRef
}

// GetVMRef get the status of vm (a Map with tag name with machine uid associate with resource response id).
func (m *MachineScope) GetVMRef() *infrastructurev1beta1.OscResourceMapReference {
	return &m.OscMachine.Status.Node.VMRef
}

// IsControlPlane check if it is control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// GetNode return the node.
func (m *MachineScope) GetNode() *infrastructurev1beta1.OscNode {
	return &m.OscMachine.Spec.Node
}

// GetRole return the role.
func (m *MachineScope) GetRole() string {
	if m.IsControlPlane() {
		return infrastructurev1beta1.APIServerRoleTagValue
	}
	return infrastructurev1beta1.NodeRoleTagValue
}

// GetProviderID return the providerID.
func (m *MachineScope) GetProviderID() string {
	if m.OscMachine.Spec.ProviderID != nil {
		return *m.OscMachine.Spec.ProviderID
	}
	return ""
}

// GetInstanceID return the instanceID.
func (m *MachineScope) GetInstanceID() string {
	parsed, err := noderefutil.NewProviderID(m.GetProviderID())
	if err != nil {
		return ""
	}
	return parsed.ID()
}

// SetProviderID set the instanceID.
func (m *MachineScope) SetProviderID(vmID string) {
	pid := fmt.Sprintf("osc://%s", vmID)
	m.OscMachine.Spec.ProviderID = pointer.StringPtr(pid)
}

// GetVMState return the vmState.
func (m *MachineScope) GetVMState() *infrastructurev1beta1.VMState {
	return m.OscMachine.Status.VMState
}

// SetVMState set vmstate.
func (m *MachineScope) SetVMState(v infrastructurev1beta1.VMState) {
	m.OscMachine.Status.VMState = &v
}

// SetReady set machine status ready.
func (m *MachineScope) SetReady() {
	m.OscMachine.Status.Ready = true
}

// SetNotReady set machine status not ready.
func (m *MachineScope) SetNotReady() {
	m.OscMachine.Status.Ready = false
}

// SetFailureMessage set failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.OscMachine.Status.FailureMessage = pointer.StringPtr(v.Error())
}

// SetFailureReason set failure reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.OscMachine.Status.FailureReason = &v
}

// SetAddresses set node address.
func (m *MachineScope) SetAddresses(addrs []corev1.NodeAddress) {
	m.OscMachine.Status.Addresses = addrs
}

// PatchObject keep the machine configuration and status.
func (m *MachineScope) PatchObject() error {
	applicableConditions := []clusterv1.ConditionType{
		infrastructurev1beta1.VMReadyCondition,
	}
	conditions.SetSummary(m.OscMachine,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(m.OscMachine.ObjectMeta.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)
	return m.patchHelper.Patch(
		context.TODO(),
		m.OscMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrastructurev1beta1.VMReadyCondition,
		}})
}

// GetBootstrapData return bootstrapData.
func (m *MachineScope) GetBootstrapData() (string, error) {
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.DataSecretName is nil")
	}
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.GetNamespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", fmt.Errorf("%w failed to retrieve bootstrap data secret for OscMachine %s/%s", err, m.GetNamespace(), m.GetName())
	}
	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}
	return string(value), nil
}
