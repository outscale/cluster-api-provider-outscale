/*
Copyright 2022.

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
	"time"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	//      "k8s.io/apimachinery/pkg/runtime"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/util/reconciler"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OscCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OscClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()
	log := ctrl.LoggerFrom(ctx)
	oscCluster := &infrastructurev1beta1.OscCluster{}

	log.Info("Please WAIT !!!!")

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

// CheckAssociate return if the resourcename is an item of ResourceNames
func CheckAssociate(resourceName string, ResourceNames []string) bool {
	for i := 0; i < len(ResourceNames); i++ {
		if ResourceNames[i] == resourceName {
			return true
		}
	}
	return false
}


// AlertDuplicate alert if item is present more than once in array
func AlertDuplicate(nameArray []string) error {
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

// Contains check if item is present in slice
func Contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster
	controllerutil.AddFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	if err := clusterScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}
        netName, err := CheckNetFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create net %s for OscCluster %s/%s", err, netName, osccluster.Namespace, osccluster.Name)
        }
        subnetName, err := CheckSubnetFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create subnet %s for OscCluster %s/%s", err, subnetName, osccluster.Namespace, osccluster.Name)
        }

        internetServiceName, err := CheckInternetServiceFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create internetService %s for OscCluster %s/%s", err, internetServiceName, osccluster.Namespace, osccluster.Name)
        }

        publicIpName, err := CheckPublicIpFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create internetService %s for OscCluster %s/%s", err, publicIpName, osccluster.Namespace, osccluster.Name)
        }

	natName, err := CheckNatFormatParameters(clusterScope)
	if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create natService %s for OscCluster %s/%s", err, natName, osccluster.Namespace, osccluster.Name)
	}
	
        routeTableName, err := CheckRouteTableFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create routeTable %s for OscCluster %s/%s", err, routeTableName, osccluster.Namespace, osccluster.Name)
        }

        securityGroupName, err := CheckSecurityGroupFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create securityGroup %s for OscCluster %s/%s", err, securityGroupName, osccluster.Namespace, osccluster.Name)
        }

        routeName, err := CheckRouteFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create route %s for OscCluster %s/%s", err, routeName, osccluster.Namespace, osccluster.Name)
        }

        securityGroupRuleName, err := CheckSecurityGroupRuleFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create security group rule %s for OscCluster %s/%s", err, securityGroupRuleName, osccluster.Namespace, osccluster.Name)
        }
        reconcileLoadBalancerName, err := CheckLoadBalancerFormatParameters(clusterScope)
        if err != nil {
                return reconcile.Result{}, fmt.Errorf("%w: Can not create loadBalancer %s for OscCluster %s/%s", err, reconcileLoadBalancerName, osccluster.Namespace, osccluster.Name)
        }

	duplicateResourceRouteTableErr := CheckRouteTableOscDuplicateName(clusterScope)
	if duplicateResourceRouteTableErr != nil {
		return reconcile.Result{}, duplicateResourceRouteTableErr
	}

	duplicateResourceSecurityGroupErr := CheckSecurityGroupOscDuplicateName(clusterScope)
	if duplicateResourceSecurityGroupErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupErr
	}

	duplicateResourceRouteErr := CheckRouteOscDuplicateName(clusterScope)
	if duplicateResourceRouteErr != nil {
		return reconcile.Result{}, duplicateResourceRouteErr
	}

	duplicateResourceSecurityGroupRuleErr := CheckSecurityGroupRuleOscDuplicateName(clusterScope)
	if duplicateResourceSecurityGroupRuleErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupRuleErr
	}

	duplicateResourcePublicIpErr := CheckPublicIpOscDuplicateName(clusterScope)
	if duplicateResourcePublicIpErr != nil {
		return reconcile.Result{}, duplicateResourcePublicIpErr
	}

	duplicateResourceSubnetErr := CheckSubnetOscDuplicateName(clusterScope)
	if duplicateResourceSubnetErr != nil {
		return reconcile.Result{}, duplicateResourceSubnetErr
	}

	CheckOscAssociatePublicIpErr := CheckPublicIpOscAssociateResourceName(clusterScope)
	if CheckOscAssociatePublicIpErr != nil {
		return reconcile.Result{}, CheckOscAssociatePublicIpErr
	}

	CheckOscAssociateRouteTableSubnetErr := CheckRouteTableSubnetOscAssociateResourceName(clusterScope)
	if CheckOscAssociateRouteTableSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateRouteTableSubnetErr
	}

	CheckOscAssociateNatSubnetErr := CheckNatSubnetOscAssociateResourceName(clusterScope)
	if CheckOscAssociateNatSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateNatSubnetErr
	}

	CheckOscAssociateLoadBalancerSubnetErr := CheckLoadBalancerSubnetOscAssociateResourceName(clusterScope)
	if CheckOscAssociateLoadBalancerSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateLoadBalancerSubnetErr
	}

	CheckOscAssociateLoadBalancerSecurityGroupErr := CheckLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope)
	if CheckOscAssociateLoadBalancerSecurityGroupErr != nil {
		return reconcile.Result{}, CheckOscAssociateLoadBalancerSecurityGroupErr
	}

	reconcileNet, err := reconcileNet(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile net")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NetReadyCondition, infrastructurev1beta1.NetReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNet, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NetReadyCondition)

	reconcileSubnets, err := reconcileSubnet(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile subnet")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SubnetsReadyCondition, infrastructurev1beta1.SubnetsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileSubnets, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SubnetsReadyCondition)

	reconcileInternetService, err := reconcileInternetService(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile internetService")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.InternetServicesReadyCondition, infrastructurev1beta1.InternetServicesFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileInternetService, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.InternetServicesReadyCondition)

	reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile publicIp")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.PublicIpsReadyCondition, infrastructurev1beta1.PublicIpsFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcilePublicIp, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.PublicIpsReadyCondition)

	reconcileRouteTables, err := reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile routeTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileRouteTables, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	reconcileSecurityGroups, err := reconcileSecurityGroup(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile securityGroup")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition, infrastructurev1beta1.SecurityGroupReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileSecurityGroups, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition)

	reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile load balancer")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition, infrastructurev1beta1.LoadBalancerFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileLoadBalancer, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition)

	reconcileNatService, err := reconcileNatService(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile natservice")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NatServicesReadyCondition, infrastructurev1beta1.NatServicesReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNatService, nil
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NatServicesReadyCondition)

	reconcileNatRouteTable, err := reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile NatRouteTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNatRouteTable, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	clusterScope.Info("Set OscCluster status to ready")
	clusterScope.SetReady()
	return reconcile.Result{}, nil
}


func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster

	reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteLoadBalancer, err
	}

	reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteNatService, err
	}

	reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope)
	if err != nil {
		return reconcileDeletePublicIp, err
	}

	reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteRouteTable, err
	}

	reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteSecurityGroup, err
	}

	reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteInternetService, err
	}

	reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteSubnet, err
	}
	reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteNet, err
	}

	controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscCluster{}).
		Complete(r)
}
