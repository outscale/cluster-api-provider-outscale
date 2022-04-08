
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

```
git clone https://github.com/outscale-vbr/cluster-api-provider-outscale
cd cluster-api-provider-outscale
```

## User Credentials configuration 
```
export OSC_ACCESS_KEY=my-osc-access-key
export OSC_SECRET_KEY=my-osc-secret-key
cat config/secret.yaml | \
    sed "s/secret_key: \"\"/secret_key: \"$OSC_SECRET_KEY\"/g" | \
    sed "s/access_key: \"\"/access_key: \"$OSC_ACCESS_KEY\"/g" > osc-secret.yaml
/usr/local/bin/kubectl delete -f osc-secret.yaml --namespace=cluster-api-provider-outscale-system 
/usr/local/bin/kubectl apply -f osc-secret.yaml --namespace=cluster-api-provider-outscale-system 
```

## Registry credentials configuration

By default, if you use a private registry with credentials, registry credentials must be named regcred and must be deployed in cluster-api-provider-outscale-system namespace.

```
kubectl get secret regcred  -n cluster-api-provider-outscale-system 
NAME      TYPE                             DATA   AGE
regcred   kubernetes.io/dockerconfigjson   1      52s
```

If you want to change it with another name, you change change it in *cluster-api-provider-outscale/config/default/kustomization.yaml*:
```
      value: [{ name: regcred }]
```


# Build
##  Build and push image
This step will build and push image to your public or private registry
```
IMG=my-registry/controller:my-tag make docker-build
IMG=my-registry/controller:my-tag make docker-push
```

# Deploying Cluster Api

Please look at [cluster-api][cluster-api] section about deployment of cert-manager and cluster-api

# Deploying  Cluster API Provider Outscale

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
You can watch controller log:
```
kubectl logs -f cluster-api-provider-outscale-controller-manager-7d5c48d67t6d7f  -n cluster-api-provider-outscale-system  -c manager
```

## Create your cluster

This step will create your infrastructure cluster. It create vpc, net, sg, routetables, eip, nat.
You can change parameter from cluster-template.yaml if you need:
```
kubectl apply -f example/cluster-template.yaml
```


# CleanUp

##  Delete cluster

You can delete your cluster cluster with:
```
kubectl delete -f example/cluster-template.yaml
```

## Delete Cluster API Outscale controller manager

You can delete the Cluster Api Outscale controller manager with:
```
IMG=my-registry/controller:my-tag make undeploy
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

