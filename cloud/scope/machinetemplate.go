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
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams is a collection of input parameters to create a new scope
type MachineTemplateScopeParams struct {
	Client             client.Client
	OscMachineTemplate *infrastructurev1beta1.OscMachineTemplate
}

// NewMachineScope create new machineScope from parameters which is called at each reconciliation iteration
func NewMachineTemplateScope(params MachineTemplateScopeParams) (*MachineTemplateScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineTemplateScope")
	}

	helper, err := patch.NewHelper(params.OscMachineTemplate, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}
	return &MachineTemplateScope{
		client:             params.Client,
		OscMachineTemplate: params.OscMachineTemplate,
		patchHelper:        helper,
	}, nil
}

// MachineTemplateScope is the basic context of the actuator that will be used
type MachineTemplateScope struct {
	client             client.Client
	patchHelper        *patch.Helper
	OscMachineTemplate *infrastructurev1beta1.OscMachineTemplate
}

// Close closes the scope of the machine configuration and status
func (m *MachineTemplateScope) Close(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.OscMachineTemplate)
}

// GetName return the name of the machine
func (m *MachineTemplateScope) GetName() string {
	return m.OscMachineTemplate.Name
}

// GetNamespace return the namespace of the machine
func (m *MachineTemplateScope) GetNamespace() string {
	return m.OscMachineTemplate.Namespace
}

func (m *MachineTemplateScope) PatchObject(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.OscMachineTemplate)
}

func (m *MachineTemplateScope) GetVmType() string {
	return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.VmType
}
func (m *MachineTemplateScope) GetTags() map[string]string {
	return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.Tags
}

func (m *MachineTemplateScope) GetReplica() int32 {
	return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.Replica
}
func (m *MachineTemplateScope) GetRole() string {
	if m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.Role != "" {
		return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.Role
	}
	return ""
}

func (m *MachineTemplateScope) GetClusterName() string {
	return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.ClusterName
}

func (m *MachineTemplateScope) GetCapacity() corev1.ResourceList {
	return m.OscMachineTemplate.Status.Capacity
}

func (m *MachineTemplateScope) SetCapacity(capacity corev1.ResourceList) {
	m.OscMachineTemplate.Status.Capacity = capacity
}
