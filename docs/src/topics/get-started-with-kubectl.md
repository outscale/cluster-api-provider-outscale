
# Prerequisite 
- Install [kubectl][kubectl]
- Install [kustomize][kustomize]  `v3.1.0+`
- Outscale account with ak/sk [Outscale Access Key and Secret Key][Outscale Access Key and Secret Key]
- A Kubernetes cluster:
    - You can use a Vm with [kubeadm][kubeadm] or [Minikube][Minikube]. 
    - You can use a container with [kind][kind]. 
    - You can use a rke cluster with [osc-rke][osc-rke].
- Container registry to store container image
- Registry secret [registry-secret][registry-secret]
# Configuration

## Clone

Please clone the project:
```
git clone https://github.com/outscale-dev/cluster-api-provider-outscale
```

## User Credentials configuration 
Put your ak/sk in osc-secret.yaml and launch:
```
/usr/local/bin/kubectl apply -f osc-secret.yaml
```

## Registry credentials configuration

### Public Outscale dockerhub

You can either use outscale/cluster-api-provider-outscale latest image.


###  Build and push own image

Or you can build and push image to your public or private registry

```
IMG=my-registry/controller:my-tag make docker-build
IMG=my-registry/controller:my-tag make docker-push
```

# Deploying Cluster Api

Please look at [cluster-api][cluster-api] section about deployment of cert-manager and cluster-api

Or you can use this to deploy cluster-api with cert-manager:
```
make deploy-clusterapi
```

# Deploying Cluster API Provider Outscale

# Deploy Cluster API Outscale controller manager
This step will deploy the Outscale  Cluster API controller manager (currently compose of only of the Cluster Infrastructure Provider controller)
```
IMG=my-registry/controller:my-tag make deploy
```

## Check your cluster is deployed
```
[root@cidev-admin cluster-api-provider-outscale]# kubectl get pod -n cluster-api-provider-outscale-system
NAME                                                              READY   STATUS    RESTARTS   AGE
cluster-api-provider-outscale-controller-manager-7d5c48d67t6d7f   2/2     Running   0          22s
```
##  Watch controller log
this step will watch controller log:
```
kubectl logs -f cluster-api-provider-outscale-controller-manager-7d5c48d67t6d7f  -n cluster-api-provider-outscale-system  -c manager
```

## Create your cluster

This step will create your infrastructure cluster. 

It will create vpc, net, sg, routetables, eip, nat.

You can change parameter from cluster-template.yaml (please look at [configuration][configuration]) if you need:
```
kubectl apply -f example/cluster-template.yaml
```

## Kubeadmconfig

You can use [bootstrap][bootstrap] to custom the bootstrap config.

Currently, we overwrite runc with containerd because of [containerd][containerd]

# CleanUp

##  Delete cluster

This step will delete your cluster cluster with:
```
kubectl delete -f example/cluster-template.yaml
```

## Delete Cluster API Outscale controller manager

This step will delete the Cluster Api Outscale controller manager with:
```
IMG=my-registry/controller:my-tag make undeploy
```

# Delete Cluster Api

Please look at [cluster-api][cluster-api] section about deployment of cert-manager and cluster-api

Or you can use this to undeploy cluster-api with cert-manager:

```
make deploy-clusterapi
```

<!-- References -->
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kustomize]: https://github.com/kubernetes-sigs/kustomize/releases
[kind]: https://github.com/kubernetes-sigs/kind#installation-and-usage
[kubeadm]: https://kubernetes.io/fr/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
[Outscale Access Key and Secret Key]: https://wiki.outscale.net/display/EN/Creating+an+Access+Key
[osc-rke]: https://github.com/outscale-dev/osc-k8s-rke-cluster
[Minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/
[registry-secret]: https://kubernetes.io/fr/docs/tasks/configure-pod-container/pull-image-private-registry/
[cluster-api]: https://cluster-api.sigs.k8s.io/developer/providers/implementers-guide/building_running_and_testing.html
[configuration]: config.md
[bootstrap]: https://cluster-api.sigs.k8s.io/tasks/kubeadm-bootstrap.html 
[containerd]: https://github.com/containerd/containerd/releases/tag/v1.6.0
