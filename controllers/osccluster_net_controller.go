package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
        osc "github.com/outscale/osc-sdk-go/v2"

)

// GetNetResourceId return the netId from the resourceMap base on resourceName (tag name + cluster object uid) 
func GetNetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
                netRef := clusterScope.GetNetRef()
                if netId, ok := netRef.ResourceMap[resourceName]; ok {
                        return netId, nil
                } else {
                        return "", fmt.Errorf("%s is not exist", resourceName)
                }
}

// CheckNetFormatParameters check net parameters format (Tag format, cidr format, ..)
func CheckNetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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

// ReconcileNet reconcile the Net of the cluster.
func reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {

	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

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
			return reconcile.Result{}, fmt.Errorf("%w Can not create net for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
		}
		clusterScope.Info("### Get net ###", "net", net)
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = *net.NetId
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = net.GetNetId()
	}
	clusterScope.Info("Info net", "net", net)
	return reconcile.Result{}, nil
}

// ReconcileDeleteNet reconcile the destruction of the Net of the cluster.
func reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netId :=  netSpec.ResourceId 

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
		return reconcile.Result{}, fmt.Errorf("%w Can not delete net for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
	}
	return reconcile.Result{}, nil
}


