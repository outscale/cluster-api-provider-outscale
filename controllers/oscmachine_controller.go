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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OscMachineReconciler reconciles a OscMachine object
type OscMachineReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachines/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OscMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

// getVolumeSvc retrieve volumeSvc
func (r *OscMachineReconciler) getVolumeSvc(ctx context.Context, scope scope.ClusterScope) storage.OscVolumeInterface {
	return storage.NewService(ctx, &scope)
}

func (r *OscMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)

	oscMachine := &infrastructurev1beta1.OscMachine{}
	if err := r.Get(ctx, req.NamespacedName, oscMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	machine, err := util.GetOwnerMachine(ctx, r.Client, oscMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}

	if machine == nil {
		log.Info("Machine Controller has not yet set OwnRef")
		return reconcile.Result{}, nil
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("machine", machine.Name)
	oscCluster := &infrastructurev1beta1.OscCluster{}
	oscClusterNamespacedName := client.ObjectKey{
		Namespace: oscMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Get(ctx, oscClusterNamespacedName, oscCluster); err != nil {
		log.Info("OscCluster is not available yet")
		return reconcile.Result{}, nil
	}
	if annotations.IsPaused(cluster, oscCluster) {
		log.Info("OscMachine or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:     log,
		Client:     r.Client,
		Cluster:    cluster,
		Machine:    machine,
		OscCluster: oscCluster,
		OscMachine: oscMachine,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	if !oscMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope, clusterScope)
	}
	return r.reconcile(ctx, machineScope, clusterScope)
}

// reconcile reconcile the creation of the machine
func (r *OscMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling OscMachine")
	oscmachine := machineScope.OscMachine
	if oscmachine.Status.FailureReason != nil || oscmachine.Status.FailureMessage != nil {
		machineScope.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	controllerutil.AddFinalizer(oscmachine, "oscmachine.infrastructure.cluster.x-k8s.io")

	machineScope.Info("Set OscMachine status to not ready")
	machineScope.SetNotReady()
	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.InstanceReadyCondition, infrastructurev1beta1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}
	machineScope.Info("Check bootstrap data")
	if machineScope.Machine.Spec.Bootstrap.DataSecretName != nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
	}
	volumeName, err := checkVolumeFormatParameters(machineScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not create volume %s for OscMachine %s/%s", err, volumeName, machineScope.GetNamespace(), machineScope.GetName())
	}
	duplicateResourceVolumeErr := checkVolumeOscDuplicateName(machineScope)
	if duplicateResourceVolumeErr != nil {
		return reconcile.Result{}, duplicateResourceVolumeErr
	}

	volumeSvc := r.getVolumeSvc(ctx, *clusterScope)
	reconcileVolume, err := reconcileVolume(ctx, machineScope, volumeSvc)
	if err != nil {
		machineScope.Error(err, "failed to reconcile volume")
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VolumeReadyCondition, infrastructurev1beta1.VolumeReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileVolume, err
	}
	conditions.MarkTrue(oscmachine, infrastructurev1beta1.VolumeReadyCondition)
	machineScope.Info("Set OscMachine status to ready")
	machineScope.SetReady()

	return reconcile.Result{}, nil
}

// reconcileDelete reconcile the deletion of the machine
func (r *OscMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	machineScope.Info("Reconciling delete OscMachine")
	oscmachine := machineScope.OscMachine

	volumeSvc := r.getVolumeSvc(ctx, *clusterScope)
	reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, volumeSvc)
	if err != nil {
		return reconcileDeleteVolume, err
	}
	controllerutil.RemoveFinalizer(oscmachine, "oscmachine.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscMachine{}).
		Complete(r)
}

// OscClusterToOscMachines convert the cluster to machine spec
func (r *OscMachineReconciler) OscClusterToOscMachines(ctx context.Context) handler.MapFunc {
	return func(o client.Object) []ctrl.Request {
		result := []ctrl.Request{}
		log := log.FromContext(ctx)

		c, ok := o.(*infrastructurev1beta1.OscCluster)
		if !ok {
			log.Error(fmt.Errorf("expected a OscCluster but got a %T", o), "failed to get OscMachine for OscCluster")
			return nil
		}
		log = log.WithValues("objectMapper", "oscClusterToOscMachine", "namespace", c.Namespace, "oscCluster", c.Name)

		if !c.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(1).Info("OscCluster has a deletion timestamp, skipping mapping.")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			log.V(1).Info("Cluster for OscCluster not found, skipping mapping.")
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster, skipping mapping.")
			return result
		}

		labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines, skipping mapping.")
			return nil
		}
		for _, m := range machineList.Items {
			log.WithValues("machine", m.Name)
			if m.Spec.InfrastructureRef.GroupVersionKind().Kind != "OscMachine" {
				log.V(1).Info("Machine has an InfrastructureRef for a different type, will not add to reconcilation request.")
				continue
			}
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			log.WithValues("oscMachine", name.Name)
			log.V(1).Info("Adding OscMachine to reconciliation request.")
			result = append(result, ctrl.Request{NamespacedName: name})
		}
		return result
	}
}
