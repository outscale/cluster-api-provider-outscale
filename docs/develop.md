
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

By default, if you use a private registry with credential, registry credential must be call regcred and must be deployed in cluster-api-provider-outscale-system namespace.

```
kubectl get secret regcred  -n cluster-api-provider-outscale-system 
NAME      TYPE                             DATA   AGE
regcred   kubernetes.io/dockerconfigjson   1      52s
```

If you want to change it with another name, you change change it in *cluster-api-provider-outscale/config/default*:
```
      value: [{ name: regcred }]
```


# Build and  deploy
##  Build, Push and Deploy
This step will build and push image to your public or private registry and deploy it.
Launch tilt at  the project's root folder.
```
[root@cidev-admin cluster-api-provider-outscale]# tilt up
Tilt started on http://localhost:10350/
v0.25.3, built 2022-03-04

(space) to open the browser
(s) to stream logs (--stream=true)
(t) to open legacy terminal mode (--legacy=true)
(ctrl-c) to exit
```
Watch docker build and controller log in your web browser. 

## Check your cluster is deployed
```
[root@cidev-admin cluster-api-provider-outscale]# kubectl get pod -n cluster-api-provider-outscale-system
NAME                                                              READY   STATUS    RESTARTS   AGE
cluster-api-provider-outscale-controller-manager-7d5c48d67t6d7f   2/2     Running   0          22s
```

# Devellop

## Using tiltfile
Choose your favorite editor to devellop.
With the current tiltfile, each time yu change a file, it build docker image, push into the registry and deploy it with kustomize.

## Create your cluster

This step will create your infrastructure cluster. It create vpc, net, sg, routetables,  eip, nat.
You can change parameter from cluster-template.yaml if you need:
```
kubectl apply -f example/cluster-template.yaml
```


# CleanUp

##  Delete cluster

You can delete your cluster cluster
```
kubectl delete -f example/cluster-template.yaml
```

## Delète Outscale controller manager

You will delete the outscale controller manager
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
