/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var RandIntN = rand.IntN

type MultiAZAllocator struct {
	client client.Client

	mu          sync.Mutex
	deployments map[types.NamespacedName][]types.NamespacedName
	azs         map[types.NamespacedName]string
}

func NewMultiAZAllocator(c client.Client) *MultiAZAllocator {
	return &MultiAZAllocator{
		client:      c,
		deployments: map[types.NamespacedName][]types.NamespacedName{},
		azs:         map[types.NamespacedName]string{},
	}
}

func (a *MultiAZAllocator) deployment(m *infrastructurev1beta1.OscMachine) (types.NamespacedName, error) {
	md := m.GetLabels()[clusterv1.MachineDeploymentNameLabel]
	if md == "" {
		return types.NamespacedName{}, errors.New("no MachineDeployment label found")
	}
	key := types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      md,
	}
	return key, nil
}

func (a *MultiAZAllocator) name(m *infrastructurev1beta1.OscMachine) types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      m.GetName(),
	}
}

func (a *MultiAZAllocator) AllocateAZ(ctx context.Context, m *infrastructurev1beta1.OscMachine, mode infrastructurev1beta1.SubregionMode, azs []string) (string, error) {
	switch {
	case len(azs) == 0:
		return "", errors.New("no subregions configured")
	case len(azs) == 1:
		return azs[0], nil
	case mode == infrastructurev1beta1.SubregionModeRandom:
		return a.allocateRandomAZ(ctx, m, azs)
	default:
		return a.allocateLeastNodeAZ(ctx, m, azs)
	}
}

func (a *MultiAZAllocator) allocateRandomAZ(ctx context.Context, m *infrastructurev1beta1.OscMachine, azs []string) (string, error) {
	az := azs[RandIntN(len(azs))]
	log.FromContext(ctx).V(3).Info("Assigning machine to subregion", "machine", m.Name, "subregion", az)
	return az, nil
}

func (a *MultiAZAllocator) allocateLeastNodeAZ(ctx context.Context, m *infrastructurev1beta1.OscMachine, azs []string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	name := a.name(m)
	az, found := a.azs[name]
	if !found {
		err := a.refreshWorkers(ctx, m.GetNamespace())
		if err != nil {
			return "", fmt.Errorf("allocate AZ: %w", err)
		}
		az, found = a.azs[name]
		if !found {
			return "", errors.New("allocate AZ: no machine found")
		}
	}
	if az != "" {
		log.FromContext(ctx).V(3).Info("Found assigned subregion", "machine", name.Name, "subregion", az)
		return az, nil
	}
	return a.allocateAZ(ctx, m, azs)
}

func (a *MultiAZAllocator) refreshWorkers(ctx context.Context, ns string) error {
	var ms infrastructurev1beta1.OscMachineList
	err := a.client.List(ctx, &ms, client.InNamespace(ns))
	if err != nil {
		return err
	}

	// truncate all clusters in cache
	for _, m := range ms.Items {
		deploy, err := a.deployment(&m)
		if err != nil {
			continue
		}
		if len(a.deployments[deploy]) > 0 {
			a.deployments[deploy] = a.deployments[deploy][:0]
		}
	}

	// refill cache
	for _, m := range ms.Items {
		deploy, err := a.deployment(&m)
		if err != nil {
			continue
		}
		if m.Spec.Node.Vm.GetRole() == infrastructurev1beta1.RoleControlPlane {
			continue
		}
		name := a.name(&m)
		a.deployments[deploy] = append(a.deployments[deploy], name)
		a.azs[name] = ptr.Deref(m.Status.FailureDomain, "")
	}
	return nil
}

func (a *MultiAZAllocator) allocateAZ(ctx context.Context, m *infrastructurev1beta1.OscMachine, azs []string) (string, error) {
	logger := log.FromContext(ctx)
	deploy, err := a.deployment(m)
	if err != nil {
		return "", fmt.Errorf("allocate AZ: %w", err)
	}
	perAZ := lo.Associate(azs, func(az string) (string, int) { return az, 0 })
	for _, name := range a.deployments[deploy] {
		if a.azs[name] != "" {
			perAZ[a.azs[name]]++
		}
	}
	for _, name := range a.deployments[deploy] {
		if a.azs[name] == "" {
			logger.V(5).Info(fmt.Sprintf("Subregion counts: %v", perAZ), "MachineDeployment", deploy)

			min := math.MaxInt
			var az string
			for k, v := range perAZ {
				if v < min {
					az = k
					min = v
				}
			}

			a.azs[name] = az
			perAZ[az]++
			logger.V(3).Info("Assigning machine to subregion", "machine", name.Name, "subregion", az)
		}
		if m.GetName() == name.Name {
			return a.azs[name], nil
		}
	}
	return "", errors.New("allocate AZ: machine was not found")
}
