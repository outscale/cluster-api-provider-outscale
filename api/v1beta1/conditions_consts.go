/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

const (
	NetCreatedReason              string                  = "NetCreated"
	NetReadyCondition             clusterv1.ConditionType = "NetReady"
	NetReconciliationFailedReason string                  = "NetReconciliationFailed"
)

const (
	NetPeeringCreatedReason              string                  = "NetPeeringCreated"
	NetPeeringReadyCondition             clusterv1.ConditionType = "NetPeeringReady"
	NetPeeringReconciliationFailedReason string                  = "NetPeeringReconciliationFailed"
)

const (
	SubnetCreatedReason               string                  = "SubnetCreated"
	SubnetsReadyCondition             clusterv1.ConditionType = "SubnetsReady"
	SubnetsReconciliationFailedReason string                  = "SubnetsReconciliationFailed"
)

const (
	InternetServicesCreatedReason  string                  = "InternetServiceCreated"
	InternetServicesReadyCondition clusterv1.ConditionType = "InternetServiceReady"
	InternetServicesFailedReason   string                  = "InternetServiceFailed"
)

const (
	NatServicesCreatedReason              string                  = "NatServicesCreated"
	NatServicesReadyCondition             clusterv1.ConditionType = "NatServicesReady"
	NatServicesReconciliationFailedReason string                  = "NatServicesReconciliationFailed"
)

const (
	RouteTableCreatedReason              string                  = "RouteTableCreated"
	RouteTablesReadyCondition            clusterv1.ConditionType = "RouteTablesReady"
	RouteTableReconciliationFailedReason string                  = "RouteTableReconciliationFailed"
)

const (
	VmReadyCondition                      clusterv1.ConditionType = "VmReady"
	VmNotFoundReason                      string                  = "VmNotFound"
	VmTerminatedReason                    string                  = "VmTerminated"
	VmStoppedReason                       string                  = "VmStopped"
	VmNotReadyReason                      string                  = "VmNotReady"
	VmCreatedReason                       string                  = "VmCreated"
	VmProvisionFailedReason               string                  = "VmProvisionFailed"
	WaitingForClusterInfrastructureReason string                  = "WaitingForClusterInfrastructure"
	WaitingForBootstrapDataReason         string                  = "WaitingForBoostrapData"
)

const (
	SecurityGroupCreatedReason              string                  = "SecurityGroupCreated"
	SecurityGroupReadyCondition             clusterv1.ConditionType = "SecurityGroupsReady"
	SecurityGroupReconciliationFailedReason string                  = "SecurityGroupReconciliationFailed"
)

const (
	LoadBalancerCreatedReason  string                  = "LoadBalancerCreated"
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	LoadBalancerFailedReason   string                  = "LoadBalancerFailed"
)

const (
	VolumeReadyCondition             clusterv1.ConditionType = "VolumeReady"
	VolumeReconciliationFailedReason string                  = "VolumeFailed"
)
