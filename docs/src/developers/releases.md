
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


# Release
### Versioning
Please use this semantic version:
- Pre-release: `v0.1.1-alpha.1`
- Minor release: `v0.1.0`
- Patch release: `v0.1.1`
- Major release: `v1.0.0`

### Update metadata.yaml
You should have update metadata.yaml to included new release version for cluster-api contract-version. You don't have to do it for patch/minor version.
Add in metadata.yaml:
```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
...
  - major: 1
    minor: 5
    contract: v1beta1
```
### Update config test
Please also update `type: InfrastructureProvider` spec of config.

### Create a tag
Create a new branch for release.
:warning: Never use the main
And create tag

For patch/major release:
```bash
git checkout release-1.x
git fetch upstream
git rebase upstream/release-1.x
```

Create tag with git:
```bash
export RELEASE_TAG=v1.2.3
git tag -s ${RELEASE_TAG} -m "${RELEASE_TAG}
git push upstream ${RELEASE_TAG}
```

This will trigger this github action [release][release]
This github action will generate image, and will create the new release.

### Test locally
If you want to test locally what is done by github action you can test you get changelog:
```bash
make release
make release-changelog
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
[release]: https://github.com/outscale-dev/cluster-api-provider-outscale/blob/main/.github/workflows/release.yml 
