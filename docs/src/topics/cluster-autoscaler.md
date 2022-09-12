# Cluster Autoscaler  Guide

Use cluster-autoscaler
Please look at [cluster-api][cluster-api]  first.
### Install with helm
We will use kubeconfig-incluster mode

```bash
helm install --set 'autoDiscovery.clusterName=hello-osc' --set 'cloudProvider=clusterapi' --set 'clusterAPIKubeconfigSecret=hello-osc-kubeconfig' --set 'clusterAPIMode=kubeconfig-incluster' 

```
hello-osc is the clusterName
hello-osc-kubeconfig is the generated workload kubeconfig

### Add Labels
You need to have at least this annotations in each machineDeployment:
```yaml

  annotations:
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size: "5"
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size: "0"
```

<!-- References -->
[cluster-api]: https://cluster-api.sigs.k8s.io/tasks/automated-machine-management/autoscaling.html
