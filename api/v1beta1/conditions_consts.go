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
	NetReadyCondition             clusterv1.ConditionType = "NetReady"
	NetCreationStartedReason                              = "NetCreationStarted"
	NetReconciliationFailedReason                         = "NetReconciliationFailed"
)

const (
	SubnetsReadyCondition             clusterv1.ConditionType = "SubnetsReady"
	SubnetsReconciliationFailedReason                         = "SubnetsReconciliationFailed"
)

const (
	InternetServicesReadyCondition clusterv1.ConditionType = "InternetServiceReady"
	InternetServicesFailedReason                           = "InternetServiceFailed"
)

const (
	NatServicesReadyCondition             clusterv1.ConditionType = "NatServicesReady"
	NatServicesCreationStartedReason                              = "NatServicesCreationStarted"
	NatServicesReconciliationFailedReason                         = "NatServicesReconciliationFailed"
)

const (
	RouteTablesReadyCondition            clusterv1.ConditionType = "RouteTablesReady"
	RouteTableReconciliationFailedReason                         = "RouteTableReconciliationFailed"
)

const (
	VmReadyCondition                      clusterv1.ConditionType = "VmReady"
	VmNotFoundReason                                              = "VmNotFound"
	VmTerminatedReason                                            = "VmTerminated"
	VmStoppedReason                                               = "VmStopped"
	VmNotReadyReason                                              = "VmNotReady"
	VmProvisionStartedReason                                      = "VmProvisionStarted"
	VmProvisionFailedReason                                       = "VmProvisionFailed"
	WaitingForClusterInfrastructureReason                         = "WaitingForClusterInfrastructure"
	WaitingForBootstrapDataReason                                 = "WaitingForBoostrapData"
)

const (
	SecurityGroupReadyCondition             clusterv1.ConditionType = "SecurityGroupsReady"
	SecurityGroupReconciliationFailedReason                         = "SecurityGroupReconciliationFailed"
)

const (
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	LoadBalancerFailedReason                           = "LoadBalancerFailed"
)

const (
	PublicIpsReadyCondition clusterv1.ConditionType = "PublicIpsReady"
	PublicIpsFailedReason                           = "PublicIpsFailed"
)

const (
	VolumeReadyCondition             clusterv1.ConditionType = "VolumeReady"
	VolumeReconciliationFailedReason                         = "VolumeFailed"
)
