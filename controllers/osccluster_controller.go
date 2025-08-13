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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	predicates "sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const OscClusterFinalizer = "oscclusters.infrastructure.cluster.x-k8s.io"

// OscClusterReconciler reconciles a OscCluster object
type OscClusterReconciler struct {
	client.Client
	Tracker          *ClusterResourceTracker
	Cloud            services.Servicer
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

	oscCluster := &infrastructurev1beta1.OscCluster{}

	if err := r.Get(ctx, req.NamespacedName, oscCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(3).Info("Cluster was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	cluster, err := util.GetOwnerCluster(ctx, r.Client, oscCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.V(3).Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, oscCluster) {
		log.V(3).Info("oscCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		OscClient:  r.Cloud.OscClient(),
		Cluster:    cluster,
		OscCluster: oscCluster,
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

	errs := infrastructurev1beta1.ValidateOscClusterSpec(osccluster.Spec)
	if len(errs) > 0 {
		return reconcile.Result{}, errs.ToAggregate()
	}

	// Reconcile each element of the cluster
	_, err := r.reconcileNet(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NetReadyCondition, infrastructurev1beta1.NetReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile net: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NetReadyCondition)

	_, err = r.reconcileSubnets(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SubnetsReadyCondition, infrastructurev1beta1.SubnetsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile subnets: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SubnetsReadyCondition)

	_, err = r.reconcileInternetService(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.InternetServicesReadyCondition, infrastructurev1beta1.InternetServicesFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile internetService: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.InternetServicesReadyCondition)

	// Add public route table to mark public subnet as public & enable NAT creation
	_, err = r.reconcileRouteTable(ctx, clusterScope, infrastructurev1beta1.RoleNat)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile public routeTables: %w", err)
	}

	_, err = r.reconcileNatService(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NatServicesReadyCondition, infrastructurev1beta1.NatServicesReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile natServices: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NatServicesReadyCondition)

	// Add all other route tables, whose destinations are the NAT services previously created.
	_, err = r.reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile routeTables: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	// Security groups need NAT services to allow NAT to connect to LB.
	_, err = r.reconcileSecurityGroup(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition, infrastructurev1beta1.SecurityGroupReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile securityGroups: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition)

	_, err = r.reconcileLoadBalancer(ctx, clusterScope)
	if err != nil {
		conditions.MarkFalse(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition, infrastructurev1beta1.LoadBalancerFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconcile loadBalancer: %w", err)
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition)

	if clusterScope.OscCluster.Spec.Network.Bastion.Enable {
		_, err := r.reconcileBastion(ctx, clusterScope)
		if err != nil {
			conditions.MarkFalse(osccluster, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.VmNotReadyReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcile.Result{}, fmt.Errorf("reconcile bastion: %w", err)
		}
		conditions.MarkTrue(osccluster, infrastructurev1beta1.VmReadyCondition)
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

	if clusterScope.OscCluster.Spec.Network.Bastion.Enable {
		_, err := r.reconcileDeleteBastion(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("reconcile delete bastion: %w", err)
		}
	}
	_, err = r.reconcileDeleteLoadBalancer(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete loadBalancer: %w", err)
	}

	_, err = r.reconcileDeleteNatService(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete natServices: %w", err)
	}

	_, err = r.reconcileDeletePublicIp(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete publicIPs: %w", err)
	}
	_, err = r.reconcileDeleteRouteTable(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete routeTables: %w", err)
	}

	_, err = r.reconcileDeleteSecurityGroup(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete securityGroups: %w", err)
	}

	_, err = r.reconcileDeleteInternetService(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile delete internetServices: %w", err)
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
func (r *OscClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscCluster{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetScheme(), ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}
