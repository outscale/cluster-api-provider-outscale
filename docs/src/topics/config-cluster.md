# Configuring a cluster

A cluster is defined by a cluster resource and an infrastructure osccluster resource.

An osccluster resource defines:
* an optional bastion,
* a net and its subnets,
* an internet service,
* one or more NAT services,
* route tables,
* security groups,
* a load-balancer for the kubernetes API.

Roles define the role(s) a network resource may have:
* `loadbalancer` (kube API load-balancer),
* `nat` (NAT services),
* `bastion` (Bastion),
* `controlplane` (control plane nodes),
* `worker` (worker nodes),
* `service` (public service load-balancers),
* `service.internal` (internal service load-balancers).

Resources may be either automatically or manually created.

> Note: parameters that are not listed below are unused/deprecated.

## Net

### Automatic mode

A net with IP range 10.0.0.0/16 will be created.

### Manual mode

| Name |  Required | Description
| --- | --- | ---
| `name`| false | the name of the Net
| `ipRange` | true | the Ip range in CIDR notation

## Subnet

A subnet may have multiple roles.
Other resources will match a subnet either based on its name or its role.

When using roles:
* there must be a single subnet having the loadbalancer and bastion roles,
* there must be one subnet with the controlplane or worker role per subregion where the corresponding nodes will be deployed,
* there must be one subnet with the nat role per subregion in the cluster.

### Service load-balancers

When a service of type `LoadBalancer` is created, the CCM will configure a new LBU.

If the service is configured as internal, the CCM will fetch a subnet having the `service.internal` role.
Otherwise, a subnet having the `service` role is used.
If no subnet is found, the CCM fetches the subnet with the `loadbalancer` role.

### Automatic mode

CAPOSC automatically creates /24 subnets within the net IP range:
* one nat+loadbalancer+bastion subnet (public subnet) (10.0.2.0/24 if net is 10.0.0.0/16),
* one worker subnet (10.0.3.0/24 if net is 10.0.0.0/16),
* one controlplane subnet (10.0.4.0/24 if net is 10.0.0.0/16).

### Manual mode

| Name | Required | Description
| --- | --- | ---
| `name`| false | The name of the Subnet (required in roles are not used)
| `ipSubnetRange` | true | Subnet IP range in CIDR notation
| `roles` | false | The list of roles for this subnet (required if name is not used)

## Internet service

An internet service is automatically created. No need for manual configuration.

## NAT service

By default, NAT services use dynamically generated public IPs.

You can set `network.natPublicIpPool` with the name of a public IP pool to use predefined public IPs. See [Reusing public IPs](config-cluster-reuse.md) for more information.

### Automatic mode

One NAT service is created for each subnet having the `nat` role.

A public IP is automatically provisioned, no need to list it.

### Manual mode

| Name | Required | Description
| --- | --- | ---
| `name`| false | The name of the Nat Service
| `subnetName`| false | The name of the Subnet to which the NAT service will be attached

## Routing tables

### Automatic mode

One route table is created per subnet, with a default route to the internet service (public subnets) or to the NAT service of the same subregion (private subnets).

### Manual mode

| Name | Required | Description
| --- | --- | ---
| `name`| false | The name of the Route Table
| `subnetName` |  true | The subnet name
| `route` | true | The list of routes to add

Each route is defined by the following attributes:

| Name | Required | Description
| --- | --- | ---
| `targetName` | true |  The name of target resource (Internet Service or NAT Service)
| `targetType` | true |  The target resource type which can be Internet Service (`gateway`) or NAT Service (`nat` or `nat-service`)
| `destination` | true |  the destination IP range in CIDR notation

## Security Groups

Security Groups may have multiple roles.
Other resources will match a security group either based on its name or its role.

Rules may be added without setting `extraSecurityGroupRule`.
If a rule is removed from the spec, it won't be removed from the IaaS security group.

Security groups may be authoritative or not. If not set, rules may be added but will never be deleted by CAPOSC.
In authoritative mode, all rules not present in the spec will be deleted, including any default outbound rules.

### Automatic mode

5 security groups may be created:
* load-balancer (applied to the loadbalancer),
* controlplane (applied to controlplane nodes),
* worker (applied to worker nodes),
* node (applied to controlplane and worker nodes),
* bastion, only if enabled.

The default configuration is :

```yaml
- name: {cluster_name}-lb
  description: LB securityGroup for {cluster_name}
  authoritative: true
  roles:
  - loadbalancer
  rules:
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 6443
    toPortRange: 6443
    ipRanges:
    - 0.0.0.0/0
  - flow: Outbound
    ipProtocol: tcp
    fromPortRange: 6443
    toPortRange: 6443
    ipRanges:
    - {controlplane_subnet_ranges}
- name: {cluster_name}-worker
  description: Worker securityGroup for {cluster_name}
  authoritative: true
  roles:
  - worker
  rules:
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 30000
    toPortRange: 32767
    ipRanges:
    - {controlplane_subnet_ranges}
    - {worker_subnet_ranges}
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 10250
    toPortRange: 10250
    ipRanges:
    - {controlplane_subnet_ranges}
    - {worker_subnet_ranges}
- name: {cluster_name}-controlplane
  description: Controlplane securityGroup for {cluster_name}
  authoritative: true
  roles:
  - controlplane
  rules:
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 6443
    toPortRange: 6443
    ipRanges:
    - {net_range}
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 30000
    toPortRange: 32767
    ipRanges:
    - {controlplane_subnet_ranges}
    - {worker_subnet_ranges}
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 2378
    toPortRange: 2380
    ipRanges:
    - {controlplane_subnet_ranges}
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 10250
    toPortRange: 10252
    ipRanges:
    - {controlplane_subnet_ranges}
- name: {cluster_name}-node
  description: Controlplane securityGroup for {cluster_name}
  authoritative: true
  roles:
  - controlplane
  - worker
  rules:
  - flow: Inbound # ICMP
    ipProtocol: icmp
    fromPortRange: 8
    toPortRange: 8
    ipRanges:
    - {net_range}
  - flow: Inbound # BGP
    ipProtocol: tcp
    fromPortRange: 179
    toPortRange: 179
    ipRanges:
    - {net_range}
  - flow: Inbound # Calico VXLAN
    ipProtocol: udp
    fromPortRange: 4789
    toPortRange: 4789
    ipRanges:
    - {net_range}
  - flow: Inbound # Typha
    ipProtocol: udp
    fromPortRange: 5473
    toPortRange: 5473
    ipRanges:
    - {net_range}
  - flow: Inbound # Wiregard
    ipProtocol: udp
    fromPortRange: 51820
    toPortRange: 51821
    ipRanges:
    - {net_range}
  - flow: Inbound # Flannel
    ipProtocol: udp
    fromPortRange: 8285
    toPortRange: 8285
    ipRanges:
    - {net_range}
  - flow: Inbound # Flannel VXLAN
    ipProtocol: udp
    fromPortRange: 8472
    toPortRange: 8472
    ipRanges:
    - {net_range}
  - flow: Inbound # Cillium health
    ipProtocol: tcp
    fromPortRange: 4240
    toPortRange: 4240
    ipRanges:
    - {net_range}
  - flow: Inbound # Cillium hubble
    ipProtocol: tcp
    fromPortRange: 4244
    toPortRange: 4244
    ipRanges:
    - {net_range}
  - flow: Inbound # SSH, only if bastion is enabled
    ipProtocol: tcp
    fromPortRange: 22
    toPortRange: 22
    ipRanges:
    - {bastion_subnet_ranges}
  - flow: Outbound # default outbound rule
    ipProtocol: "-1"
    fromPortRange: -1
    toPortRange: -1
    ipRanges:
    - "0.0.0.0/0"
- name: {cluster_name}-bastion
  description: Bastion securityGroup for {cluster_name}
  authoritative: true
  roles:
  - bastion
  rules:
  - flow: Inbound
    ipProtocol: tcp
    fromPortRange: 22
    toPortRange: 22
    ipRanges:
    - 0.0.0.0/0
  - flow: Outbound
    ipProtocol: tcp
    fromPortRange: 22
    toPortRange: 22
    ipRanges:
    - {net_range}
  - flow: Outbound # default outbound rule
    ipProtocol: "-1"
    fromPortRange: -1
    toPortRange: -1
    ipRanges:
    - "0.0.0.0/0"
```

> Note: the default configuration is never rewritten to the spec.

### Adding rules in automatic mode

If you want to have add rules to the default automatic config, you can add them to `additionalSecurityRules`.

```yaml
network:
  additionalSecurityRules:
    - roles:
      - controlplane
      - worker
      rules:
      - flow: Inbound
        ipProtocol: tcp
        fromPortRange: 4240
        toPortRange: 4240
        ipRanges:
        - 10.0.3.0/24
```

For each `additionalSecurityRules` entry, a single security group is matched, having the sames roles in the same order, and rules will be added to it.

### Manual mode

| Name | Required | Description
| --- | --- | ---
| `name`| true | The name of the security group
| `description` | false | The description of the security group
| `authoritative` | false | Is the Security Group configuration authoritative ? (if yes, all rules not found in configuration will be deleted).
| `roles` | false | The list of roles this security group applies to
| `securityGroupRules` | false | The list of rules to apply

Each rule is defined by:

| Name |  Required | Description
| --- | --- | ---
| `flow` | true | The flow of the security group (`Inbound` or `Outbound`)
| `ipProtocol` | true|  The protocol (`tcp`, `udp`, `icmp` or `-1`)
| `ipRange` | false |  The ip range of the security group rule (deprecated, use `ipRanges`)
| `ipRanges` | false |  The list of ip ranges of the security group rule
| `fromPortRange` | true |  The beginning of the port range
| `toPortRange` | true |  The end of the port range

> Note: If you define your own security groups, `additionalSecurityRules` is ignored.

## Load balancer

### Automatic mode

A load balancer named `loadBalancer.loadbalancername` is created.

### Manual mode

| Name |  Required | Description
| --- | --- | ---
| `loadbalancername`| true | The Load Balancer  unique name 
| `listener` | false | The Listener Spec
| `healthcheck` | false | The healthcheck Spec


The listener has the following attributes:
| Name |  Default | Required | Description
| --- | --- | --- | ---
| `loadbalancerport` | `6443` | false | The frontend port

The health check has the following attributes:
| Name |  Default | Required | Description
| --- | --- | --- | ---
| `checkinterval`| `10` | false | The interval in seconds between two checks
| `healthythreshold` | `3` | false | The consecutive number of successful checks for a backend vm to be considered healthy
| `unhealthythreshold` | `3` | false | The consecutive number of failed checks for a backend vm to be considered unhealthy
| `timeout` | `10` | false | The timeout after which a check is considered unhealthy

## Bastion

### Automatic mode

By default, no bastion is created.

### Manual mode

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `enable`| `false` | false | Enable to have bastion
| `name` | n/a | true | The name of the bastion
| `imageId` | n/a | false |  the omi ID (imageId or imageName must be present)
| `imageName` | n/a | false |  the omi name (not recommended, use imageId)
| `keypairName` | n/a | true |  The keypair name used to access bastion
| `rootDiskSize` | `15` | false |  The Root Disk Size
| `rootDiskIops` | n/a | false |  The Root Disk Iops (only for io1)
| `rootDiskType` | `gp2` | false |  The Root Disk Type (io1, gp2, standard)
| `vmType` | `tinav6.c1r1p2` | false |  The vmType use for the bastion
