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
	"slices"
	"sync"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var RandIntN = rand.IntN

type MultiAZAllocator struct {
	client client.Client

	mu          sync.Mutex
	deployments map[types.NamespacedName][]types.NamespacedName
	azs         map[types.NamespacedName]infrastructurev1beta2.OscSubRegion
}

func NewMultiAZAllocator(c client.Client) *MultiAZAllocator {
	return &MultiAZAllocator{
		client:      c,
		deployments: map[types.NamespacedName][]types.NamespacedName{},
		azs:         map[types.NamespacedName]infrastructurev1beta2.OscSubRegion{},
	}
}

func (a *MultiAZAllocator) deployment(m *infrastructurev1beta2.OscMachine) (types.NamespacedName, error) {
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

func (a *MultiAZAllocator) name(m *infrastructurev1beta2.OscMachine) types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      m.GetName(),
	}
}

func (a *MultiAZAllocator) AllocateAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, mode infrastructurev1beta2.SubregionMode, role infrastructurev1beta2.OscRole, nets []infrastructurev1beta2.OscNet, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscNet, infrastructurev1beta2.OscSubnet, infrastructurev1beta2.OscSubRegion, error) {
	var az infrastructurev1beta2.OscSubRegion
	var err error
	switch {
	case len(azs) == 0:
		err = errors.New("no subregions configured")
	case len(azs) == 1:
		az = azs[0]
	case mode == infrastructurev1beta2.SubregionModeRandom:
		az, err = a.allocateRandomAZ(ctx, m, azs)
	default:
		az, err = a.allocateLeastNodeAZ(ctx, m, azs)
	}
	if err != nil {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", err
	}
	possibles := lo.FilterMap(nets, func(n infrastructurev1beta2.OscNet, i int) (int, bool) {
		if slices.ContainsFunc(n.Subnets, func(sn infrastructurev1beta2.OscSubnet) bool {
			return sn.Subregion == az && sn.HasRole(role)
		}) {
			return i, true
		}
		return -1, false
	})
	var n infrastructurev1beta2.OscNet
	switch len(possibles) {
	case 0:
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", errors.New("no net found")
	case 1:
		n = nets[possibles[0]]
	default:
		n = nets[possibles[RandIntN(len(possibles))]]
	}
	sn, found := lo.Find(n.Subnets, func(sn infrastructurev1beta2.OscSubnet) bool {
		return sn.Subregion == az && sn.HasRole(role)
	})
	if !found {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", errors.New("no subnet found")
	}
	return n, sn, az, nil
}

func (a *MultiAZAllocator) allocateRandomAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscSubRegion, error) {
	az := azs[RandIntN(len(azs))]
	log.FromContext(ctx).V(3).Info("Assigning machine to subregion", "machine", m.Name, "subregion", az)
	return az, nil
}

func (a *MultiAZAllocator) allocateLeastNodeAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscSubRegion, error) {
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
	var ms infrastructurev1beta2.OscMachineList
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
		if m.Spec.Vm.GetRole() == infrastructurev1beta2.RoleControlPlane {
			continue
		}
		name := a.name(&m)
		a.deployments[deploy] = append(a.deployments[deploy], name)
		a.azs[name] = infrastructurev1beta2.OscSubRegion(ptr.From(m.Status.FailureDomain))
	}
	return nil
}

func (a *MultiAZAllocator) allocateAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscSubRegion, error) {
	logger := log.FromContext(ctx)
	deploy, err := a.deployment(m)
	if err != nil {
		return "", fmt.Errorf("allocate AZ: %w", err)
	}
	perAZ := lo.Associate(azs, func(az infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscSubRegion, int) { return az, 0 })
	for _, name := range a.deployments[deploy] {
		if a.azs[name] != "" {
			perAZ[a.azs[name]]++
		}
	}
	for _, name := range a.deployments[deploy] {
		if a.azs[name] == "" {
			logger.V(5).Info(fmt.Sprintf("Subregion counts: %v", perAZ), "MachineDeployment", deploy)

			min := math.MaxInt
			var az infrastructurev1beta2.OscSubRegion
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
