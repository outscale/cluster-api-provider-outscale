
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

Please clone the project

```
git clone https://github.com/outscale-dev/cluster-api-provider-outscale
```

## User Credentials configuration 

This step wil deploy user credential secret 
Put your credentials in osc-secret.yaml and apply:
```
/usr/local/bin/kubectl apply -f osc-secret.yaml
```

## Registry credentials configuration

By default, if you use a private registry (docker registry, harbor, dockerhub, quay.io, ....)  with credentials, registry credentials must be named regcred and must be deployed in cluster-api-provider-outscale-system namespace.

```
kubectl get secret regcred  -n cluster-api-provider-outscale-system 
NAME      TYPE                             DATA   AGE
regcred   kubernetes.io/dockerconfigjson   1      52s
```

If you want to change it with another name, you change it in *cluster-api-provider-outscale/config/default*:
```
      value: [{ name: regcred }]
```


# Build and  deploy
# Deploying Cluster Api

Please look at [cluster-api][cluster-api] section about deployment of cert-manager and cluster-api

##  Build, Push and Deploy
This step will build and push image to your public or private registry and deploy it.

### Environment variable

Set those environment variable with yours:
```
export K8S_CONTEXT=phandalin
export CONTROLLER_IMAGE=my-registry/controller
```
	
* K8S_CONTEXT is your context in your kubeconfig file.
	
* CONTROLLER_IMAGE is the project path where the image will be stored. Tilt will add a tag each time it build an new image.

### CAPM

Please run to generate capm.yaml:
```
IMG=my-registry/controller:latest make capm
```

* IMG is the CONTROLLER_IMAGE with CONTROLLER_IMAGE_TAG. Tilt will change the tag each time it build an new image.

### Tilt
Please launch tilt at the project's root folder:
```
[root@cidev-admin cluster-api-provider-outscale]# tilt up
Tilt started on http://localhost:10350/
v0.25.3, built 2022-03-04

(space) to open the browser
(s) to stream logs (--stream=true)
(t) to open legacy terminal mode (--legacy=true)
(ctrl-c) to exit
```

You can track your docker build and controller log in your web browser. 

## Check your cluster is deployed
```
[root@cidev-admin cluster-api-provider-outscale]# kubectl get pod -n cluster-api-provider-outscale-system
NAME                                                              READY   STATUS    RESTARTS   AGE
cluster-api-provider-outscale-controller-manager-7d5c48d67t6d7f   2/2     Running   0          22s
```

# Develop

## Using tiltfile

Choose your favorite editor to develop.

With the current tiltfile, each time you change a file, it build docker image, push into the registry and deploy it with kustomize.

## Create your cluster

This step will create your infrastructure cluster. It create vpc, net, sg, routetables, eip, nat.

You can change parameter from cluster-template.yaml (please look at [configuration][configuration]) if you need:
```
kubectl apply -f example/cluster-template.yaml
```

# Test 

## Generate Mock

Please use if you make some changes(add, changes, delete ) in *cloud/services/net*, *cloud/services/service*, *cloud/services/security*:
```
make mock-generate
```

## Unit test
Please use if you want to launch unit test:

```
make unit-test
```

Yòu can look at code coverage with covers.txt and covers.html

# CleanUp

##  Delete cluster

This step will delete your cluster 
```
kubectl delete -f example/cluster-template.yaml
```

## Delete Cluster Api Outscale controller manager

This step  will delete the outscale controller manager
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
[cluster-api]: https://cluster-api.sigs.k8s.io/developer/providers/implementers-guide/building_running_and_testing.html
[registry-secret]: https://kubernetes.io/fr/docs/tasks/configure-pod-container/pull-image-private-registry/
[configuration]: config.md

