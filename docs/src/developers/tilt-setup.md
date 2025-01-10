
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

### Install Tilt
If you want to install tilt with all dev tools:
```
make install-dev-prerequisites: 
```

Or if you want to install till only:

```
make install-tilt
```

### Tilt configuration:
You can either configure tilt with setting in your bashrc or profile:
```
export CONTROLLER_IMAGE=myregistry/osc/cluster-api-outscale-controllers:latest
export K8S_CONTEXT=cluster-api-dev
```
K8S_CONTEXT is the name of the cluster. (k8s context in kubeconfig).

CONTROLLER_IMAGE is the image controller (myregistry is the url of your registry, osc is the project, cluster-api-outscale-controllers is the name of the image,  latest  is the tag of the image )

Or you can set with tilt.config:
```
{
  "allowed_contexts": cluster-api-dev,
  "controller_image": myregistry/osc/cluster-api-outscale-controllers:latest,
}
```
### Tilt
We need SSH forwarding in Tilt to securely relay authentication credentials from your local machine to containers running in the Kubernetes cluster managed by Tilt. 
This enables you to access remote resources that require SSH authentication

A way to enable SSH forwarding in a single line is to do so :
```
ssh -f -N -i <private_key_file> <username>@<remote_host> -L <local_port>:<destination_host>:<destination_port>
```
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

