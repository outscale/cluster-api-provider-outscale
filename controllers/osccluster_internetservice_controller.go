package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid)
func GetInternetServiceResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	internetServiceRef := clusterScope.GetInternetServiceRef()
	if internetServiceId, ok := internetServiceRef.ResourceMap[resourceName]; ok {
		return internetServiceId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format
func CheckInternetServiceFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Internet Service parameters")
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()
	internetServiceTagName, err := tag.ValidateTagNameValue(internetServiceName)
	if err != nil {
		return internetServiceTagName, err
	}
	return "", nil
}

// ReconcileInternetService reconcile the InternetService of the cluster.
func reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create InternetGateway")
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceRef := clusterScope.GetInternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(internetServiceRef.ResourceMap) == 0 {
		internetServiceRef.ResourceMap = make(map[string]string)
	}
	if internetServiceSpec.ResourceId != "" {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
	}
	internetServiceId := internetServiceRef.ResourceMap[internetServiceName]
	clusterScope.Info("### Get internetServiceId ###", "internetservice", internetServiceRef.ResourceMap)
	internetService, err := netsvc.GetInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetService == nil {
		internetService, err = netsvc.CreateInternetService(internetServiceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not create internetservice for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
		err = netsvc.LinkInternetService(*internetService.InternetServiceId, netId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not link internetService with net for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("### Get internetService ###", "internetservice", internetService)
		internetServiceRef.ResourceMap[internetServiceName] = *internetService.InternetServiceId
		internetServiceSpec.ResourceId = *internetService.InternetServiceId

	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete internetService")

	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceRef := clusterScope.GetInternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()

	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	internetServiceId := internetServiceRef.ResourceMap[internetServiceName]
	internetservice, err := netsvc.GetInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.UnlinkInternetService(internetServiceRef.ResourceMap[internetServiceName], netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w: Can not unlink internetService and net for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
	}
	err = netsvc.DeleteInternetService(internetServiceRef.ResourceMap[internetServiceName])
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w: Can not delete internetService for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, nil
}
