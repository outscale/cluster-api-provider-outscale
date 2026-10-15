/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	cloudnet "github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *OscClusterReconciler) getManagementRouteTablesAndIPRange(ctx context.Context, mgmtSvc cloudnet.Servicer, clusterScope *scope.ClusterScope) ([]osc.RouteTable, string, error) {
	if !clusterScope.GetSpec().NetPeering.Enable {
		return nil, "", nil
	}
	netId := r.getMgmtNetID(clusterScope)
	rtbls, err := mgmtSvc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return nil, "", fmt.Errorf("find mgmt route tables: %w", err)
	}
	subnetId := clusterScope.GetSpec().NetPeering.ManagementSubnetID
	if subnetId == "" {
		n, err := mgmtSvc.GetNet(ctx, netId)
		if err != nil {
			return nil, "", fmt.Errorf("get mgmt net: %w", err)
		}
		return rtbls, n.IpRange, nil
	}
	sn, err := mgmtSvc.GetSubnet(ctx, subnetId)
	if err != nil {
		return nil, "", fmt.Errorf("get mgmt subnet: %w", err)
	}
	for _, rtbl := range rtbls {
		if slices.ContainsFunc(rtbl.LinkRouteTables, func(l osc.LinkRouteTable) bool {
			return l.SubnetId == subnetId
		}) {
			return []osc.RouteTable{rtbl}, sn.IpRange, nil
		}
	}
	return nil, "", errors.New("no routing table are linked to managementSubnetId")
}

// reconcileNetPeeringRoutes reconcile the NetPeering routes.
func (r *OscClusterReconciler) reconcileNetPeeringRoutes(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not reconciling netPeering routes for existing net")
		return reconcile.Result{}, nil
	}
	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerNetPeeringRoutes) {
		log.V(4).Info("No need for netPeering routes reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling netPeering routes")

	netSpec := clusterScope.GetNets()[0]
	hubNetId, err := r.Tracker.getNetId(ctx, netSpec.IpRange, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get net: %w", err)
	}

	svc := r.Cloud.Net(clusterScope.Tenant)
	mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get mgmt credentials: %w", err)
	}
	mgmtSvc := r.Cloud.Net(mgmt)
	mgmtRtbls, mgmtIPRange, err := r.getManagementRouteTablesAndIPRange(ctx, mgmtSvc, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list mgmt route tables: %w", err)
	}
	spokeIpRange := func(spec netPeering) string {
		if spec.management {
			return mgmtIPRange
		}
		return spec.ipRange
	}
	hubIpRange := netSpec.IpRange

	netPeerings := r.listNetPeerings(clusterScope)
	for _, spec := range netPeerings {
		np, err := r.Tracker.getNetPeering(ctx, clusterScope, spec.management, spec.ipRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("get netpeering: %w", err)
		}

		checkRoute := func(hubSpoke string, rtbl osc.RouteTable, destRange string, svc cloudnet.Servicer) error {
			// check if route to netpeering already exists
			if slices.ContainsFunc(rtbl.Routes, func(r osc.Route) bool {
				return ptr.From(r.NetPeeringId) == np.NetPeeringId
			}) {
				return nil
			}
			// check that no route overlaps
			if r, found := lo.Find(rtbl.Routes, func(r osc.Route) bool {
				return r.DestinationIpRange != "0.0.0.0/0" && overlappingCIDR(r.DestinationIpRange, destRange)
			}); found {
				return FatalErrorf("%s overlaps an existing route to %s in %s route table %s", destRange, destination(r), hubSpoke, rtbl.RouteTableId)
			}
			// create route
			log.V(3).Info("Creating "+hubSpoke+" route to netPeering", "routeTableId", rtbl.RouteTableId, "IPRange", destRange)
			_, err := svc.CreateRoute(ctx, destRange, rtbl.RouteTableId, np.NetPeeringId, "netPeering")
			if err != nil {
				return fmt.Errorf("create %s route: %w", hubSpoke, err)
			}
			r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.NetPeeringCreatedReason, "NetPeering "+hubSpoke+" route created")
			return nil
		}

		// Add all routes to spoke route table
		spokeSvc := svc
		if spec.management {
			spokeSvc = mgmtSvc
		}
		spokeRtbls := mgmtRtbls
		if !spec.management {
			spokeNetID, err := r.Tracker.getNetId(ctx, spec.ipRange, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("get spoke net: %w", err)
			}
			spokeRtbls, err = svc.GetRouteTablesFromNet(ctx, spokeNetID)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("list spoke route tables: %w", err)
			}
		}
		// Add route to hub in spoke route table
		for _, mgmtRtbl := range spokeRtbls {
			if err := checkRoute("spoke", mgmtRtbl, hubIpRange, spokeSvc); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Add routes to other spokes in spoke route table
		spokeRange := spokeIpRange(spec)
		for _, spec := range netPeerings {
			destRange := spokeIpRange(spec)
			if destRange == spokeRange {
				continue
			}
			for _, mgmtRtbl := range spokeRtbls {
				if err := checkRoute("spoke", mgmtRtbl, destRange, spokeSvc); err != nil {
					return reconcile.Result{}, err
				}
			}
		}
		// Add routes to spoke in hub route tables
		rtbls, err := svc.GetRouteTablesFromNet(ctx, hubNetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("list hub route tables: %w", err)
		}
		for _, rtbl := range rtbls {
			if err := checkRoute("hub", rtbl, spokeRange, svc); err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerNetPeeringRoutes)
	return reconcile.Result{}, nil
}

// reconcileDeleteNetPeeringRoutes deletes routes to netpeering on the mgmt cluster. No need to delete workload routes, they will be deleted with the route tables.
func (r *OscClusterReconciler) reconcileDeleteNetPeeringRoutes(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not deleting existing netPeering routes")
		return reconcile.Result{}, nil
	}

	npId, err := r.Tracker.getNetPeeringId(ctx, clusterScope, true, "")
	switch {
	case IsNotFound(err):
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, err
	}

	mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get mgmt credentials: %w", err)
	}
	mgmtSvc := r.Cloud.Net(mgmt)
	// remove routes from management route tables
	mgmtRtbls, _, err := r.getManagementRouteTablesAndIPRange(ctx, mgmtSvc, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list mgmt route tables: %w", err)
	}
	for _, mgmtRtbl := range mgmtRtbls {
		for _, r := range mgmtRtbl.Routes {
			if ptr.From(r.NetPeeringId) != npId {
				continue
			}
			log.V(3).Info("Deleting management route to netPeering", "routeTableId", mgmtRtbl.RouteTableId, "IPRange", r.DestinationIpRange)
			err = mgmtSvc.DeleteRoute(ctx, r.DestinationIpRange, mgmtRtbl.RouteTableId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("delete mgmt route: %w", err)
			}
		}
	}

	return reconcile.Result{}, nil
}

func overlappingCIDR(r1, r2 string) bool {
	if r1 == r2 {
		return true
	}
	_, c1, err1 := net.ParseCIDR(r1)
	_, c2, err2 := net.ParseCIDR(r2)
	if err1 != nil || err2 != nil {
		return false
	}
	return c2.Contains(c1.IP) || c1.Contains(c2.IP)
}

func destination(r osc.Route) string {
	switch {
	case r.DestinationServiceId != nil:
		return *r.DestinationServiceId
	case r.GatewayId != nil:
		return *r.GatewayId
	case r.NatServiceId != nil:
		return *r.NatServiceId
	case r.NetAccessPointId != nil:
		return *r.NetAccessPointId
	case r.NetPeeringId != nil:
		return *r.NetPeeringId
	case r.NicId != nil:
		return *r.NicId
	default:
		return "???"
	}
}
