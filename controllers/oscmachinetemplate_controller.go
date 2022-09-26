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

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileCapacity reconcile oscmachinetemplate capacity
func reconcileCapacity(ctx context.Context, clusterScope *scope.ClusterScope, machineTemplateScope *scope.MachineTemplateScope, vmSvc compute.OscVmInterface) (reconcile.Result, error) {
	var machineSize int
	var machineKcpCount int32
	var machineKwCount int32
	var machineKcpReady int32
	var machineKwReady int32
	var machines []*clusterv1.Machine
	var err error
	vmReplica := machineTemplateScope.GetReplica()
	if vmReplica != 1 {
		machines, _, err = clusterScope.ListMachines(ctx)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get ListMachine", err)
		}
		machineSize = len(machines)
		clusterScope.Info("Get info OscMachine", "machineSize", machineSize)
	} else {
		clusterScope.Info("Do not wait for OscMachine")
		machineSize = 1
		machineKcpReady = 1
		machineKcpCount = 1
	}

	if machineSize > 0 {
		if vmReplica != 1 {
			clusterScope.Info("Get  MachineList")
			names := make([]string, len(machines))
			for i, m := range machines {
				names[i] = fmt.Sprintf("machine/%s", m.Name)
				machineTemplateScope.Info("Get Machines", "machine", m.Name)
				machineLabel := m.Labels
				for labelKey := range machineLabel {
					if labelKey == "cluster.x-k8s.io/control-plane" {
						machineTemplateScope.Info("Get Kcp Machine", "machineKcp", m.Name)
						machineKcpCount++
						if m.Status.Phase == "Running" || m.Status.Phase == "Provisioned" {
							machineKcpReady++
						}
					}
					if labelKey == "cluster.x-k8s.io/deployment-name" {
						machineTemplateScope.Info("Get Kw Machine", "machineKw", m.Name)
						machineKwCount++
						if m.Status.Phase == "Running" || m.Status.Phase == "Provisioned" {
							machineKwReady++
						}
					}
				}
			}
		}
		role := machineTemplateScope.GetRole()
		if role == "controlplane" && machineKcpReady > 0 && machineKcpCount > 0 {
			machineTemplateScope.Info("At least one controlplane node ready")
		} else if role == "" && machineKwReady > 0 && machineKwCount > 0 {
			machineTemplateScope.Info("At least one worker node ready")
		} else {
			machineTemplateScope.Info("Node is not ready yet")
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}
	clusterName := machineTemplateScope.GetClusterName() + "-" + clusterScope.GetUID()
	vmType := machineTemplateScope.GetVmType()
	machineTemplateScope.Info("### Get ClusterName ####", "clusterName", clusterName)
	capacity, err := vmSvc.GetCapacity("OscK8sClusterID/"+clusterName, "owned", vmType)
	if err != nil {
		return reconcile.Result{}, err
	}
	machineTemplateScope.SetCapacity(capacity)
	return reconcile.Result{}, nil
}
