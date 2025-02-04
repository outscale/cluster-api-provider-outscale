
# Prerequisite 
- Install [kubectl][kubectl]
- Install [helm][helm]
- Outscale account with ak/sk [Outscale Access Key and Secret Key][Outscale Access Key and Secret Key]
- A Kubernetes cluster:
    - You can use a Vm with [kubeadm][kubeadm] or [Minikube][Minikube]. 
    - You can use a container with [kind][kind]. 
    - You can use a rke cluster with [osc-rke][osc-rke].
- Container registry to store container image
- Registry secret [registry-secret][registry-secret]

# Deploying Cluster Api Outscale Charts

You can choose namespace and control-plane tag value is composed of helm release with ' -controller-manager'.
Create namespace:
Ex:
```
kubectl create namespace cluster-api-outscale-system
kubectl label namespaces cluster-api-outscale-system control-plane=cluster-api-outscale-controller-manager --overwrite=true
```

Install charts with your helm release name:
ex:
```
helm upgrade --install cluster-api-provider-outscale -n cluster-api-outscale-system ./helm/clusterapioutscale/
```

# CleanUp

##  Delete clusterapioutscale charts
ex:
```
helm uninstall cluster-api-provider-outscale   -n cluster-api-outscale-system
```

<!-- References -->
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kubeadm]: https://kubernetes.io/fr/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
[Outscale Access Key and Secret Key]: https://wiki.outscale.net/display/EN/Creating+an+Access+Key
[osc-rke]: https://github.com/outscale-dev/osc-k8s-rke-cluster
[Minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/
[helm]: https://github.com/helm/helm/releases
[kind]: https://github.com/kubernetes-sigs/kind#installation-and-usage
