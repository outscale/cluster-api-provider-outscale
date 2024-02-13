# Upgrade cluster

How to upgrade cluster and switch version 
## Matrix compatibility

There are a compatibility list between version ([version support][version support]).
So with the change of version, you can have to change Core Provider, Kubeadm Bootstrap Provider and Kubeadm Control Plane Provider.

|                   |  v0.3.0 (v1beta1)  |
| ---------------   | ------------------ |
| Kubernetes v1.22  |         ✓          |
| Kubernetes v1.23  |         ✓          |
| Kubernetes v1.24  |         ✓          | 
| Kubernetes v1.25  |         ✓          |
| Kubernetes v1.26  |         ✓          |
| Kubernetes v1.27  |         ✓          |
| Kubernetes v1.28  |         ✓          |


## Upgrade with clusterctl

Based on k8s compatibility, you will have to change operator with a specific version specially if you have to increase your cluster with multiple versions.

It is also possible that you will change several times cluster-api version if you upgrade from an old version of kubernetes (ex: v1.22) to a recent one (v1.28.5).

In order to do so, please follow the following steps.

* Delete cluster-api controllers

```
cluserctl delete --all
```

* Delete some cluster-api crd

```
kubectl delete crd ipaddressclaims.ipam.cluster.x-k8s.io ipaddresses.ipam.cluster.x-k8s.io
```

* Specify operator version

Please write in $HOME/.cluster-api/clusterctl.yaml :
```
providers:
- name: "kubeadm"
  url:  "https://github.com/kubernetes-sigs/cluster-api/releases/v1.3.10/bootstrap-components.yaml"
  type: "BootstrapProvider"
- name: "kubeadm"
  url: "https://github.com/kubernetes-sigs/cluster-api/releases/v1.3.10/control-plane-components.yaml"
  type: "ControlPlaneProvider"
- name: "cluster-api"
  url: "https://github.com/kubernetes-sigs/cluster-api/releases/v1.3.10/core-components.yaml"
  type: "CoreProvider" 
```

* Deploy operators

```
clusterctl init --infrastructure outscale
```
:warning: It is possible that cert-manager-test is stuck in terminating state

In order to force clean cert-manager-test namespace ([Namespace stuck as Terminating, How I removed it][Namespace stuck as Terminating, How I removed it]):
```
NAMESPACE=cert-manager-test
kubectl proxy &
kubectl get namespace $NAMESPACE -o json |jq '.spec = {"finalizers":[]}' >temp.json
curl -k -H "Content-Type: application/json" -X PUT --data-binary @temp.json 127.0.0.1:8001/api/v1/namespaces/$NAMESPACE/finalize
```

##  Upgrade control plane

We will first update control plane ([Updating Machine Infrastructure and Bootstrap Templates][Updating Machine Infrastructure and Bootstrap Templates])

* Create new template based on previous one:
```bash
kubectl get oscmachinetemplate <name> -o yaml > file.yaml
```

* Change the metadata name and change the omi version:
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
...
  name: cluster-api-control-plane-1-28
...
spec:
  template:
    spec:
      node:
...      
        image:
          name: ubuntu-2204-2204-kubernetes-v1.28.5-2024-01-10
```

* Create new templates:
```bash
kubectl apply -f file.yaml
```

* Edit kubeadmcontrolplane
```bash
kubectl edit kubeadmcontrolplane <name>
```

* Change version and infrastructure reference
```yaml
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
spec:
...
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
      kind: OscMachineTemplate
      name: cluster-api-control-plane-1-28
      namespace: default
...
  version: v1.28.5
```

**Warning**<br>
It is possible that old  control plane vm will only be deleted when you upgrade your first worker.


##  Upgrade worker

We will after upgrade worker ([Updating Machine Infrastructure and Bootstrap Templates][Updating Machine Infrastructure and Bootstrap Templates])

* Create new template based on previous one:
```bash
kubectl get oscmachinetemplate <name> -o yaml > file.yaml
```

* Change the metadata name and change the omi version:
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
...
  name: cluster-api-md-1-28
...
spec:
  template:
    spec:
      node:
...      
        image:
          name: ubuntu-2204-2204-kubernetes-v1.28.5-2024-01-10
```

* Create new templates:
```bash
kubectl apply -f file.yaml
```

* Edit machinedeployments
```bash
kubectl edit machinedeployments.cluster.x-k8s.io <name>
```

* Change version and infrastructure reference
```yaml
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
spec:
...
    spec:
...
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: OscMachineTemplate
        name: cluster-api-md-1-28
        namespace: default
      version: v1.28.5
```

**Warning**<br>
Depends on your strategy of rolling out ([Upgrading management and workload clusters][Upgrading management and workload clusters])


* Delete old machineset
```
kubectl delete machinesets.cluster.x-k8s.io <name>

```



<!-- References -->
[version support]: https://cluster-api.sigs.k8s.io/reference/versions#kubeadm-bootstrap-provider-kubeadm-bootstrap-controller
[Namespace stuck as Terminating, How I removed it]: https://stackoverflow.com/questions/52369247/namespace-stuck-as-terminating-how-i-removed-it
[Updating Machine Infrastructure and Bootstrap Templates]: https://cluster-api.sigs.k8s.io/tasks/updating-machine-templates
[Upgrading management and workload clusters]: https://cluster-api.sigs.k8s.io/tasks/upgrading-clusters
