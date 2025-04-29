# Configuring nodes

## Keypairs

Keypairs must be configured. CAPOSC does not create keypairs anymore.

## Adding controlplane nodes

A `OscMachineTemplate` resource must be created for controlplane nodes.

No need to define the node subregion, CAPI automaticaly assigns controlplane nodes to subregions (failure domains).

## Adding worker nodes

A `MachineDeployment` resource and an `OscMachineTemplate` resource must be created per worker node pool.

The node subregion needs to be configured.

## Adding volumes to nodes

By default, nodes use a single root volume (/dev/sda1). Additional volumes can be added to VM.

> Volumes are created unformatted. You will need to format the newly created volumes during cloud-init.

In your oscmachinetemplate definition, add the list of volumes required:

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

`OscMachineTemplate` resources define a `spec.template.spec.node` node definition, with 3 attributes: `image`, `vm` and `volumes`.

### image

| Name |  Required | Description
| --- | --- | ---
| `name` | false | The image name you will use
| `accountId` | false | The ID of the account owning the image

### vm

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `role` | `worker` | false |  The role of the vm (`controlplane` or `worker`)
| `vmType` | `tinav6.c4r8p1` | false |  The type of VM to use
| `imageId` | n/a | false |  The omi ID (unless image.name is used)
| `keypairName` | n/a | true |  The keypair name used to access vm
| `rootDiskSize` | `60` | false |  The root disk size
| `rootDiskType` | `io1` | false |  The root disk type (`io1`, `gp2` or `standard`)
| `rootDiskIops` | `1500` | false |  The root disk iops (only for the `io1` type)
| `subregionName` | n/a | false | The subregionName where the node will be deployed (required for workers, unused for controlplanes)
| `subnetName` | n/a | false | The name of the subnet where to deploy the VM
| `securityGroupNames` | n/a | false | The name of the security groups to associate the VM with

### volumes

`volumes`is a list of additional volumes.

| Name |  Required | Description
| --- | --- | ---
| `name` | false |  The volume name
| `device` | true |  The volume device (`/dev/sdX`)
| `size` | true |  The volume size
| `volumeType` | true |  The volume type (`io1`, `gp2` or `standard`)
| `iops` | false |  The volume iops (only for the `io1` type)

### Automatic mode

If not set, CAPOSC will search for a subnet having the right role in the specified subregion.

If not set, CAPOSC wil search for all security groups having the right role.