package controllers

import (
	"context"
	"time"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getVmSvc retrieve vmSvc
func (r *OscMachineTemplateReconciler) getVmSvc(ctx context.Context, scope scope.ClusterScope) compute.OscVmInterface {
	return compute.NewService(ctx, &scope)
}

type OscMachineTemplateReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates/status,verbs=get;update;patch

// Reconcile manages the lifecycle of an OscMachineTemplate object.
func (r *OscMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx).WithValues("oscmachinetemplate", req.NamespacedName)
	log.Info("Reconcile OscMachineTemplate")

	machineTemplate := &infrastructurev1beta1.OscMachineTemplate{}
	if err := r.Get(ctx, req.NamespacedName, machineTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	machineTemplateScope, err := scope.NewMachineTemplateScope(scope.MachineTemplateScopeParams{
		Client:             r.Client,
		Logger:             log,
		OscMachineTemplate: machineTemplate,
	})
	clusterName := machineTemplateScope.GetClusterName()

	labels := map[string]string{"ccm": clusterName + "-crs-ccm"}
	clusterList := &clusterv1.ClusterList{}
	cluster := clusterv1.Cluster{}
	err = r.List(ctx, clusterList, client.InNamespace(machineTemplate.Namespace), client.MatchingLabels(labels))
	if err != nil {
		log.Info("Cluster is not available yet")
		log.Error(err, "failed to get owning cluster.")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	} else {
		for _, cluster = range clusterList.Items {
			machineTemplateScope.Info("Get Cluster", "cluster", cluster.Name)
			log.Info("Find cluster")

		}
		if len(clusterList.Items) == 0 {
			log.Info("OscCluster is not available yet")
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}
	oscCluster := &infrastructurev1beta1.OscCluster{}
	oscClusterName := client.ObjectKey{
		Namespace: machineTemplate.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, oscClusterName, oscCluster); err != nil {
		log.Info("OscCluster is not available yet")
		return reconcile.Result{}, err
	}
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    &cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}
	defer func() {
		if err := machineTemplateScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	if !machineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineTemplateScope, clusterScope)
	}
	return r.reconcile(ctx, machineTemplateScope, clusterScope)
}

// reconcile reconcile the creation of the machine
func (r *OscMachineTemplateReconciler) reconcile(ctx context.Context, machineTemplateScope *scope.MachineTemplateScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineTemplateScope.Info("Reconciling OscMachineTemplate")
	controllerutil.AddFinalizer(machineTemplateScope.OscMachineTemplate, "oscmachine.infrastructure.cluster.x-k8s.io")

	if err := machineTemplateScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	vmSvc := r.getVmSvc(ctx, *clusterScope)
	reconcileCapacity, err := reconcileCapacity(ctx, clusterScope, machineTemplateScope, vmSvc)
	if err != nil {
		return reconcileCapacity, err
	}
	return reconcileCapacity, nil

}

// reconcileDelete reconcile the deletion of the machine
func (r *OscMachineTemplateReconciler) reconcileDelete(ctx context.Context, machineTemplateScope *scope.MachineTemplateScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineTemplateScope.Info("Reconciling delete OscMachineTemplate")
	oscmachinetemplate := machineTemplateScope.OscMachineTemplate
	controllerutil.RemoveFinalizer(oscmachinetemplate, "oscmachine.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscMachineTemplateReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrastructurev1beta1.OscMachineTemplate{}).
		Complete(r)
}
