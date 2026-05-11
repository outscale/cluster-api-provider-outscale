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
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/conditions/deprecated/v1beta1"
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
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
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
		return ctrl.Result{}, nil
	case err != nil:
		return ctrl.Result{}, err
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
		return r.reconcileDelete(ctx, clusterScope)
	}
	return r.reconcile(ctx, clusterScope)
}

// reconcile reconcile the creation of the cluster
func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	osccluster := clusterScope.OscCluster
	controllerutil.AddFinalizer(osccluster, OscClusterFinalizer)
	clusterScope.EnsureExplicitUID()

	// Reconcile each element of the cluster
	_, err := r.reconcileNet(ctx, clusterScope)
	if err != nil {
		v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.NetReadyCondition, infrastructurev1beta2.NetReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile net: %w", err)
	}
	v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.NetReadyCondition)

	_, err = r.reconcileSubnets(ctx, clusterScope)
	if err != nil {
		v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.SubnetsReadyCondition, infrastructurev1beta2.SubnetsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile subnets: %w", err)
	}
	v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.SubnetsReadyCondition)

	if !clusterScope.IsInternetDisabled() {
		_, err = r.reconcileInternetService(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.InternetServicesReadyCondition, infrastructurev1beta2.InternetServicesFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile internetService: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.InternetServicesReadyCondition)

		// Add public route table to mark public subnet as public & enable NAT creation
		_, err = r.reconcileRouteTable(ctx, clusterScope, infrastructurev1beta2.RoleNat)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.RouteTablesReadyCondition, infrastructurev1beta2.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile public routeTables: %w", err)
		}

		_, err = r.reconcileNatService(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.NatServicesReadyCondition, infrastructurev1beta2.NatServicesReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile natServices: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.NatServicesReadyCondition)
	}

	// Add all other route tables, whose destinations are the NAT services previously created.
	_, err = r.reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.RouteTablesReadyCondition, infrastructurev1beta2.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile routeTables: %w", err)
	}
	v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.RouteTablesReadyCondition)

	if clusterScope.GetSpec().NetPeering.Enable {
		_, err = r.reconcileNetPeering(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.NetPeeringReadyCondition, infrastructurev1beta2.NetPeeringReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile netPeering: %w", err)
		}
		_, err = r.reconcileNetPeeringRoutes(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.NetPeeringReadyCondition, infrastructurev1beta2.NetPeeringReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile netPeering: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.NetPeeringReadyCondition)
	}

	if len(clusterScope.GetSpec().NetAccessPoints) > 0 {
		_, err = r.reconcileNetAccessPoints(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.NetAccessPointsReadyCondition, infrastructurev1beta2.NetAccessPointsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile netAccessPoints: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.NetAccessPointsReadyCondition)
	}

	// Security groups need NAT services to allow NAT to connect to LB.
	_, err = r.reconcileSecurityGroup(ctx, clusterScope)
	if err != nil {
		v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.SecurityGroupReadyCondition, infrastructurev1beta2.SecurityGroupReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile securityGroups: %w", err)
	}
	v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.SecurityGroupReadyCondition)

	if !clusterScope.IsLBDisabled() {
		_, err = r.reconcileLoadBalancer(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.LoadBalancerReadyCondition, infrastructurev1beta2.LoadBalancerFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile loadBalancer: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.LoadBalancerReadyCondition)
	}

	if clusterScope.GetSpec().Bastion.Enable {
		_, err := r.reconcileBastion(ctx, clusterScope)
		if err != nil {
			v1beta1conditions.MarkFalse(osccluster, infrastructurev1beta2.VmReadyCondition, infrastructurev1beta2.VmNotReadyReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile bastion: %w", err)
		}
		v1beta1conditions.MarkTrue(osccluster, infrastructurev1beta2.VmReadyCondition)
	}

	log.V(2).Info("OscCluster is ready")
	clusterScope.SetReady()
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
		_, err := r.reconcileDeleteBastion(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete bastion: %w", err)
		}
	}

	if !clusterScope.IsLBDisabled() {
		_, err = r.reconcileDeleteLoadBalancer(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete loadBalancer: %w", err)
		}
	}

	if !clusterScope.IsInternetDisabled() {
		_, err = r.reconcileDeleteNatService(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete natServices: %w", err)
		}
		_, err = r.reconcileDeletePublicIp(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete publicIPs: %w", err)
		}
	}

	_, err = r.reconcileDeleteNetAccessPoints(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete netAccessPoints: %w", err)
	}

	if clusterScope.GetSpec().NetPeering.Enable {
		_, err = r.reconcileDeleteNetPeeringRoutes(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete netPeering routes: %w", err)
		}
		_, err = r.reconcileDeleteNetPeering(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete netPeering: %w", err)
		}
	}
	_, err = r.reconcileDeleteRouteTable(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete routeTables: %w", err)
	}

	_, err = r.reconcileDeleteSecurityGroup(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete securityGroups: %w", err)
	}
	if !clusterScope.IsInternetDisabled() {
		_, err = r.reconcileDeleteInternetService(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete internetServices: %w", err)
		}
	}
	_, err = r.reconcileDeleteSubnets(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete subnets: %w", err)
	}

	_, err = r.reconcileDeleteNet(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete net: %w", err)
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
