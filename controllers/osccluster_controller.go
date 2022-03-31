/*
Copyright 2022.

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
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"time"
        "os"
        "fmt"
	//      "k8s.io/apimachinery/pkg/runtime"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/util/reconciler"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service" 
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OscClusterReconciler reconciles a OscCluster object
type OscClusterReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OscCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OscClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()
	log := ctrl.LoggerFrom(ctx)
	oscCluster := &infrastructurev1beta1.OscCluster{}

	log.Info("Please WAIT !!!!")

	if err := r.Get(ctx, req.NamespacedName, oscCluster); err != nil {
		if apierrors.IsNotFound(err) {
    			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
          	return ctrl.Result{}, err
        }
	log.Info("Still WAIT !!!!")
        log.Info("Create info", "env", os.Environ())

	cluster, err := util.GetOwnerCluster(ctx, r.Client, oscCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, oscCluster) {
		log.Info("oscCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	osccluster := clusterScope.OscCluster
	if !osccluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}
        loadBalancerSpec := clusterScope.LoadBalancer()
        loadBalancerSpec.SetDefaultValue()
        log.Info("Create loadBalancer", "loadBalancerName", loadBalancerSpec.LoadBalancerName, "SubregionName", loadBalancerSpec.SubregionName)
	return r.reconcile(ctx, clusterScope)
}

func GetResourceId(resourceName string, resourceType string, clusterScope *scope.ClusterScope) (string, error) {
    switch {
    case resourceType == "net":
        netRef := clusterScope.NetRef()
        if netId, ok := netRef.ResourceMap[resourceName]; ok {
            return netId, nil
        } else {
            return "", fmt.Errorf("%s is not exist", resourceName)
        }      
    case resourceType == "subnet":
        subnetRef := clusterScope.SubnetRef()
        if subnetId, ok := subnetRef.ResourceMap[resourceName]; ok {
            return subnetId, nil
        } else {
            return "", fmt.Errorf("%s is not exist", resourceName)
        }
    case resourceType == "gateway":
        internetServiceRef := clusterScope.InternetServiceRef()
        if internetServiceId, ok := internetServiceRef.ResourceMap[resourceName]; ok {
            return internetServiceId, nil
        } else {
            return "", fmt.Errorf("%s is not exist", resourceName)
        }
    case resourceType == "route-table":
        routeTableRef := clusterScope.RouteTablesRef()
        if routeTableId, ok := routeTableRef.ResourceMap[resourceName]; ok {
            return routeTableId, nil
        } else {
            return "", fmt.Errorf("%s is not exist", resourceName)
        }
    case resourceType == "route":  
        routeRef := clusterScope.RouteRef()
        if routeId, ok := routeRef.ResourceMap[resourceName]; ok {
            return routeId, nil
        } else {
            return "", fmt.Errorf("%s is not exist", resourceName)
        }
    default:
        return "", fmt.Errorf("%s does not exist", resourceType)
    }
}        


func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
    clusterScope.Info("Reconcile OscCluster")
    osccluster := clusterScope.OscCluster
    servicesvc := service.NewService(ctx, clusterScope)
    clusterScope.Info("Get Service", "service", servicesvc)

    clusterScope.Info("Create Loadbalancer")
    loadBalancerSpec := clusterScope.LoadBalancer()
    loadBalancerSpec.SetDefaultValue()
    loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, err
    }
    if loadbalancer == nil {
    	_, err := servicesvc.CreateLoadBalancer(loadBalancerSpec)
	if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    	}
        _, err = servicesvc.ConfigureHealthCheck(loadBalancerSpec)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not configure healthcheck for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        } 
    }
    netsvc := net.NewService(ctx, clusterScope)
    clusterScope.Info("Get net", "net", netsvc)

    clusterScope.Info("Create Net")
    netSpec := clusterScope.Net()
    netSpec.SetDefaultValue()
    netRef := clusterScope.NetRef()
    netName := "cluster-api-net-" + clusterScope.UID()
    if len(netRef.ResourceMap) == 0 {
        netRef.ResourceMap = make(map[string]string)
    }
    var netIds = []string{netRef.ResourceMap[netName]}
    net, err := netsvc.GetNet(netIds)
    clusterScope.Info("### len nets ###", "net", len(netRef.ResourceMap))
    clusterScope.Info("### Get net ###", "net", net)
    clusterScope.Info("### Get netIds ###", "net", netIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if net == nil {
        clusterScope.Info("### Empty Net ###")
        net, err = netsvc.CreateNet(netSpec, netName)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        netRef.ResourceMap[netName] = *net.NetId
        clusterScope.Info("### content updatee net ###", "net", netRef.ResourceMap)

    }
    netRef.ResourceMap[netName] = *net.NetId
    clusterScope.Info("Info net", "net", net)

    clusterScope.Info("Create Subnet")
    subnetSpec := clusterScope.Subnet()
    subnetSpec.SetDefaultValue()
    subnetRef := clusterScope.SubnetRef()   
    subnetName := "cluster-api-subnet-" + clusterScope.UID()
    if len(subnetRef.ResourceMap) == 0 {
        subnetRef.ResourceMap = make(map[string]string)
    }
    var subnetIds = []string{subnetRef.ResourceMap[subnetName]}
    subnet, err := netsvc.GetSubnet(subnetIds)
    clusterScope.Info("### len subnet ###", "subnet", len(subnetRef.ResourceMap))
    clusterScope.Info("### Get subnet ###", "subnet", subnet)
    clusterScope.Info("### Get subnetIds ###", "subnet", subnetIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if subnet == nil {
        clusterScope.Info("### Empty Subnet ###") 
        subnet, err = netsvc.CreateSubnet(subnetSpec, netRef.ResourceMap[netName], subnetName)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create subnet for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        subnetRef.ResourceMap[subnetName] = *subnet.SubnetId
        clusterScope.Info("### content update subnet ###", "subnet", subnetRef.ResourceMap)
    }

    clusterScope.Info("Create InternetGateway")
    internetServiceSpec := clusterScope.InternetService()
    internetServiceRef := clusterScope.InternetServiceRef()
    internetServiceName := "cluster-api-internetservice-" + clusterScope.UID()
    if len(internetServiceRef.ResourceMap) == 0 {
        internetServiceRef.ResourceMap = make(map[string]string)
    }
    var internetServiceIds = []string{internetServiceRef.ResourceMap[internetServiceName]}
    internetService, err := netsvc.GetInternetService(internetServiceIds)
    clusterScope.Info("### len internetService ###", "internetservice", len(internetServiceRef.ResourceMap))
    clusterScope.Info("### Get internetService ###", "internetservice",  internetService)
    clusterScope.Info("### Get internetServiceIds ###", "internetservice",  internetServiceIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if internetService == nil {
        clusterScope.Info("### Empty internetService ###")
        internetService, err = netsvc.CreateInternetService(internetServiceSpec, internetServiceName)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create internetservice for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        err = netsvc.LinkInternetService(*internetService.InternetServiceId, netRef.ResourceMap[netName])
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not link internetService with net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        internetServiceRef.ResourceMap[internetServiceName] = *internetService.InternetServiceId
        clusterScope.Info("### content update internetService ###", "internetservice", internetServiceRef.ResourceMap)

    }

    clusterScope.Info("Create PublicIp")
    var publicIpsSpec *[]infrastructurev1beta1.OscPublicIp
    networkSpec := clusterScope.Network()
    if networkSpec.PublicIps == nil {
        networkSpec.SetPublicIpDefaultValue()
        publicIpsSpec = &networkSpec.PublicIps
    } else {
        publicIpsSpec = clusterScope.PublicIp()
    }
    publicIpRef := clusterScope.PublicIpRef()
    var publicIpsId []string
    for _, publicIpSpec := range *publicIpsSpec {
        publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
        publicIpsId = []string{publicIpRef.ResourceMap[publicIpName]}
        if len(publicIpRef.ResourceMap) == 0 {
            publicIpRef.ResourceMap = make(map[string]string)
        }
        publicIp, err := netsvc.GetPublicIp(publicIpsId)
        if err !=nil {
            return reconcile.Result{}, err
        }
        if publicIp == nil {
            publicIp, err = netsvc.CreatePublicIp(&publicIpSpec, publicIpName)
            if err != nil {
                return reconcile.Result{}, errors.Wrapf(err, "Can not create publicIp for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
            }
        }
        publicIpRef.ResourceMap[publicIpName] = *publicIp.PublicIpId
        clusterScope.Info("### content update publicIpName ###", "publicip", publicIpRef.ResourceMap)
    } 


    clusterScope.Info("Create RouteTable")
    routeTablesSpec := clusterScope.RouteTables()
    routeTablesRef := clusterScope.RouteTablesRef()
    routeRef := clusterScope.RouteRef()
    linkRouteTablesRef := clusterScope.LinkRouteTablesRef()
    var routeTableIds []string
    var resourceIds []string
    for _, routeTableSpec := range *routeTablesSpec {
        routeTableName := routeTableSpec.Name + "-" + clusterScope.UID()
        routeTableIds = []string{routeTablesRef.ResourceMap[routeTableName]}    
        if len(routeTablesRef.ResourceMap) == 0 {
            routeTablesRef.ResourceMap = make(map[string]string)
        }
        if len(linkRouteTablesRef.ResourceMap) == 0 {
            linkRouteTablesRef.ResourceMap = make(map[string]string)
        }
        routeTable, err := netsvc.GetRouteTable(routeTableIds)
        clusterScope.Info("### Get routeTable ###", "routeTable", routeTable)
        clusterScope.Info("### Get routeTableIds ###", "routeTable",  routeTableIds)
        if err != nil {
            return reconcile.Result{}, err
        }
        if routeTable == nil {
            clusterScope.Info("### Empty routeTable ###")
            routeTable, err = netsvc.CreateRouteTable(&routeTableSpec, netRef.ResourceMap[netName], routeTableName)
            if err != nil {
                return reconcile.Result{}, errors.Wrapf(err, "Can not create routetable for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
            }
            linkRouteTableId, err := netsvc.LinkRouteTable(*routeTable.RouteTableId, subnetRef.ResourceMap[subnetName])
            if err != nil {
                return reconcile.Result{}, errors.Wrapf(err, "Can not link routetable with net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
            }
            routeTablesRef.ResourceMap[routeTableName] = *routeTable.RouteTableId
            linkRouteTablesRef.ResourceMap[routeTableName] = linkRouteTableId
            clusterScope.Info("### content update routeTable ###", "routeTable", routeTablesRef.ResourceMap)

            clusterScope.Info("check route")
            if len(routeRef.ResourceMap) == 0 {
                routeRef.ResourceMap = make(map[string]string)
            }
            routesSpec := clusterScope.Route(routeTableName)
            for _, routeSpec := range *routesSpec {
                resourceName := routeSpec.TargetName + "-" + clusterScope.UID()
                resourceType := routeSpec.TargetType
                routeName := routeSpec.Name + "-" + clusterScope.UID()
                resourceId, err := GetResourceId(resourceName, resourceType, clusterScope)
                if err != nil {
                    return reconcile.Result{}, err
                }
                resourceIds = []string{resourceId}
                destinationIpRange := routeSpec.Destination
                routeTableFromRoute, err := netsvc.GetRouteTableFromRoute(routeTableIds, resourceIds, resourceType)
                if err != nil {
                    return reconcile.Result{}, err
                }
                if routeTableFromRoute == nil {
                    routeTableFromRoute, err = netsvc.CreateRoute(destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
                    if err != nil {
                        return reconcile.Result{}, errors.Wrapf(err, "Can not create route for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
                    }
                }
                routeRef.ResourceMap[routeName] = *routeTableFromRoute.RouteTableId

            }     
        }
    }

    clusterScope.Info("Create NatService")
    natServiceSpec := clusterScope.NatService()
    natServiceRef := clusterScope.NatServiceRef()
    natServiceSpec.SetDefaultValue() 
    natServiceName := natServiceSpec.Name + clusterScope.UID()
    if len(natServiceRef.ResourceMap) == 0{
        natServiceRef.ResourceMap = make(map[string]string)
    }
    var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
    natService, err := netsvc.GetNatService(natServiceIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    publicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()

    if natService == nil {
        clusterScope.Info("### Empty NatService ###")
        clusterScope.Info("### Request Info ###", "natService", natServiceSpec.PublicIpName)
        clusterScope.Info("### Request Info ###", "natService", publicIpRef.ResourceMap[publicIpName])
        clusterScope.Info("### Request Info ###", "natService", subnetRef.ResourceMap[subnetName])
        clusterScope.Info("### Request Info ###", "natService", subnetName)
        clusterScope.Info("### Request Info ###", "natService", natServiceName)

        natService, err = netsvc.CreateNatService(natServiceSpec, publicIpRef.ResourceMap[publicIpName], subnetRef.ResourceMap[subnetName], natServiceName)
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not create natservice for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        natServiceRef.ResourceMap[natServiceName] = *natService.NatServiceId
        clusterScope.Info("### content update natService ###", "natservice", natServiceRef.ResourceMap)
    }

    controllerutil.AddFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
    clusterScope.Info("Set OscCluster status to ready")
    clusterScope.SetReady()
    return reconcile.Result{}, nil
}

func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
    clusterScope.Info("Reconcile OscCluster")
    osccluster := clusterScope.OscCluster
    servicesvc := service.NewService(ctx, clusterScope)
    clusterScope.Info("Get Service", "service", servicesvc)
    netRef := clusterScope.NetRef()

    clusterScope.Info("Delete LoadBalancer")
    loadBalancerSpec := clusterScope.LoadBalancer()
    loadBalancerSpec.SetDefaultValue()
    netName := "cluster-api-net-" + clusterScope.UID()
    var netIds = []string{netRef.ResourceMap[netName]}
    loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, err
    }
    if loadbalancer == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = servicesvc.DeleteLoadBalancer(loadBalancerSpec)
    if err != nil {
        return reconcile.Result{}, errors.Wrapf(err, "Can not delete load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }
    netsvc := net.NewService(ctx, clusterScope)
    clusterScope.Info("Get Net", "net", netsvc)

    clusterScope.Info("Delete natService")
    natServiceSpec := clusterScope.NatService()
    natServiceRef := clusterScope.NatServiceRef()
    natServiceName :=  natServiceSpec.Name + clusterScope.UID()
    var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
    natservice, err := netsvc.GetNatService(natServiceIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if natservice == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = netsvc.DeleteNatService(natServiceRef.ResourceMap[natServiceName])
    if err != nil {
         return reconcile.Result{}, errors.Wrapf(err, "Can not delete natService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }
    
    clusterScope.Info("Delete PublicIp")
    var publicIpsSpec *[]infrastructurev1beta1.OscPublicIp
    networkSpec := clusterScope.Network()
    if networkSpec.PublicIps == nil {
        networkSpec.SetPublicIpDefaultValue()
        publicIpsSpec = &networkSpec.PublicIps
    } else {
        publicIpsSpec = clusterScope.PublicIp()
    }
    publicIpRef := clusterScope.PublicIpRef()
    var publicIpsId []string
    for _, publicIpSpec := range *publicIpsSpec {
        publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
        publicIpsId = []string{publicIpRef.ResourceMap[publicIpName]}
        publicIp, err := netsvc.GetPublicIp(publicIpsId)
        if err != nil {
            return reconcile.Result{}, err
        }
        if publicIp == nil {
            controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
            return reconcile.Result{}, nil
        }       
     }


    clusterScope.Info("Delete RouteTable")
    routeTablesSpec := clusterScope.RouteTables()
    routeTablesRef := clusterScope.RouteTablesRef()
    linkRouteTablesRef := clusterScope.LinkRouteTablesRef()
    var routeTableIds []string
    var resourceIds []string
    for _, routeTableSpec := range *routeTablesSpec {
        routeTableSpec.SetDefaultValue()
        routeTableName := routeTableSpec.Name + "-" + clusterScope.UID()
        routeTableIds = []string{routeTablesRef.ResourceMap[routeTableName]}
        clusterScope.Info("### content delete routeTable ###", "routeTable", routeTablesRef.ResourceMap)
        routetable, err := netsvc.GetRouteTable(routeTableIds)
        clusterScope.Info("### delete routeTable ###", "routeTable", routetable)
        if err != nil {
            return reconcile.Result{}, err
        }
        if routetable == nil {
            controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
            return reconcile.Result{}, nil
        }
        clusterScope.Info("Remove Route")
        routesSpec := clusterScope.Route(routeTableName)
        for _, routeSpec := range *routesSpec {
            resourceName := routeSpec.TargetName + "-" + clusterScope.UID()
            resourceType := routeSpec.TargetType
            routeName := routeSpec.Name + "-" + clusterScope.UID()
            resourceId, err := GetResourceId(resourceName, resourceType, clusterScope)
            if err != nil {
                return reconcile.Result{}, err
            }
            routeTableId, err := GetResourceId(routeName, "route", clusterScope)
            if err != nil {
                return reconcile.Result{}, err
            }
            resourceIds = []string{resourceId}
            destinationIpRange := routeSpec.Destination
            routeTableFromRoute, err := netsvc.GetRouteTableFromRoute(routeTableIds, resourceIds, resourceType)
            if err != nil {
                return reconcile.Result{}, err
            }
            if routeTableFromRoute == nil {
                controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
                return reconcile.Result{}, nil                 
            }
            clusterScope.Info("Delete Route")
            err = netsvc.DeleteRoute(destinationIpRange, routeTableId)
            if err != nil {
                return reconcile.Result{}, errors.Wrapf(err, "Can not delete route for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
            }
        }
        clusterScope.Info("Unlink Routetable")
        
        err = netsvc.UnlinkRouteTable(linkRouteTablesRef.ResourceMap[routeTableName])
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not delete routeTable for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
        clusterScope.Info("Delete RouteTable")

        err = netsvc.DeleteRouteTable(routeTablesRef.ResourceMap[routeTableName])
        if err != nil {
            return reconcile.Result{}, errors.Wrapf(err, "Can not delete internetService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
        }
    }

    clusterScope.Info("Delete internetService")
    internetServiceRef := clusterScope.InternetServiceRef()
    internetServiceName := "cluster-api-internetservice-" + clusterScope.UID()
    var internetServiceIds = []string{internetServiceRef.ResourceMap[internetServiceName]}
    internetservice, err := netsvc.GetInternetService(internetServiceIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if internetservice == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = netsvc.UnlinkInternetService(internetServiceRef.ResourceMap[internetServiceName], netRef.ResourceMap[netName])
    if err != nil {
         return reconcile.Result{}, errors.Wrapf(err, "Can not unlink internetService and net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }
    err = netsvc.DeleteInternetService(internetServiceRef.ResourceMap[internetServiceName])
    if err != nil {
         return reconcile.Result{}, errors.Wrapf(err, "Can not delete internetService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }
 
    clusterScope.Info("Delete subnet")
    subnetRef := clusterScope.SubnetRef()
    subnetName := "cluster-api-subnet-" + clusterScope.UID()
    var subnetIds = []string{subnetRef.ResourceMap[subnetName]}
    subnet, err := netsvc.GetSubnet(subnetIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if subnet == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = netsvc.DeleteSubnet(subnetRef.ResourceMap[subnetName])
    if err != nil {
         return reconcile.Result{}, errors.Wrapf(err, "Can not delete subnet for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }

    clusterScope.Info("Delete net")
    net, err := netsvc.GetNet(netIds)
    if err != nil {
        return reconcile.Result{}, err
    }
    if net == nil {
        controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
        return reconcile.Result{}, nil
    }
    err = netsvc.DeleteNet(netRef.ResourceMap[netName])
    if err != nil {
        return reconcile.Result{}, errors.Wrapf(err, "Can not delete net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
    }
    controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
    return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscCluster{}).
		Complete(r)
}
