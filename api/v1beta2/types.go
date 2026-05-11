package v1beta2

type OscRole string

const (
	RoleControlPlane    OscRole = "controlplane"
	RoleWorker          OscRole = "worker"
	RoleLoadBalancer    OscRole = "loadbalancer"
	RoleBastion         OscRole = "bastion"
	RoleNat             OscRole = "nat"
	RoleService         OscRole = "service"
	RoleInternalService OscRole = "service.internal"
)

// +kubebuilder:validation:Enum:=bastion;net;netPeering;netPeering/routes;subnet;internetService;netAccessPoint;natService;routeTable;securityGroup;loadbalancer;vm;*
type Reconciler string

const (
	ReconcilerBastion          Reconciler = "bastion"
	ReconcilerNet              Reconciler = "net"
	ReconcilerNetPeering       Reconciler = "netPeering"
	ReconcilerNetPeeringRoutes Reconciler = "netPeering/routes"
	ReconcilerSubnet           Reconciler = "subnet"
	ReconcilerInternetService  Reconciler = "internetService"
	ReconcilerNetAccessPoint   Reconciler = "netAccessPoint"
	ReconcilerNatService       Reconciler = "natService"
	ReconcilerRouteTable       Reconciler = "routeTable"
	ReconcilerSecurityGroup    Reconciler = "securityGroup"
	ReconcilerLoadbalancer     Reconciler = "loadbalancer"

	ReconcilerVm Reconciler = "vm"

	ReconcilerAll Reconciler = "*"
)

type OscReconcilerGeneration map[Reconciler]int64

// +kubebuilder:validation:Enum:=onChange;always;random
type ReconciliationMode string

const (
	ReconciliationModeOnChange ReconciliationMode = "onChange"
	ReconciliationModeAlways   ReconciliationMode = "always"
	ReconciliationModeRandom   ReconciliationMode = "random"
)

type OscReconciliationRule struct {
	// The list of items this rule applies to (bastion, net, netPeering, netPeering/routes, subnet, internetService, netAccessPoint, natService, routeTable, securityGroup, loadbalancer, vm or * for all)
	AppliesTo []Reconciler `json:"appliesTo,omitempty"`
	// The mode of reconciliation: onChange (only when the spec change, default), always, random (onChange + randomPercent% chance)
	Mode ReconciliationMode `json:"mode,omitempty"`
	// The chance (in percent, 1-100) of a reconcilation happening when no change have been detected (when mode=random)
	// +optional
	ReconciliationChance int `json:"reconciliationChance,omitempty"`
}
