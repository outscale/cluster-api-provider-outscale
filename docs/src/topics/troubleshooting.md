# Troubleshooting Guide

Common issues that you might see.

### Missing credentials

Please set your credentials
```bash
 kubectl logs -f  cluster-api-provider-outscale-controller-manager-9f8dd7d8bqncnb  -n cluster-api-provider-outscale-system  
```

You will get:
```
1.6630978127864842e+09	ERROR	controller.oscmachine	Reconciler error	{"reconciler group": "infrastructure.cluster.x-k8s.io", "reconciler kind": "OscMachine", "name": "capo-quickstart-md-0-tpjgs", "namespace": "default", "error": "environment variable OSC_ACCESS_KEY is required failed to create Osc Client"}
```


### Override Limit 

Please check you have enough core, ram, instance quota.

Otherwise you will get:
```bash
controller.osccluster	Reconciler error	{"reconciler group": "infrastructure.cluster.x-k8s.io", "reconciler kind": "OscCluster", "name": "cluster-api-test", "namespace": "default", "error": "400 Bad Request Can not create net for Osccluster default/cluster-api-test"}
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).processNextWorkItem
	/go/pkg/mod/sigs.k8s.io/controller-runtime@v0.11.2/pkg/internal/controller/controller.go:266
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).Start.func2.2
	/go/pkg/mod/sigs.k8s.io/controller-runtime@v0.11.2/pkg/internal/controller/controller.go:227
```


### Node are not ready

Node are not ready because they need cni to be ready.


### Not running Node
If your vm is never in running phase and but still in provisonned phase, please look at the cloud init log of your vm.


### Trouble with e2etest path
You should clean a previous installation before launching e2etest:
```
make uninstall-clusterapi
kubectl get crd -A | grep x-k8s.
```

You should delete all cluster-api's CRD.
If deletion is blocked, you can use Finalizers to delete crds by patching them one by one
```
kubectl patch --namespace=my-namespace my-object my-object--name --patch='{"metadata":{"finalizers":null}}' --type=merge

```
### Clean Stack
If your vm did not reach running state, you can use:
```bash
ClusterToClean=my-cluster-name make testclean
```
to clean you stack

If there is some cluster-api k8s object (such as oscMachineTemplate) still remaning after running the cleaning script, please do:
```
kubectl delete oscmachinetemplate --all -A
kubectl patch --namespace=my-namespace oscmachinetemplate my-object-name --patch='{"metadata":{"finalizers":null}}' --type=merge
```
