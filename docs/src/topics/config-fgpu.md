# Addinf fGPU to nodes

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
