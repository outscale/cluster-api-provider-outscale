# Configuring a cluster with existing resources

## Reused resources

When configuring a cluster with esisting resources, CAPOSC expects the following to be created:
* an optional bastion,
* a net and its subnets,
* an internet service,
* one or more NAT services,
* route tables,
* security groups.

CAPOSC expects that the resources are properly configured and will not check them. CAPOSC will not reconcile those resources and will not delete them when the cluster is deleted.

If needed, CAPOSC may add rules to the existing security groups.

The load balancer currently cannot be reused.

## Configuration

Resource IDs and the associated roles for net, subnets and security groups will need to be configured:

```yaml
net:
  useExisting: true
  resourceId: vpc-xxx
subnets:
- resourceId: subnet-xxx
  roles:
  - loadbalancer
- resourceId: subnet-xxx
  subregionName: eu-west-2a
  roles:
  - controlplane
- resourceId: subnet-xxx
  subregionName: eu-west-2b
  roles:
  - controlplane
- resourceId: subnet-xxx
  subregionName: eu-west-2a
  roles:
  - worker
securityGroups:
- resourceId: sg-xxx
  roles:
  - loadbalancer
- resourceId: sg-xxx
  roles:
  - controlplane
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

> Notes:
> * NAT and bastion subnets do not need to be configured, as they are not needed in the remaining configuration.
> * Routes tables, Internet and NAT services do not need to be configured.
