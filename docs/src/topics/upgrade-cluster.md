# Upgrading a cluster

Before upgrading, ensure that the Kubernetes/etcd/CoreDNS versions you want to upgrade to are [supported by your installed providers][version support].

## Upgrading CCM

Before upgrading a cluster, the CCM needs to be upgraded to the target Kubernetes version.

## Upgrading control plane

You must start by upgrading the control plane nodes. For more information, refer to the [Updating Machine Infrastructure and Bootstrap Templates] [Updating Machine Infrastructure and Bootstrap Templates] documentation.

* Create new control plane `OscMachineTemplate` based on the current one:
```bash
kubectl get oscmachinetemplate <name> -o yaml > file.yaml
```

* Change the resource name and the OMI version:
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
[...]
  name: cluster-api-control-plane-1-33
[...]
spec:
  template:
    spec:
      node:
[...]      
        image:
          name: ubuntu-2204-kubernetes-v1.33.6-2025-11-17
```

* Create new `OscMachineTemplate`:
```bash
kubectl apply -f file.yaml
```

* Edit the `KubeadmControlPlane` resource: 
```bash
kubectl edit kubeadmcontrolplane <name>
```

* Change the version and the infrastructure reference:
```yaml
apiVersion: controlplane.cluster.x-k8s.io/v1beta1
kind: KubeadmControlPlane
spec:
[...]
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
      kind: OscMachineTemplate
      name: cluster-api-control-plane-1-33
      namespace: default
...
  version: v1.23.6
```

The new control-plane machine/nodes should be created and the old ones should be deleted.

##  Upgrading worker nodes

After having upgraded the control-plane nodes, worker nodes can be upgraded ([Updating Machine Infrastructure and Bootstrap Templates][Updating Machine Infrastructure and Bootstrap Templates]).

* Create a new `OscMachineTemplate` based on existing one:
```bash
kubectl get oscmachinetemplate <name> -o yaml > file.yaml
```

* Change the resource name and the OMI version:
```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
[...]
  name: cluster-api-md-1-33
[...]
spec:
  template:
    spec:
      node:
[...]      
        image:
          name: ubuntu-2204-kubernetes-v1.33.6-2025-11-17
```

* Create new `OscMachineTemplate`:
```bash
kubectl apply -f file.yaml
```

* Edit machinedeployments
```bash
kubectl edit machinedeployments.cluster.x-k8s.io <name>
```

* Change the version and the infrastructure reference:
```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
spec:
[...]]
    spec:
[...]
      infrastructureRef:
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
        kind: OscMachineTemplate
        name: cluster-api-md-1-33
        namespace: default
      version: v1.33.6
```

* Delete the old `MachineSet`:
```
kubectl delete machinesets.cluster.x-k8s.io <name>
```
<!-- References -->
[version support]: https://cluster-api.sigs.k8s.io/reference/versions#supported-versions-matrix-by-provider-or-component
[Namespace stuck as Terminating, How I removed it]: https://stackoverflow.com/questions/52369247/namespace-stuck-as-terminating-how-i-removed-it
[Updating Machine Infrastructure and Bootstrap Templates]: https://cluster-api.sigs.k8s.io/tasks/updating-machine-templates
[Upgrading management and workload clusters]: https://cluster-api.sigs.k8s.io/tasks/upgrading-clusters
