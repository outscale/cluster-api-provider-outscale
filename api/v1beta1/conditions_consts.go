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
	NetCreatedReason              string                  = "NetCreated"
	NetReadyCondition             clusterv1.ConditionType = "NetReady"
	NetReconciliationFailedReason string                  = "NetReconciliationFailed"
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
