/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	predicates "sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const OscClusterFinalizer = "oscclusters.infrastructure.cluster.x-k8s.io"

// OscClusterReconciler reconciles a OscCluster object
type OscClusterReconciler struct {
	Client           client.Client
	Tracker          *ClusterResourceTracker
	Cloud            services.Servicer
	Metadata         services.Metadata
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
	WatchFilterValue string
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

func (r *OscClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()
	log := ctrl.LoggerFrom(ctx)

	oscCluster := &infrastructurev1beta2.OscCluster{}

	err := r.Client.Get(ctx, req.NamespacedName, oscCluster)
	switch {
	case apierrors.IsNotFound(err):
		log.V(3).Info("OscCluster was not found; aborting")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, err
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, oscCluster.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err):
		log.V(3).Info("Cluster has been deleted; aborting")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, err
	case cluster == nil:
		log.V(3).Info("Cluster Controller has not yet set OwnerRef; aborting")
		return reconcile.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, oscCluster) {
		log.V(3).Info("OscCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	t, err := getTenant(ctx, r.Client, r.Cloud, oscCluster)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to fetch tenant: %w", err)
	}
	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Cluster:    cluster,
		OscCluster: oscCluster,
		Tenant:     t,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}
	defer func() {
		if err := clusterScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()
	osccluster := clusterScope.OscCluster
	if !osccluster.DeletionTimestamp.IsZero() {
		return HandleFatalError(r.reconcileDelete(ctx, clusterScope))
	}
	return HandleFatalError(r.reconcile(ctx, clusterScope))
}

// reconcile reconcile the creation of the cluster
func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	osccluster := clusterScope.OscCluster
	if controllerutil.AddFinalizer(osccluster, OscClusterFinalizer) {
		// Register the finalizer immediately to avoid orphaning IaaS resources on delete
		if err := clusterScope.PatchObject(ctx); err != nil {
			return reconcile.Result{}, err
		}
	}
	clusterScope.EnsureExplicitUID()

	// Reconcile each element of the cluster
	res := r.reconcileNet(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
		MarkFalseCondition(osccluster, infrastructurev1beta2.NetReadyCondition, infrastructurev1beta2.NetReconciliationFailedReason, err)
	}, func() {
		conditions.MarkTrue(osccluster, infrastructurev1beta2.NetReadyCondition)
	}, "net"); stop {
		return res, err
	}

	res = r.reconcileSubnets(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
		MarkFalseCondition(osccluster, infrastructurev1beta2.SubnetsReadyCondition, infrastructurev1beta2.SubnetsReconciliationFailedReason, err)
	}, func() {
		conditions.MarkTrue(osccluster, infrastructurev1beta2.SubnetsReadyCondition)
	}, "subnet"); stop {
		return res, err
	}

	if !clusterScope.GetSpec().Disable.Internet {
		res = r.reconcileInternetService(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.InternetServicesReadyCondition, infrastructurev1beta2.InternetServicesFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.InternetServicesReadyCondition)
		}, "subnet"); stop {
			return res, err
		}

		// Add public route table to mark public subnet as public & enable NAT creation
		res = r.reconcileRouteTable(ctx, clusterScope, infrastructurev1beta2.RoleNat)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.RouteTablesReadyCondition, infrastructurev1beta2.RouteTableReconciliationFailedReason, err)
		}, nil, "public routeTables"); stop {
			return res, err
		}

		res = r.reconcileNatService(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.NatServicesReadyCondition, infrastructurev1beta2.NatServicesReconciliationFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.NatServicesReadyCondition)
		}, "natServices"); stop {
			return res, err
		}
	}

	// Add all other route tables, whose destinations are the NAT services previously created.
	res = r.reconcileRouteTable(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
		MarkFalseCondition(osccluster, infrastructurev1beta2.RouteTablesReadyCondition, infrastructurev1beta2.RouteTableReconciliationFailedReason, err)
	}, func() {
		conditions.MarkTrue(osccluster, infrastructurev1beta2.RouteTablesReadyCondition)
	}, "routeTables"); stop {
		return res, err
	}

	if clusterScope.GetSpec().NetPeering.Enable || len(clusterScope.GetSpec().Nets) > 1 {
		res = r.reconcileNetPeering(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.NetPeeringReadyCondition, infrastructurev1beta2.NetPeeringReconciliationFailedReason, err)
		}, func() {
		}, "netPeering"); stop {
			return res, err
		}

		res = r.reconcileNetPeeringRoutes(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.NetPeeringReadyCondition, infrastructurev1beta2.NetPeeringReconciliationFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.NetPeeringReadyCondition)
		}, "netPeering routes"); stop {
			return res, err
		}
	}

	if len(clusterScope.GetSpec().NetAccessPoints) > 0 {
		res = r.reconcileNetAccessPoints(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.NetAccessPointsReadyCondition, infrastructurev1beta2.NetAccessPointsReconciliationFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.NetAccessPointsReadyCondition)
		}, "netPeering routes"); stop {
			return res, err
		}
	}

	// Security groups need NAT services to allow NAT to connect to LB.
	res = r.reconcileSecurityGroup(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
		MarkFalseCondition(osccluster, infrastructurev1beta2.SecurityGroupReadyCondition, infrastructurev1beta2.SecurityGroupReconciliationFailedReason, err)
	}, func() {
		conditions.MarkTrue(osccluster, infrastructurev1beta2.SecurityGroupReadyCondition)
	}, "securityGroups"); stop {
		return res, err
	}

	if !clusterScope.GetSpec().Disable.Loadbalancer {
		res = r.reconcileLoadBalancer(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.LoadBalancerReadyCondition, infrastructurev1beta2.LoadBalancerFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.LoadBalancerReadyCondition)
		}, "loadBalancer"); stop {
			return res, err
		}
	}

	if clusterScope.GetSpec().Keypair != nil {
		res := r.reconcileKeypair(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.KeypairReadyCondition, infrastructurev1beta2.KeypairFailedReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.KeypairReadyCondition)
		}, "keypair"); stop {
			return res, err
		}
	}

	log.V(2).Info("OscCluster is ready")
	clusterScope.SetReady()

	if clusterScope.GetSpec().Bastion.Enable {
		res := r.reconcileBastion(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, func(err error) {
			MarkFalseCondition(osccluster, infrastructurev1beta2.VmReadyCondition, infrastructurev1beta2.VmNotReadyReason, err)
		}, func() {
			conditions.MarkTrue(osccluster, infrastructurev1beta2.VmReadyCondition)
		}, "bastion"); stop {
			return res, err
		}
	}

	return reconcile.Result{}, nil
}

// reconcileDelete reconcile the deletion of the cluster
func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Cluster needs to be deleted")
	osccluster := clusterScope.OscCluster

	// reconcile deletion of each element of the cluster

	machines, _, err := clusterScope.ListMachines(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list machines: %w", err)
	}
	if len(machines) > 0 {
		names := make([]string, len(machines))
		for i, m := range machines {
			names[i] = "machine/" + m.Name
		}
		nameMachineList := strings.Join(names, ", ")
		log.V(3).Info("Machines are still running; postpone oscCluster deletion", "nameMachineList", nameMachineList)
		return reconcile.Result{RequeueAfter: time.Minute}, nil
	}

	if clusterScope.GetSpec().Bastion.Enable {
		res := r.reconcileDeleteBastion(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "bastion"); stop {
			return res, err
		}
	}

	if clusterScope.GetSpec().Keypair != nil {
		res := r.reconcileDeleteKeypair(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "keypair"); stop {
			return res, err
		}
	}

	if !clusterScope.GetSpec().Disable.Loadbalancer {
		res := r.reconcileDeleteLoadBalancer(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "loadBalancer"); stop {
			return res, err
		}
	}

	if !clusterScope.GetSpec().Disable.Internet {
		res := r.reconcileDeleteNatService(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "natServices"); stop {
			return res, err
		}
		res = r.reconcileDeletePublicIp(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "publicIPs"); stop {
			return res, err
		}
	}

	res := r.reconcileDeleteNetAccessPoints(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "netAccessPoints"); stop {
		return res, err
	}

	if clusterScope.GetSpec().NetPeering.Enable || len(clusterScope.GetSpec().Nets) > 1 {
		res := r.reconcileDeleteNetPeeringRoutes(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "netPeering routes"); stop {
			return res, err
		}
		res = r.reconcileDeleteNetPeering(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "netPeerings"); stop {
			return res, err
		}
	}
	res = r.reconcileDeleteRouteTable(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "routeTables"); stop {
		return res, err
	}

	res = r.reconcileDeleteSecurityGroup(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "routeTables"); stop {
		return res, err
	}
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete securityGroups: %w", err)
	}
	if !clusterScope.GetSpec().Disable.Internet {
		res = r.reconcileDeleteInternetService(ctx, clusterScope)
		if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "internetServices"); stop {
			return res, err
		}
	}
	res = r.reconcileDeleteSubnets(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "subnets"); stop {
		return res, err
	}

	res = r.reconcileDeleteNet(ctx, clusterScope)
	if stop, res, err := res.CheckResultAndConditions(osccluster, nil, nil, "nets"); stop {
		return res, err
	}
	controllerutil.RemoveFinalizer(osccluster, OscClusterFinalizer)
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrastructurev1beta2.OscCluster{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetScheme(), ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}
