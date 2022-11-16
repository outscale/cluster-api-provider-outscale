
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

# Test 
:warning: In order to install tools (clusterctl, ...) with makefile, you need to have installed golang to download binaries [golang][golang]

## Lint
Please use format to indent your go and yamlfile:
```bash
make format
```

Lint go :
```bash
make golint-ci
make vet
```

Lint shell:
```bash
make shellcheck
```

Lint yaml:
```bash
make yamllint
```

boilerplate:
```bash
make verify-boilerplate
```

## Generate Mock

Please use if you want to mock functions described in cloud folder for unit test:
```
make mock-generate
```

## Unit test
Please use if you want to launch unit test:

```
make unit-test
```

Yòu can look at code coverage with covers.txt and covers.html

## Functional test

Please use if you want to launch functional test:
```
export OSC_ACCESS_KEY=<your-osc-acces-key>
export OSC_SECRET_KEY=<your-osc-secret-key>
export KUBECONFIG=<your-kubeconfig-path>
make testenv
```

## E2e test
Please use if you want to launch feature e2etest:
```
export OSC_ACCESS_KEY=<your-osc-acces-key>
export OSC_SECRET_KEY=<your-osc-secret-key>
export KUBECONFIG=<your-kubeconfig-path>
export IMG=<your-image>
make e2etestexistingcluster
```

Please use if you want to launch upgrade/remediation e2etest (it will use kind):
```
export OSC_ACCESS_KEY=<your-osc-acces-key>
export OSC_SECRET_KEY=<your-osc-secret-key>
export OSC_REGION=<your-osc-region>
export IMG=<your-image>
make e2etestkind
```

Please use if you want to launch conformance e2etest (it will use kind):
```
export OSC_ACCESS_KEY=<your-osc-acces-key>
export OSC_SECRET_KEY=<your-osc-secret-key>
export OSC_REGION=<your-osc-region>
export KUBECONFIG=<your-kubeconfig-path>
export IMG=<your-image>
make e2econformance
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
[golang]: https://medium.com/@sherlock297/install-and-set-the-environment-variable-path-for-go-in-kali-linux-446d0f16a338

