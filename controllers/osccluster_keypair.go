/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/goutils/k8s/tags"
	"github.com/outscale/goutils/sdk/ptr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileKeypair reconcile the Keypair of the cluster.
func (r *OscClusterReconciler) reconcileKeypair(ctx context.Context, clusterScope *scope.ClusterScope) Result {
	log := ctrl.LoggerFrom(ctx)
	kpSpec := clusterScope.GetKeypair()
	if kpSpec == nil {
		log.V(4).Info("No keypair defined")
		return Result{Status: ResultStatusOK}
	}
	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerKeypair) {
		log.V(4).Info("No need for keypair reconciliation")
		return Result{Status: ResultStatusOK}
	}
	log.V(4).Info("Reconciling keypair")

	var secret corev1.Secret
	err := r.Client.Get(ctx, client.ObjectKey{
		Name:      kpSpec.SecretName,
		Namespace: clusterScope.OscCluster.Namespace,
	}, &secret)
	switch {
	case apierrors.IsNotFound(err):
	case err != nil:
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("get secret: %w", err)}
	default:
		log.V(4).Info("Found existing secret", "name", secret.Name)
		clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerKeypair)
		return Result{Status: ResultStatusOK}
	}
	name := kpSpec.Name
	svc := r.Cloud.Compute(clusterScope.Tenant)
	kp, err := svc.GetKeypair(ctx, name)
	switch {
	case err != nil:
		return Result{Status: ResultStatusUnknown, Error: fmt.Errorf("get existing: %w", err)}
	case kp == nil:
	default:
		log.V(4).Info("Found existing keypair", "keypairId", kp.KeypairId)
	}
	if kp != nil {
		found, lc := tags.HasClusterID(ptr.From(kp.Tags), clusterScope.GetUID())
		if !found || lc != tags.ResourceLifecycleOwned {
			return Result{Status: ResultStatusNotOK, Error: errors.New("a keypair with the same name already exists")}
		}
		log.V(2).Info("Deleting keypair created from a previous loop", "name", name)
		err = svc.DeleteKeypair(ctx, name)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot delete keypair: %w", err)}
		}
	}
	log.V(3).Info("Creating keypair")
	kpc, err := svc.CreateKeypair(ctx, name, clusterScope.GetUID())
	if err != nil {
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot create keypair: %w", err)}
	}
	log.V(2).Info("Created keypair", "keypairId", kpc.KeypairId, "name", name)

	secret = corev1.Secret{
		Name:      kpSpec.SecretName,
		Namespace: clusterScope.OscCluster.Namespace,
		Immutable: new(true),
		Type:      corev1.SecretTypeSSHAuth,
		Data: map[string][]byte{
			corev1.SSHAuthPrivateKey: []byte(ptr.From(kpc.PrivateKey)),
		},
	}
	err = r.Client.Create(ctx, &secret)
	if err != nil {
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot create keypair secret: %w", err)}
	}

	r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.KeypairCreatedReason, "Keypair created")

	clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerKeypair)
	return Result{Status: ResultStatusOK}
}

// reconcileDeleteKeypair reconcile the destruction of the keypair of the cluster.
func (r *OscClusterReconciler) reconcileDeleteKeypair(ctx context.Context, clusterScope *scope.ClusterScope) Result {
	log := ctrl.LoggerFrom(ctx)
	kpSpec := clusterScope.GetKeypair()

	if kpSpec == nil {
		return Result{Status: ResultStatusOK}
	}

	name := kpSpec.Name

	if kpSpec.KeepAfterDeletion {
		log.V(3).Info("Not deleting keypair configured to be kept", "name", name)
		return Result{Status: ResultStatusOK}
	}
	var secret corev1.Secret
	ok := client.ObjectKey{
		Name:      kpSpec.SecretName,
		Namespace: clusterScope.OscCluster.Namespace,
	}
	err := r.Client.Get(ctx, ok, &secret)
	switch {
	case apierrors.IsNotFound(err):
	case err != nil:
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("get secret: %w", err)}
	default:
		log.V(4).Info("Deleting keypair secret", "name", kpSpec.SecretName)
		err := r.Client.Delete(ctx, &secret)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("delete secret: %w", err)}
		}
	}

	svc := r.Cloud.Compute(clusterScope.Tenant)
	kp, err := svc.GetKeypair(ctx, name)
	switch {
	case err != nil:
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("get existing: %w", err)}
	case kp == nil:
		log.V(4).Info("The keypair is already deleted")
		return Result{Status: ResultStatusOK}
	}

	log.V(2).Info("Deleting keypair", "name", name)
	err = svc.DeleteKeypair(ctx, name)
	if err != nil {
		return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot delete keypair: %w", err)}
	}

	return Result{Status: ResultStatusOK}
}
