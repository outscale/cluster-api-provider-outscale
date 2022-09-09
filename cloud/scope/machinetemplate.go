package scope

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineScopeParams is a collection of input parameters to create a new scope
type MachineTemplateScopeParams struct {
	OscClient          *OscClient
	Client             client.Client
	Logger             logr.Logger
	OscMachineTemplate *infrastructurev1beta1.OscMachineTemplate
}

// NewMachineScope create new machineScope from parameters which is called at each reconciliation iteration
func NewMachineTemplateScope(params MachineTemplateScopeParams) (*MachineTemplateScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineTemplateScope")
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

	helper, err := patch.NewHelper(params.OscMachineTemplate, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %+v", err)
	}
	return &MachineTemplateScope{
		client:             params.Client,
		OscMachineTemplate: params.OscMachineTemplate,
		Logger:             params.Logger,
		patchHelper:        helper,
	}, nil
}

// MachineTemplateScope is the basic context of the actuator that will be used
type MachineTemplateScope struct {
	logr.Logger
	client             client.Client
	patchHelper        *patch.Helper
	OscMachineTemplate *infrastructurev1beta1.OscMachineTemplate
}

// Close closes the scope of the machine configuration and status
func (m *MachineTemplateScope) Close() error {
	return m.patchHelper.Patch(context.TODO(), m.OscMachineTemplate)
}

// GetName return the name of the machine
func (m *MachineTemplateScope) GetName() string {
	return m.OscMachineTemplate.Name
}

// GetNamespace return the namespace of the machine
func (m *MachineTemplateScope) GetNamespace() string {
	return m.OscMachineTemplate.Namespace
}

func (m *MachineTemplateScope) PatchObject() error {
	return m.patchHelper.Patch(context.TODO(), m.OscMachineTemplate)
}

func (m *MachineTemplateScope) GetVmType() string {
	return m.OscMachineTemplate.Spec.Template.Spec.Node.Vm.VmType
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
