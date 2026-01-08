# Configuring nodes

## Keypairs

Keypairs must be configured. CAPOSC does not create keypairs anymore.

## Adding controlplane nodes

A `OscMachineTemplate` resource must be created for controlplane nodes.

No need to define the node subregion, CAPI automatically assigns controlplane nodes to the subregions (failure domains) defined in the `OscCluster` spec.

## Adding worker nodes

A `MachineDeployment` resource and an `OscMachineTemplate` resource must be created per worker node pool.

The node subregion needs to be configured.

## Adding volumes to nodes

By default, nodes use a single root volume (/dev/sda1). Additional volumes can be added to VM.

In your `OscMachineTemplate` spec, add the list of required volumes:

```yaml
[...]
  node:
    vm: [...]
    volumes:
    - name: data
      device: /dev/xvdb
      iops: 500
      size: 50
      volumeType: io1
    - name: logs
      device: /dev/xvdc
      iops: 500
      size: 10
      volumeType: io1
[...]
```

> New volumes are unformatted. You will need to partition, format and mount them during cloud-init.

A snapshot can be used as a volume source:
```yaml
    - name: images
      device: /dev/xvdd
      fromSnapshot: snap-xxx
```

> By default, the size of the snapshot is used for the new volume.

You will need to mount snapshot-based volumes during cloud-init. In `KubeadmConfigTemplate`/`KubeadmControlPlane` resources:
```yaml
  spec:
    joinConfiguration:
      [...]
    mounts:
      - - xvdd
        - /mnt/example
        - ext4
        - auto,exec,ro
```

## OscMachineTemplate configuration

`OscMachineTemplate` resources include a `spec.template.spec.node` node definition, with three attributes: `image`, `vm`, `volumes` and `reconciliationRule`.

### `image`

| Name |  Required | Description
| --- | --- | ---
| `name` | no | The image name you will use
| `accountId` | no | The ID of the account owning the image
| `outscaleOpenSource` | no | Set to true if you use an Outscale Open Source image (requires CAPOSC v1.1.0)

Outscale Open-Source images are published on the `eu-west-2`, `us-east-2` and `cloudgouv-eu-west-1` regions with the same name. Please refer to the [Kubernetes Image Building Workflows repository][Kubernetes Image Building Workflows] for more information on those images.

### `vm`

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `role` | `worker` | no |  The role of the VM (`controlplane` or `worker`)
| `replica` | n/a | yes | The number of replicas for this node pool
| `vmType` | `tinav6.c4r8p1` | no |  The type of VM to use
| `imageId` | n/a | no |  The OMI ID (unless `image.name` is used)
| `keypairName` | n/a | yes |  The keypair name used to access vm
| `rootDiskSize` | `60` | no |  The root disk size
| `rootDiskType` | `io1` | no |  The root disk type (`io1`, `gp2` or `standard`)
| `rootDiskIops` | `1500` | no |  The root disk iops (only for the `io1` type)
| `subregionName` | n/a | no | The subregion where the node will be deployed (required for workers, unused for controlplanes)
| `subnetName` | n/a | no | The name of the subnet where to deploy the VM (not required if you have defined roles for your subnets)
| `securityGroupNames` | n/a | no | The name of the security groups to associate the VM with (not required if you have defined roles for your security groups)
| `publicIp` | false | no | Set to true if you want the node to have a public IP
| `publicIpPool` | n/a | no | Name of a public IP pool to use if you want the node to have a predefined public IP. See [Reusing public IPs](config-cluster-reuse.md) for more information (requires CAPOSC v1.1.0)
| `tags` | n/a | no | Additional tags to set on the VM

### `volumes`

`volumes` is a list of additional volumes.

| Name | Default | Required | Description
| --- | --- | --- | ---
| `name` | n/a | no |  The volume name
| `device` | n/a | yes |  The volume device (`/dev/xvdX`)
| `size` | n/a | no |  The volume size (required unless the source is a snapshot; if not set, the snapshot size is used)
| `volumeType` | `standard` | no |  The volume type (`io1`, `gp2` or `standard`)
| `iops` | n/a | no |  The volume iops (only for the `io1` type)
| `fromSnapshot` | n/a | no |  The ID of the source snapshot

## `reconciliationRule`

A reconciliation rule can be configured.

| Name | Default | Required | Description
| --- | --- | --- | ---
| `appliesTo`| n/a | yes | The list of reconcilers the rule applies to: `vm` or `*` (all reconcilers)
| `mode` | n/a | yes | `always` (always reconcile), `onChange` (reconcile only if the resource has changed) or `random` (onChange + a certain chance of reconciliation otherwise)
| `reconciliationChance` | n/a | no | The chance of reconciliation in `random` mode (a percentage from 0 to 100)

The default rule is:
```yaml
reconciliationRule:
  appliesTo: ['*']
  mode: onChange
```

### Subnet & security group selection

If not set, CAPOSC will use the subnet having the right role in the specified subregion.

If not set, CAPOSC will select all security groups having the right role.

> For compatibility purposes with v0.4 configs, `subnetName` and `securityGroupNames` can still be used but are deprecated.