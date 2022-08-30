package controllers

import (
	"context"
	"fmt"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
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
		return "", fmt.Errorf("%s does not exist", resourceName)
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
	_, err = infrastructurev1beta1.ValidateCidr(netIpRange)
	if err != nil {
		return netTagName, err
	}
	return "", nil
}

// reconcileNet reconcile the Net of the cluster.
func reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope, netSvc net.OscNetInterface) (reconcile.Result, error) {
	clusterScope.Info("Create Net")
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netRef := clusterScope.GetNetRef()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	clusterName := netSpec.ClusterName
	var net *osc.Net
	var err error
	if len(netRef.ResourceMap) == 0 {
		netRef.ResourceMap = make(map[string]string)
	}
	if netSpec.ResourceId != "" {
		netRef.ResourceMap[netName] = netSpec.ResourceId
		netId := netSpec.ResourceId
		clusterScope.Info("Check if the desired net exist", "netName", netName)
		clusterScope.Info("### Get netId ###", "net", netRef.ResourceMap)
		net, err = netSvc.GetNet(netId)
		if err != nil {
			return reconcile.Result{}, err
		}

	}
	if net == nil || netSpec.ResourceId == "" {
		clusterScope.Info("Create the desired net", "netName", netName)
		net, err := netSvc.CreateNet(netSpec, clusterName, netName)
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
func reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope, netSvc net.OscNetInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netId := netSpec.ResourceId
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	clusterScope.Info("Delete net")
	net, err := netSvc.GetNet(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if net == nil {
		clusterScope.Info("The desired net does not exist anymore", "netName", netName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netSvc.DeleteNet(netId)
	if err != nil {
		clusterScope.Info("Delete the desired net", "netName", netName)
		return reconcile.Result{}, fmt.Errorf("%w Can not delete net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
