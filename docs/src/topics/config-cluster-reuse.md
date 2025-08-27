# Configuring a cluster with existing resources

You may reuse some resources.

CAPOSC expects that the resources are properly configured and will not check them. CAPOSC will not reconcile those resources and will not delete them when the cluster is deleted.

The following can be reused :
* a net,
* security groups,
* public IPs.

## Reused a net

When configuring a cluster with a reused net, CAPOSC expects the following to exist:
* an optional bastion,
* a net and its subnets,
* an internet service,
* one or more NAT services,
* route tables.

The load balancer currently cannot be reused.

The following needs to be specified:
* net and subnet resource IDs,
* subnet roles,
* net and subnet IP ranges.

```yaml
useExisting:
  net: true
net:
  resourceId: vpc-xxx
  ipRange: 10.0.0.0/16
subnets:
- resourceId: subnet-xxx
  roles:
  - loadbalancer
- resourceId: subnet-xxx
  ipSubnetRange: 10.0.2.0/24
  subregionName: eu-west-2a
  roles:
  - controlplane
- resourceId: subnet-xxx
  ipSubnetRange: 10.0.3.0/24
  subregionName: eu-west-2b
  roles:
  - controlplane
- resourceId: subnet-xxx
  ipSubnetRange: 10.0.4.0/24
  subregionName: eu-west-2a
  roles:
  - worker
- resourceId: subnet-xxx
  ipSubnetRange: 10.0.5.0/24
  subregionName: eu-west-2b
  roles:
  - worker
```

> Notes:
> * NAT and bastion subnets do not need to be specified, as they are not needed in the remaining configuration.
> * Routes tables, Internet and NAT services do not need to be specified.

## Reusing security groups

Resource IDs and the associated roles will need to be specified fot the following security groups roles:
* loadbalancer,
* worker,
* controlplane.

```yaml
useExisting:
  securityGroups: true
securityGroups:
- resourceId: sg-xxx
  roles:
  - loadbalancer
- resourceId: sg-xxx
  roles:
  - controlplane
- resourceId: sg-xxx
  roles:
  - worker
- resourceId: sg-xxx
  roles:
  - worker
  - controlplane
```

## Reusing public IPs

You may use predefined public IPs for:
* NAT services (outbound trafic),
* worker nodes (inbound trafic).

IPs must belong to a pool, and will configure the cluster or the machine template to fetch IPs from pools.
To add a public IP to a pool, you will need to set a `OscK8sIPPool` tag with the name of the pool as a value.

Configuring NAT services (OscCluster):
```yaml
network:
  natPublicIpPool: <name of pool>
```

Configuring worker nodes (OscMachineTemplate):
```yaml
vm:
  publicIpPool: <name of pool>
```
