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
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/tele"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

// OscClusterReconciler reconciles a OscCluster object
type OscClusterReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

// getNetSvc retrieve netSvc
func (r *OscClusterReconciler) getNetSvc(ctx context.Context, scope scope.ClusterScope) net.OscNetInterface {
	return net.NewService(ctx, &scope)
}

// getSubnetSvc retrieve subnetSvc
func (r *OscClusterReconciler) getSubnetSvc(ctx context.Context, scope scope.ClusterScope) net.OscSubnetInterface {
	return net.NewService(ctx, &scope)
}

// getInternetServiceSvc retrieve internetServiceSvc
func (r *OscClusterReconciler) getInternetServiceSvc(ctx context.Context, scope scope.ClusterScope) net.OscInternetServiceInterface {
	return net.NewService(ctx, &scope)
}

// getRouteTableSvc retrieve routeTableSvc
func (r *OscClusterReconciler) getRouteTableSvc(ctx context.Context, scope scope.ClusterScope) security.OscRouteTableInterface {
	return security.NewService(ctx, &scope)
}

// getSecurityGroupSvc retrieve securityGroupSvc
func (r *OscClusterReconciler) getSecurityGroupSvc(ctx context.Context, scope scope.ClusterScope) security.OscSecurityGroupInterface {
	return security.NewService(ctx, &scope)
}

// getNatServiceSvc retrieve natServiceSvc
func (r *OscClusterReconciler) getNatServiceSvc(ctx context.Context, scope scope.ClusterScope) net.OscNatServiceInterface {
	return net.NewService(ctx, &scope)
}

// getPublicIpSvc retrieve publicIpSvc
func (r *OscClusterReconciler) getPublicIpSvc(ctx context.Context, scope scope.ClusterScope) security.OscPublicIpInterface {
	return security.NewService(ctx, &scope)
}

// getLoadBalancerSvc retrieve loadBalancerSvc
func (r *OscClusterReconciler) getLoadBalancerSvc(ctx context.Context, scope scope.ClusterScope) service.OscLoadBalancerInterface {
	return service.NewService(ctx, &scope)
}

func (r *OscClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	ctx, _, reconcileDone := tele.StartSpanWithLogger(
		ctx,
		"controllers.OscClusterReconciler.Reconcile",
		tele.KVP("namespace", req.Namespace),
		tele.KVP("name", req.Name),
		tele.KVP("kind", "OscCluster"),
	)
	defer reconcileDone()

	log := ctrl.LoggerFrom(ctx)
	oscCluster := &infrastructurev1beta1.OscCluster{}

	if err := r.Get(ctx, req.NamespacedName, oscCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	cluster, err := util.GetOwnerCluster(ctx, r.Client, oscCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, oscCluster) {
		log.Info("oscCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create scope: %+v", err)
	}
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	osccluster := clusterScope.OscCluster
	if !osccluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	log.Info("Create loadBalancer", "loadBalancerName", loadBalancerSpec.LoadBalancerName)
	return r.reconcile(ctx, clusterScope)
}

// alertDuplicate alert if item is present more than once in array
func alertDuplicate(nameArray []string) error {
	checkMap := make(map[string]bool, 0)
	for _, name := range nameArray {
		if checkMap[name] == true {
			return fmt.Errorf("%s already exist", name)
		} else {
			checkMap[name] = true
		}
	}
	return nil
}

// contains check if item is present in slice
func Contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

// reconcile reconcile the creation of the cluster
func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster
	controllerutil.AddFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	if err := clusterScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	// Check that every element of the cluster spec has the good format (CIDR, Tag, ...)
	netName, err := checkNetFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create net %s for OscCluster %s/%s", err, netName, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	subnetName, err := checkSubnetFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create subnet %s for OscCluster %s/%s", err, subnetName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	internetServiceName, err := checkInternetServiceFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create internetService %s for OscCluster %s/%s", err, internetServiceName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	publicIpName, err := checkPublicIpFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create internetService %s for OscCluster %s/%s", err, publicIpName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	natName, err := checkNatFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create natService %s for OscCluster %s/%s", err, natName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	routeTableName, err := checkRouteTableFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create routeTable %s for OscCluster %s/%s", err, routeTableName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	securityGroupName, err := checkSecurityGroupFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create securityGroup %s for OscCluster %s/%s", err, securityGroupName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	routeName, err := checkRouteFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create route %s for OscCluster %s/%s", err, routeName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	securityGroupRuleName, err := checkSecurityGroupRuleFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create security group rule %s for OscCluster %s/%s", err, securityGroupRuleName, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	reconcileLoadBalancerName, err := checkLoadBalancerFormatParameters(clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create loadBalancer %s for OscCluster %s/%s", err, reconcileLoadBalancerName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	// Check that every element of the cluster spec has a unique tag name
	duplicateResourceRouteTableErr := checkRouteTableOscDuplicateName(clusterScope)
	if duplicateResourceRouteTableErr != nil {
		return reconcile.Result{}, duplicateResourceRouteTableErr
	}

	duplicateResourceSecurityGroupErr := checkSecurityGroupOscDuplicateName(clusterScope)
	if duplicateResourceSecurityGroupErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupErr
	}

	duplicateResourceRouteErr := checkRouteOscDuplicateName(clusterScope)
	if duplicateResourceRouteErr != nil {
		return reconcile.Result{}, duplicateResourceRouteErr
	}

	duplicateResourceSecurityGroupRuleErr := checkSecurityGroupRuleOscDuplicateName(clusterScope)
	if duplicateResourceSecurityGroupRuleErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupRuleErr
	}

	duplicateResourcePublicIpErr := checkPublicIpOscDuplicateName(clusterScope)
	if duplicateResourcePublicIpErr != nil {
		return reconcile.Result{}, duplicateResourcePublicIpErr
	}

	duplicateResourceSubnetErr := checkSubnetOscDuplicateName(clusterScope)
	if duplicateResourceSubnetErr != nil {
		return reconcile.Result{}, duplicateResourceSubnetErr
	}

	// Check that every element of the cluster spec which has other element depencies has the same dependencies tag name

	checkOscAssociatePublicIpErr := checkPublicIpOscAssociateResourceName(clusterScope)
	if checkOscAssociatePublicIpErr != nil {
		return reconcile.Result{}, checkOscAssociatePublicIpErr
	}

	checkOscAssociateRouteTableSubnetErr := checkRouteTableSubnetOscAssociateResourceName(clusterScope)
	if checkOscAssociateRouteTableSubnetErr != nil {
		return reconcile.Result{}, checkOscAssociateRouteTableSubnetErr
	}

	checkOscAssociateNatSubnetErr := checkNatSubnetOscAssociateResourceName(clusterScope)
	if checkOscAssociateNatSubnetErr != nil {
		return reconcile.Result{}, checkOscAssociateNatSubnetErr
	}

	checkOscAssociateLoadBalancerSubnetErr := checkLoadBalancerSubnetOscAssociateResourceName(clusterScope)
	if checkOscAssociateLoadBalancerSubnetErr != nil {
		return reconcile.Result{}, checkOscAssociateLoadBalancerSubnetErr
	}

	checkOscAssociateLoadBalancerSecurityGroupErr := checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope)
	if checkOscAssociateLoadBalancerSecurityGroupErr != nil {
		return reconcile.Result{}, checkOscAssociateLoadBalancerSecurityGroupErr
	}
	clusterScope.Info("Set OscCluster status to not ready")
	clusterScope.SetNotReady()
	// Reconcile each element of the cluster
	ctx, _, netDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.netReconcile")

	netSvc := r.getNetSvc(ctx, *clusterScope)
	reconcileNet, err := reconcileNet(ctx, clusterScope, netSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile net")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NetReadyCondition, infrastructurev1beta1.NetReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		netDone()
		return reconcileNet, err
	}
	netDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NetReadyCondition)

	ctx, _, subnetDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.subnetReconcile")

	subnetSvc := r.getSubnetSvc(ctx, *clusterScope)
	reconcileSubnets, err := reconcileSubnet(ctx, clusterScope, subnetSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile subnet")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SubnetsReadyCondition, infrastructurev1beta1.SubnetsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		subnetDone()
		return reconcileSubnets, err
	}
	subnetDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SubnetsReadyCondition)

	ctx, _, internetServiceDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.internetServiceReconcile")

	internetServiceSvc := r.getInternetServiceSvc(ctx, *clusterScope)
	reconcileInternetService, err := reconcileInternetService(ctx, clusterScope, internetServiceSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile internetService")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.InternetServicesReadyCondition, infrastructurev1beta1.InternetServicesFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		internetServiceDone()
		return reconcileInternetService, err
	}
	internetServiceDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.InternetServicesReadyCondition)

	ctx, _, publicIpDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.publicIpReconcile")

	publicIpSvc := r.getPublicIpSvc(ctx, *clusterScope)
	reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope, publicIpSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile publicIp")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.PublicIpsReadyCondition, infrastructurev1beta1.PublicIpsFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		publicIpDone()
		return reconcilePublicIp, err
	}
	publicIpDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.PublicIpsReadyCondition)

	ctx, _, securityGroupDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.securityGroupReconcile")

	securityGroupSvc := r.getSecurityGroupSvc(ctx, *clusterScope)
	reconcileSecurityGroups, err := reconcileSecurityGroup(ctx, clusterScope, securityGroupSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile securityGroup")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition, infrastructurev1beta1.SecurityGroupReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		securityGroupDone()
		return reconcileSecurityGroups, err
	}
	securityGroupDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition)

	ctx, _, routeTableDone := tele.StartSpanWithLogger(ctx, "controller.OscClusterControllers.routeTableReconcile")

	routeTableSvc := r.getRouteTableSvc(ctx, *clusterScope)
	reconcileRouteTables, err := reconcileRouteTable(ctx, clusterScope, routeTableSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile routeTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		routeTableDone()
		return reconcileRouteTables, err
	}
	routeTableDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	ctx, _, natServiceDone := tele.StartSpanWithLogger(ctx, "controller.OscClusterControllers.natServiceReconcile")

	natServiceSvc := r.getNatServiceSvc(ctx, *clusterScope)
	reconcileNatService, err := reconcileNatService(ctx, clusterScope, natServiceSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile natservice")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NatServicesReadyCondition, infrastructurev1beta1.NatServicesReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		natServiceDone()
		return reconcileNatService, nil
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NatServicesReadyCondition)

	reconcileNatRouteTable, err := reconcileRouteTable(ctx, clusterScope, routeTableSvc)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile NatRouteTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		natServiceDone()
		return reconcileNatRouteTable, err
	}
	natServiceDone()
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	ctx, _, loadBalancerDone := tele.StartSpanWithLogger(ctx, "controllers.OscClusterControllers.loadBalancerReconcile")

	loadBalancerSvc := r.getLoadBalancerSvc(ctx, *clusterScope)
	_, err = reconcileLoadBalancer(ctx, clusterScope, loadBalancerSvc)
	loadBalancerDone()
	clusterScope.Info("Set OscCluster status to ready")
	clusterScope.SetReady()
	return reconcile.Result{}, nil
}

// reconcileDelete reconcile the deletion of the cluster
func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster

	// reconcile deletion of each element of the cluster

	machines, _, err := clusterScope.ListMachines(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list machines for OscCluster %s/%s: %+v", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	if len(machines) > 0 {
		names := make([]string, len(machines))
		for i, m := range machines {
			names[i] = fmt.Sprintf("machine/%s", m.Name)
		}
		nameMachineList := strings.Join(names, ", ")
		clusterScope.Info("Machine are still running, postpone oscCluster deletion", "nameMachineList", nameMachineList)
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	ctx, _, loadBalancerDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileLoadBalancerDelete")

	loadBalancerSvc := r.getLoadBalancerSvc(ctx, *clusterScope)
	reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, loadBalancerSvc)
	if err != nil {
		loadBalancerDone()
		return reconcileDeleteLoadBalancer, err
	}
	loadBalancerDone()

	ctx, _, natServiceDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileNatServiceDelete")

	natServiceSvc := r.getNatServiceSvc(ctx, *clusterScope)
	reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope, natServiceSvc)
	if err != nil {
		natServiceDone()
		return reconcileDeleteNatService, err
	}
	natServiceDone()

	ctx, _, publicIpDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcilePublicIpDelete")
	publicIpSvc := r.getPublicIpSvc(ctx, *clusterScope)
	reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope, publicIpSvc)
	if err != nil {
		publicIpDone()
		return reconcileDeletePublicIp, err
	}
	publicIpDone()

	ctx, _, routeTableDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileRouteTableDelete")

	routeTableSvc := r.getRouteTableSvc(ctx, *clusterScope)
	reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, routeTableSvc)
	if err != nil {
		routeTableDone()
		return reconcileDeleteRouteTable, err
	}
	routeTableDone()

	ctx, _, securityGroupDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileSecurityGroupDelete")

	securityGroupSvc := r.getSecurityGroupSvc(ctx, *clusterScope)
	reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, securityGroupSvc)
	if err != nil {
		securityGroupDone()
		return reconcileDeleteSecurityGroup, err
	}
	securityGroupDone()

	ctx, _, internetServiceGroupDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileInternetServiceGroupDelete")

	internetServiceSvc := r.getInternetServiceSvc(ctx, *clusterScope)
	reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope, internetServiceSvc)
	if err != nil {
		internetServiceGroupDone()
		return reconcileDeleteInternetService, err
	}
	internetServiceGroupDone()

	ctx, _, subnetDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileSubnetDelete")

	subnetSvc := r.getSubnetSvc(ctx, *clusterScope)
	reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope, subnetSvc)
	if err != nil {
		subnetDone()
		return reconcileDeleteSubnet, err
	}
	subnetDone()

	ctx, _, netDone := tele.StartSpanWithLogger(ctx, "controllers.OscMachineReconciler.reconcileNetDelete")

	netSvc := r.getNetSvc(ctx, *clusterScope)
	reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope, netSvc)
	if err != nil {
		netDone()
		return reconcileDeleteNet, err
	}
	netDone()

	controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscCluster{}).
		Complete(r)
}
