package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getNetResourceId return the netId from the resourceMap base on resourceName (tag name + cluster object uid)
func getNetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	netRef := clusterScope.GetNetRef()
	if netId, ok := netRef.ResourceMap[resourceName]; ok {
		return netId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// checkNetFormatParameters check net parameters format (Tag format, cidr format, ..)
func checkNetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Net name parameters ")
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netTagName, err := tag.ValidateTagNameValue(netName)
	if err != nil {
		return netTagName, err
	}
	clusterScope.Info("Check Net IpRange parameters")
	netIpRange := netSpec.IpRange
	_, err = net.ValidateCidr(netIpRange)
	if err != nil {
		return netTagName, err
	}
	return "", nil
}

// reconcileNet reconcile the Net of the cluster.
func reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {

	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Create Net")
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netRef := clusterScope.GetNetRef()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	var net *osc.Net
	var err error
	if len(netRef.ResourceMap) == 0 {
		netRef.ResourceMap = make(map[string]string)
	}
	if netSpec.ResourceId != "" {
		netRef.ResourceMap[netName] = netSpec.ResourceId
		netId := netSpec.ResourceId
		clusterScope.Info("### Get netId ###", "net", netRef.ResourceMap)
		net, err = netsvc.GetNet(netId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if net == nil || netSpec.ResourceId == "" {
		net, err := netsvc.CreateNet(netSpec, netName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("### Get net ###", "net", net)
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = *net.NetId
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = net.GetNetId()
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteNet reconcile the destruction of the Net of the cluster.
func reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netId := netSpec.ResourceId

	clusterScope.Info("Delete net")
	net, err := netsvc.GetNet(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if net == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.DeleteNet(netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
