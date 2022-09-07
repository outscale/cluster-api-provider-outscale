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

package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	// NetReadyCondition is the status of net.
	NetReadyCondition clusterv1.ConditionType = "NetReady"
	// NetCreationStartedReason is the status start of net.
	NetCreationStartedReason = "NetCreationStarted"
	// NetReconciliationFailedReason is the status failed of net.
	NetReconciliationFailedReason = "NetReconciliationFailed"
)

const (
	// SubnetsReadyCondition is the status of subnet.
	SubnetsReadyCondition clusterv1.ConditionType = "SubnetsReady"
	// SubnetsReconciliationFailedReason is the status failed of subnet.
	SubnetsReconciliationFailedReason = "SubnetsReconciliationFailed"
)

const (
	// InternetServicesReadyCondition is the status of internetService.
	InternetServicesReadyCondition clusterv1.ConditionType = "InternetServiceReady"
	// InternetServicesFailedReason is the satus failed of internetService.
	InternetServicesFailedReason = "InternetServiceFailed"
)

const (
	// NatServicesReadyCondition is the status of natService.
	NatServicesReadyCondition clusterv1.ConditionType = "NatServicesReady"
	// NatServicesCreationStartedReason is the status start of natService.
	NatServicesCreationStartedReason = "NatServicesCreationStarted"
	// NatServicesReconciliationFailedReason is the satus failed of natService.
	NatServicesReconciliationFailedReason = "NatServicesReconciliationFailed"
)

const (
	// RouteTablesReadyCondition is the status of routetable.
	RouteTablesReadyCondition clusterv1.ConditionType = "RouteTablesReady"
	// RouteTableReconciliationFailedReason is the status failed of routeTable.
	RouteTableReconciliationFailedReason = "RouteTableReconciliationFailed"
)

const (
	// VMReadyCondition is vm ready condition.
	VMReadyCondition clusterv1.ConditionType = "VmReady"
	// VMNotFoundReason is not found vm.
	VMNotFoundReason = "VmNotFound"
	// VMTerminatedReason is terminated vm.
	VMTerminatedReason = "VmTerminated"
	// VMStoppedReason is stopped vm.
	VMStoppedReason = "VmStopped"
	// VMNotReadyReason is not ready vm.
	VMNotReadyReason = "VmNotReady"
	// VMProvisionStartedReason is not started vm provision.
	VMProvisionStartedReason = "VmProvisionStarted"
	// VMProvisionFailedReason is failed vm.
	VMProvisionFailedReason = "VmProvisionFailed"
	// WaitingForClusterInfrastructureReason is wait cluster infrastructure reason.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"
	// WaitingForBootstrapDataReason wait for bootstrap reason.
	WaitingForBootstrapDataReason = "WaitingForBoostrapData"
)

const (
	// SecurityGroupReadyCondition is the status of  SecurityGroup.
	SecurityGroupReadyCondition clusterv1.ConditionType = "SecurityGroupsReady"
	// SecurityGroupReconciliationFailedReason is the status failed of SecurityGroup.
	SecurityGroupReconciliationFailedReason = "SecurityGroupReconciliationFailed"
)

const (
	// LoadBalancerReadyCondition is the status of LoadBalancer.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	// LoadBalancerFailedReason  is the status failed of LoadBalancer.
	LoadBalancerFailedReason = "LoadBalancerFailed"
)

const (
	// PublicIPSReadyCondition  is the status of PublicIPS.
	PublicIPSReadyCondition clusterv1.ConditionType = "PublicIPSReady"
	// PublicIPSFailedReason is the status failed of PublicIPS.
	PublicIPSFailedReason = "PublicIPSFailed"
)

const (
	// VolumeReadyCondition is the status of volume.
	VolumeReadyCondition clusterv1.ConditionType = "VolumeReady"
	// VolumeReconciliationFailedReason is the status failed of volume.
	VolumeReconciliationFailedReason = "VolumeFailed"
)
