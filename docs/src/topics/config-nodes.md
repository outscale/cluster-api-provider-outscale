# Adding nodes

## Adding control-plane nodes

A `OscMachineTemplate` resource must be created for control plane nodes.

When adding subnets with the `controlplane` role on multiple subregions, control plane nodes will be automatically deployed on the specified subregions.

## Adding worker nodes

A `MachineDeployment` resource, an `OscMachineTemplate` resource must be created per worker node pool.

In a multi-AZ deployment, a `MachineDeployment`/`OscMachineTemplate` pool must be created per AZ.

## OscMachineTemplate configuration

`OscMachineTemplate` resources define a `spec.template.spec.node` node definition, with 2 attributes: `image` and `vm`.

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
| `subregionName` | n/a | false | The subregionName where the node will be deployed (worker nodes only, if none set, will be deployed on the cluster default subregion)
| `subnetName` | n/a | false | The name of the subnet where to deploy the VM
| `securityGroupNames` | n/a | false | The name of the security groups to associate the VM with

### Automatic mode

If not set, CAPOSC will search for a subnet having the right role in the specified subregion.

If not set, CAPOSC wil search for all security groups having the right role.