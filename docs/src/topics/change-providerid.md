# How to migrate with new providerid

Until version v0.2.2, our providerId use the following format aws://availability-zone/instance-id

New version of cluster-api-provider-outscale will use the following format aws:///availability-zone/instance-id

This documentation is only for kubeadm bootstrapper.

## How to migrate

### Cluster-api provider outscale requirement

```
root@ip-10-9-24-171:/home/outscale# kubectl get deployment -n cluster-api-provider-outscale-system 
NAME                                               READY   UP-TO-DATE   AVAILABLE   AGE
cluster-api-provider-outscale-controller-manager   1/1     1            1           14h
```

Please make sure you have the version of the manager v0.2.2 in cluster-api-provider-outscale-controller-manager deployment

```
outscale/cluster-api-outscale-controllers:v0.2.2
```

### Control Plane requirement
Please make sure you have 3 replicas of kubeadmControlPlane.

In kubeadmControlPlane config:

```
kind: KubeadmControlPlane
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
...
spec:
  replicas: 3 
```

```
kubectl get machine -A
default        capu-quickstart-control-plane-bzmwp          capu-quickstart   ip-10-0-4-81                              aws:///eu-west-2a/i-8d5b8257   Running    11h   v1.22.11
default        capu-quickstart-control-plane-n75kk          capu-quickstart   ip-10-0-3-230                             aws:///eu-west-2a/i-c896acba   Running    11h   v1.22.11
default        capu-quickstart-control-plane-kgx4w          capu-quickstart   ip-10-0-4-62                              aws:///eu-west-2a/i-7db45201   Running    11h   v1.22.11
default        capu-quickstart-md-0-c54f966f8xjqbbf-pnhv9   capu-quickstart  ip-10-0-3-230.eu-west-2.compute.internal   aws:///eu-west-2a/i-298e40bc   Running    19h   v1.22.11
default        capu-quickstart-md-0-c54f966f8xjqbbf-x55lv   capu-quickstart   ip-10-0-3-10.eu-west-2.compute.internal    aws:///eu-west-2a/i-817cc83b   Running    11h   v1.22.11
```

### Worker requirement

Please make sure you have at least 2 replica of machineDeployment

```
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: "capu-quickstart-md-0"
  namespace: default
spec:
  clusterName: "capu-quickstart"
  replicas: 2
```


```
kubectl get machinedeployment -A
default        capu-quickstart-md-0   capu-quickstart   1                  1         1             ScalingUp   13h   v1.22.11
```

#### Change worker node

Please change manager image with the latest version in cluster-api-provider-outscale-controller-manager deployment:

```
image: outscale/cluster-api-outscale-controllers:v0.x.y
```

Please drain node [drain][drain].
ex:

```
kubectl drain --ignore-daemonsets ip-10-0-3-10.eu-west-2.compute.internal  
```
Replace in KubeadmConfigTemplate:

```
aws://
```

By:

```
aws:///
```

Then delete previous node

ex:
```
kubectl delete node capu-quickstart-md-0-c54f966f8xjqbbf-x55lv
```

A new node with the new providerId config will be available.

Please repeat this procedure for each node.

#### Change master node


Please drain [https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/] one node.

ex:

```
kubectl drain --ignore-daemonsets ip-10-0-4-62           
```

Replace in KubeadmControlPlane:

```
aws://
```

By:

```
aws:///
```

Then delete previous node

ex:

```
kubectl delete node capu-quickstart-control-plane-kgx4w
```

A new node with the new providerId config will be available.

Please repeat this procedure for each node.



<!-- References -->
[drain]: https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/
