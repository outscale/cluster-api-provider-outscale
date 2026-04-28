# Adding fGPU to nodes

> Only worker nodes are allowed to have fGPUs attached.

## Subregion configuration

It is recommended to use multi-az node pools, using a `random` `subregionMode`:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
[...]
spec:
    template:
        spec:
            node:
                vm:
                    subregionMode: random
                    subregionNames:
                        - eu-west-2a
                        - eu-west-2b
                        - eu-west-2c
```

## fGPU configuration

The fGPU model need to be configured in the VM spec. Do not forget to check that the [fGPU model](https://docs.outscale.com/en/userguide/About-Flexible-GPUs.html#_models_of_fgpus) is compatible with the VM type you have chosen.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
[...]
spec:
    template:
        spec:
            node:
                vm:
                    fGPU:
                      model: nvidia-xxx
```

fGPUs are released when the node VM is deleted.

## Nvidia drivers

CAPOSC does not install Nvidia software (drivers, GPU operator). Please refer to the [Nvidia GPU Operator documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/overview.html) for more information.
