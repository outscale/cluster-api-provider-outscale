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


### E2e test failed to launch
If e2e test doesn't launch when you run it:
```
clusterctl init --core cluster-api --bootstrap kubeadm --control-plane kubeadm --infrastructure outscale
```
and launched after and failed because it can not communicate with your cluster:
```
clusterctl config cluster capo-yayi5z --infrastructure (default) --kubernetes-version v1.22.11 --control-plane-machine-count 1 --worker-machine-count 1 --flavor with-clusterclass
```

You can launch yourself.
Please use this config (~/.cluster-api/clusterctl.yaml):
```yaml

CLUSTER_NAME: clusterctl
CLUSTER_NAMESPACE: default
CONTROL_PLANE_MACHINE_TEMPLATE_UPGRADE_TO: cp-k8s-upgrade-and-conformance
KUBERNETES_VERSION: v1.24.1
KUBERNETES_VERSION_UPGRADE_FROM: v1.23.6
KUBERNETES_VERSION_UPGRADE_TO: v1.24.1
OSC_NET_NAME: clusterctl-net
WORKERS_MACHINE_TEMPLATE_UPGRADE_TO: worker-k8s-upgrade-and-conformance
overridesFolder: /root/cluster-api-provider-outscale/artifact/repository/overrides
providers:
- name: cluster-api
  type: CoreProvider
  url: /root/cluster-api-provider-outscale/artifact/repository/cluster-api/v1.1.4/components.yaml
- name: kubeadm
  type: BootstrapProvider
  url: /root/cluster-api-provider-outscale/artifact/repository/bootstrap-kubeadm/v1.1.4/components.yaml
- name: kubeadm
  type: ControlPlaneProvider
  url: /root/cluster-api-provider-outscale/artifact/repository/control-plane-kubeadm/v1.1.4/components.yaml
- name: outscale
  type: InfrastructureProvider
  url: /root/cluster-api-provider-outscale/artifact/repository/infrastructure-outscale/v0.1.99/components.yaml
```
and launch this command before relaunch e2etest:
```
clusterctl init --core cluster-api --bootstrap kubeadm --control-plane kubeadm --infrastructure outscale
```

### Clean Stack

Please set the cluster name that you use:
```
export ClusterToClean=hello-osc
```
If your vm never reach running, you can use 
```bash
ClusterToClean=my-cluster-name make testclean
```
to clean you stack
