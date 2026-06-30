
# Prerequisite 
- [kubectl][kubectl] installed
- [kustomize][kustomize] `v3.1.0+` installed
- [kind][kind] installed
- [octl][octl] installed
- Outscale account with ak/sk [Outscale Access Key and Secret Key][Outscale Access Key and Secret Key]

# Configure development cluster

## Configuration credentials

Configure an octl profile. It will be used by CAPOSC to create resources. 

## Create cluster

```shell
make setup-dev
```

This will bootstrap a Kind cluster, configure the docker registry and setup a `kind-caposc` kubectl context.

# Build and deploy

```shell
KIND_IMG_TAG=`date '+%Y%m%d-%H%M%S'` make deploy-dev
```

This will build the CAPOSC image, push it to the docker registry and deploy the CAPOSC manifest.

# Clean up

```shell
make cleanup-dev
```

This will delete the Kind cluster.

## Upgrading Calico

The `.github/actions/deploy_cluster` action and the E2E framewwork need to be based on the same Calico version.

When upgrading Calico in the action, you need to upgrade the E2E framework:

```shell
curl -L https://raw.githubusercontent.com/projectcalico/calico/vX.Y.Z/manifests/calico.yaml -o ./test/e2e/data/cni/calico/calico.yaml
```

<!-- References -->
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kustomize]: https://github.com/kubernetes-sigs/kustomize/releases
[kind]: https://github.com/kubernetes-sigs/kind#installation-and-usage
[Outscale Access Key and Secret Key]: https://wiki.outscale.net/display/EN/Creating+an+Access+Key
[octl]: https://github.com/outscale/octl
