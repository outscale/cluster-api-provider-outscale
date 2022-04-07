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
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	LoadBalancerFailedReason                           = "LoadBalancerFailed"
)

const (
	PublicIpsReadyCondition clusterv1.ConditionType = "PublicIpsReady"
	PublicIpsFailedReason                           = "PublicIpsFailed"
)
