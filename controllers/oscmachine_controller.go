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
	"time"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OscMachineReconciler reconciles a OscMachine object
type OscMachineReconciler struct {
	client.Client
	Cloud            services.Servicer
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
	WatchFilterValue string
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

func (r *OscMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()

	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling OscMachine")

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
		OscClient:  r.Cloud.OscClient(),
		Cluster:    cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, err
	}
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:     r.Client,
		OscClient:  r.Cloud.OscClient(),
		Cluster:    cluster,
		Machine:    machine,
		OscCluster: oscCluster,
		OscMachine: oscMachine,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}
	defer func() {
		if err := machineScope.Close(ctx); err != nil && reterr == nil {
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
	log := ctrl.LoggerFrom(ctx)
	oscmachine := machineScope.OscMachine
	if oscmachine.Status.FailureReason != nil || oscmachine.Status.FailureMessage != nil {
		log.V(3).Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	controllerutil.AddFinalizer(oscmachine, "oscmachine.infrastructure.cluster.x-k8s.io")

	if !machineScope.Cluster.Status.InfrastructureReady {
		log.V(3).Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		log.V(3).Info("Bootstrap data secret reference is not yet availablle")
		return ctrl.Result{}, nil
	}
	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		log.V(2).Info("Find volumes")
		volumeName, err := checkVolumeFormatParameters(machineScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create volume %s: %w", volumeName, err)
		}
	}

	UseFailureDomain(clusterScope, machineScope)

	vmName, err := checkVmFormatParameters(machineScope, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create vm %s: %w", vmName, err)
	}

	keypairName, err := checkKeypairFormatParameters(machineScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create vm %s: %w", keypairName, err)
	}

	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		duplicateResourceVolumeErr := checkVolumeOscDuplicateName(machineScope)
		if duplicateResourceVolumeErr != nil {
			return reconcile.Result{}, duplicateResourceVolumeErr
		}
	}

	duplicateResourceVmPrivateIpErr := checkVmPrivateIpOscDuplicateName(machineScope)
	if duplicateResourceVmPrivateIpErr != nil {
		return reconcile.Result{}, duplicateResourceVmPrivateIpErr
	}
	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		checkOscAssociateVmVolumeErr := checkVmVolumeOscAssociateResourceName(machineScope)
		if checkOscAssociateVmVolumeErr != nil {
			return reconcile.Result{}, checkOscAssociateVmVolumeErr
		}
	}

	checkKeypairSameNameErr := checkKeypairSameName(machineScope)
	if checkKeypairSameNameErr != nil {
		return reconcile.Result{}, checkKeypairSameNameErr
	}

	checkOscAssociateVmSecurityGroupErr := checkVmSecurityGroupOscAssociateResourceName(machineScope, clusterScope)
	if checkOscAssociateVmSecurityGroupErr != nil {
		return reconcile.Result{}, checkOscAssociateVmSecurityGroupErr
	}

	checkOscAssociateVmSubnetErr := checkVmSubnetOscAssociateResourceName(machineScope, clusterScope)
	if checkOscAssociateVmSubnetErr != nil {
		return reconcile.Result{}, checkOscAssociateVmSubnetErr
	}

	vmSpec := machineScope.GetVm()
	vmSpec.SetDefaultValue()
	if vmSpec.PublicIpName != "" {
		checkOscAssociateVmPublicIpErr := checkVmPublicIpOscAssociateResourceName(machineScope, clusterScope)
		if checkOscAssociateVmPublicIpErr != nil {
			return reconcile.Result{}, checkOscAssociateVmPublicIpErr
		}
	}

	if vmSpec.LoadBalancerName != "" {
		checkOscAssociateVmLoadBalancerErr := checkVmLoadBalancerOscAssociateResourceName(machineScope, clusterScope)
		if checkOscAssociateVmLoadBalancerErr != nil {
			return reconcile.Result{}, checkOscAssociateVmLoadBalancerErr
		}
	}
	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		checkVmVolumeSubregionNameErr := checkVmVolumeSubregionName(machineScope)
		if checkVmVolumeSubregionNameErr != nil {
			return reconcile.Result{}, checkVmVolumeSubregionNameErr
		}
	}

	imageSvc := r.Cloud.Image(ctx, *clusterScope)
	reconcileImage, err := reconcileImage(ctx, machineScope, imageSvc)
	if err != nil {
		log.V(2).Error(err, "failed to reconcile Image")
		return reconcileImage, err
	}

	volumeSvc := r.Cloud.Volume(ctx, *clusterScope)
	tagSvc := r.Cloud.Tag(ctx, *clusterScope)
	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		reconcileVolume, err := reconcileVolume(ctx, machineScope, volumeSvc, tagSvc)
		if err != nil {
			log.V(2).Error(err, "failed to reconcile volume")
			conditions.MarkFalse(oscmachine, infrastructurev1beta1.VolumeReadyCondition, infrastructurev1beta1.VolumeReconciliationFailedReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
			return reconcileVolume, err
		}
	}
	conditions.MarkTrue(oscmachine, infrastructurev1beta1.VolumeReadyCondition)

	keypairSvc := r.Cloud.KeyPair(ctx, *clusterScope)
	reconcileKeypair, err := reconcileKeypair(ctx, machineScope, keypairSvc)
	if err != nil {
		log.V(2).Error(err, "failed to reconcile keypair")
		return reconcileKeypair, err
	}

	publicIpSvc := r.Cloud.PublicIp(ctx, *clusterScope)
	vmSvc := r.Cloud.VM(ctx, *clusterScope)
	loadBalancerSvc := r.Cloud.LoadBalancer(ctx, *clusterScope)
	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)

	reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, vmSvc, volumeSvc, publicIpSvc, loadBalancerSvc, securityGroupSvc, tagSvc)
	if err != nil {
		log.V(2).Error(err, "failed to reconcile vm")
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.VmNotReadyReason, clusterv1.ConditionSeverityWarning, "%s", err.Error())
		return reconcileVm, err
	}

	vmState := machineScope.GetVmState()
	switch *vmState {
	case infrastructurev1beta1.VmStatePending:
		machineScope.SetNotReady()
		log.V(4).Info("Vm pending", "state", vmState)
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.VmNotReadyReason, clusterv1.ConditionSeverityWarning, "")
	case infrastructurev1beta1.VmStateStopping, infrastructurev1beta1.VmStateStopped:
		machineScope.SetNotReady()
		log.V(4).Info("Vm stopped", "state", vmState)
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.VmStoppedReason, clusterv1.ConditionSeverityWarning, "")
	case infrastructurev1beta1.VmStateRunning:
		machineScope.SetReady()
		log.V(4).Info("Vm running", "state", vmState)
		conditions.MarkTrue(oscmachine, infrastructurev1beta1.VmReadyCondition)
	case infrastructurev1beta1.VmStateShuttingDown, infrastructurev1beta1.VmStateTerminated:
		machineScope.SetNotReady()
		log.V(4).Info("Unexpected vm termination", "state", vmState)
		conditions.MarkFalse(oscmachine, infrastructurev1beta1.VmReadyCondition, infrastructurev1beta1.VmTerminatedReason, clusterv1.ConditionSeverityError, "")
	default:
		machineScope.SetNotReady()
		log.V(4).Info("Vm state is undefined", "state", vmState)
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(fmt.Errorf("instance state %+v  is undefined", vmState))
		conditions.MarkUnknown(oscmachine, infrastructurev1beta1.VmReadyCondition, "", "")
	}
	return reconcile.Result{}, nil
}

// reconcileDelete reconcile the deletion of the machine
func (r *OscMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Reconciling delete OscMachine")
	oscmachine := machineScope.OscMachine
	if len(machineScope.OscMachine.Spec.Node.Volumes) > 0 {
		volumeSvc := r.Cloud.Volume(ctx, *clusterScope)
		reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, volumeSvc)
		if err != nil {
			return reconcileDeleteVolume, err
		}
	}
	publicIpSvc := r.Cloud.PublicIp(ctx, *clusterScope)
	vmSvc := r.Cloud.VM(ctx, *clusterScope)
	loadBalancerSvc := r.Cloud.LoadBalancer(ctx, *clusterScope)
	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, vmSvc, publicIpSvc, loadBalancerSvc, securityGroupSvc)
	if err != nil {
		return reconcileDeleteVm, err
	}
	keypairSvc := r.Cloud.KeyPair(ctx, *clusterScope)
	reconcileDeleteKeyPair, err := reconcileDeleteKeypair(ctx, machineScope, keypairSvc)
	if err != nil {
		return reconcileDeleteKeyPair, err
	}
	controllerutil.RemoveFinalizer(oscmachine, "oscmachine.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &infrastructurev1beta1.OscMachineList{}, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to create mapper for Cluster to OscMachines: %w", err)
	}
	err = ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		For(&infrastructurev1beta1.OscMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrastructurev1beta1.GroupVersion.WithKind("OscMachine"))),
		).
		Watches(
			&infrastructurev1beta1.OscCluster{},
			handler.EnqueueRequestsFromMapFunc(r.OscClusterToOscMachines(ctx)),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx))),
		).
		Complete(r)

	if err != nil {
		return fmt.Errorf("error creating controller: %w", err)
	}
	return err
}

// OscClusterToOscMachines convert the cluster to machine spec
func (r *OscMachineReconciler) OscClusterToOscMachines(ctx context.Context) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}
		log := log.FromContext(ctx)

		c, ok := o.(*infrastructurev1beta1.OscCluster)
		if !ok {
			log.V(1).Error(fmt.Errorf("expected a OscCluster but got a %T", o), "failed to get OscMachine for OscCluster")
			return nil
		}
		log = log.WithValues("objectMapper", "oscClusterToOscMachine", "namespace", c.Namespace)

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
			log.V(1).Error(err, "failed to get owning cluster, skipping mapping.")
			return result
		}

		labels := map[string]string{"cluster.x-k8s.io/cluster-name": cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.V(1).Error(err, "failed to list Machines, skipping mapping.")
			return nil
		}
		for _, m := range machineList.Items {
			log.WithValues("machine", m.Name)
			if m.Spec.InfrastructureRef.GroupVersionKind().Kind != "OscMachine" {
				log.V(1).Info("Machine has an InfrastructureRef for a different type, will not add to reconciliation request.")
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
