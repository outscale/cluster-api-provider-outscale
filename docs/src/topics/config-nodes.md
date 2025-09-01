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

> Volumes are created unformatted. You will need to format the newly created volumes during cloud-init.

In your `OscMachineTemplate` spec, add the list of volumes required:

```yaml
[...]
  node:
    vm: [...]
    volumes:
    - name: data
      device: /dev/sdb
      iops: 500
      size: 50
      volumeType: io1
    - name: logs
      device: /dev/sdc
      iops: 500
      size: 10
      volumeType: io1
[...]
```

## OscMachineTemplate configuration

`OscMachineTemplate` resources include a `spec.template.spec.node` node definition, with three attributes: `image`, `vm` and `volumes`.

### image

| Name |  Required | Description
| --- | --- | ---
| `name` | false | The image name you will use
| `accountId` | false | The ID of the account owning the image
| `outscaleOpenSource` | false | Set to true if you use an Outscale Open Source image (requires CAPOSC v1.1.0)

Outscale Open-Source images are published on the `eu-west-2`, `us-east-2` and `cloudgouv-eu-west-1` regions with the same name. Please refer to the [Kubernetes Image Building Workflows repository][Kubernetes Image Building Workflows] for more information on those images.

### vm

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `role` | `worker` | false |  The role of the VM (`controlplane` or `worker`)
| `replica` | n/a | yes | The number of replicas for this node pool
| `vmType` | `tinav6.c4r8p1` | false |  The type of VM to use
| `imageId` | n/a | false |  The OMI ID (unless `image.name` is used)
| `keypairName` | n/a | true |  The keypair name used to access vm
| `rootDiskSize` | `60` | false |  The root disk size
| `rootDiskType` | `io1` | false |  The root disk type (`io1`, `gp2` or `standard`)
| `rootDiskIops` | `1500` | false |  The root disk iops (only for the `io1` type)
| `subregionName` | n/a | false | The subregion where the node will be deployed (required for workers, unused for controlplanes)
| `subnetName` | n/a | false | The name of the subnet where to deploy the VM (not required if you have defined roles for your subnets)
| `securityGroupNames` | n/a | false | The name of the security groups to associate the VM with (not required if you have defined roles for your security groups)
| `publicIp` | false | false | Set to true if you want the node to have a public IP
| `publicIpPool` | n/a | false | Name of a public IP pool to use if you want the node to have a predefined public IP. See [Reusing public IPs](config-cluster-reuse.md) for more information (requires CAPOSC v1.1.0)
| `tags` | n/a | false | additional tags to set on the VM

### volumes

`volumes` is a list of additional volumes.

| Name |  Required | Description
| --- | --- | ---
| `name` | false |  The volume name
| `device` | true |  The volume device (`/dev/sdX`)
| `size` | true |  The volume size
| `volumeType` | true |  The volume type (`io1`, `gp2` or `standard`)
| `iops` | false |  The volume iops (only for the `io1` type)

### Subnet & security group selection

If not set, CAPOSC will use the subnet having the right role in the specified subregion.

If not set, CAPOSC will select all security groups having the right role.

> For compatibility purposes with v0.4 configs, `subnetName` and `securityGroupNames` can still be used but are deprecated.