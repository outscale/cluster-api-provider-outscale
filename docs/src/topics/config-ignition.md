# Using Flatcar Linux

To use Flatcar Linux on Outscale, you need to build a Flatcar OMI and configure Ignition.

## Build Flatcar OMI

Follow the instructions in the [kubernetes-sigs/image-builder documentation](https://image-builder.sigs.k8s.io/capi/providers/3dsoutscale) on how to build an OMI
 for Flatcar.

```
make build-osc-flatcar
```

## Enable Ignition

In `KubeadmConfigTemplate`/`KubeadmControlPlane` resources:
```yaml
  spec:
    format: ignition
    preKubeadmCommands:
      - "hostnamectl set-hostname $(curl -s http://169.254.169.254/latest/meta-data/local-hostname)"
      - "while ! /opt/bin/crictl info >/dev/null 2>&1; do echo 'Waiting for containerd...'; sleep 2; done"
```

## Configure Flatcar OMI

In `OscMachineTemplate` resources - replace with your Flatcar OMI name:
```yaml
  spec:
    node:
      image:
        name: flatcar-kubernetes-v1.34.3
```
