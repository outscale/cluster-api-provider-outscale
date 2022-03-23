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
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"time"
        "os"
	//      "k8s.io/apimachinery/pkg/runtime"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/util/reconciler"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service" 
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
	log.Info("Still WAIT !!!!")
        log.Info("Create info", "env", os.Environ())

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
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
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
        loadBalancerSpec := clusterScope.LoadBalancer()
        loadBalancerSpec.SetDefaultValue()
        log.Info("Create loadBalancer", "loadBalancerName", loadBalancerSpec.LoadBalancerName, "SubregionName", loadBalancerSpec.SubregionName)
	return r.reconcile(ctx, clusterScope)
}


func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
    clusterScope.Info("Reconcile OscCluster")
    osccluster := clusterScope.OscCluster
    servicesvc := service.NewService(ctx, clusterScope)
    clusterScope.Info("Get Service", "service", servicesvc)
    loadBalancerSpec := clusterScope.LoadBalancer()
    loadBalancerSpec.SetDefaultValue()
    loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, err
    }
    if loadbalancer == nil {
    	_, err := servicesvc.CreateLoadBalancer(loadBalancerSpec)
	if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    	}
        _, err = servicesvc.ConfigureHealthCheck(loadBalancerSpec)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not configure healthcheck for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        } 
    }
    controllerutil.AddFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
    return reconcile.Result{}, nil
}

func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
    clusterScope.Info("Reconcile OscCluster")
    osccluster := clusterScope.OscCluster
    servicesvc := service.NewService(ctx, clusterScope)
    clusterScope.Info("Get Service", "service", servicesvc)
    loadBalancerSpec := clusterScope.LoadBalancer()
    loadBalancerSpec.SetDefaultValue()
    loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, err
    }
    if loadbalancer == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = servicesvc.DeleteLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, errors.Wrapf(err, "Can not delete load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
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
