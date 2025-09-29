/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var reVmType = regexp.MustCompile("tinav[0-9]+.c([0-9]+)r([0-9]+)p[0-9]+")

// reconcileCapacity reconcile oscmachinetemplate capacity
func reconcileCapacity(ctx context.Context, machineTemplateScope *scope.MachineTemplateScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	vmType := machineTemplateScope.GetVmType()
	if vmType == "" {
		return reconcile.Result{}, nil
	}
	matches := reVmType.FindStringSubmatch(vmType)
	if len(matches) == 0 {
		log.V(5).Info("status.capacity is only computed for tina vm types")
		return reconcile.Result{}, nil
	}
	capacity := corev1.ResourceList{}

	cpu, err := resource.ParseQuantity(matches[1])
	if err != nil {
		log.V(5).Error(err, "unable to compute cpu capacity for autoscaler")
		return reconcile.Result{}, nil
	}
	capacity[corev1.ResourceCPU] = cpu

	mem, err := resource.ParseQuantity(matches[2] + "Gi")
	if err != nil {
		log.V(5).Error(err, "unable to compute memory capacity for autoscaler")
		return reconcile.Result{}, nil
	}
	capacity[corev1.ResourceMemory] = mem

	log.V(3).Info(fmt.Sprintf("Setting status.capacity to %v", capacity))
	machineTemplateScope.SetCapacity(capacity)
	return reconcile.Result{}, nil
}
