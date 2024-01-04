# Cluster-template
There are a relationship between controller
 
# Configuration

## cluster infrastructure controller OscCluster
example without bastion:

```
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: hello-osc
  namespace: default
spec:
  network:
    bastion:
      enable: false
    clusterName: cluster-api
    subregionName: eu-west-2a
    loadBalancer:
      loadbalancername: OscSdkExample-7
      subregionname: eu-west-2a
    net:
      name: cluster-api-net
      clusterName: cluster-api
      ipRange: "172.19.95.128/25"
    subnets:
      - name: cluster-api-subnet
        ipSubnetRange: "172.19.95.192/27"
    publicIps:
      - name: cluster-api-publicip

    internetService:
      clusterName: cluster-api
      name: cluster-api-internetservice
    natService:
      clusterName: cluster-api
      name: cluster-api-natservice
      publicipname: cluster-api-publicip
      subnetname: cluster-api-subnet
    routeTables:
      - name: cluster-api-routetable
        subnetname: cluster-api-subnet
        routes:
          - name: cluster-api-routes
            targetName: cluster-api-internetservice
            targetType: gateway 
            destination: "0.0.0.0/0"
    securityGroups:
      - name: cluster-api-securitygroups
        description: Security Group with cluster-api   
        securityGroupRules:
          - name: cluste-api-securitygrouprule
            flow: Inbound
            ipProtocol: tcp
            ipRange: "46.231.147.5/32"
            fromPortRange: 22
            toPortRange: 22 
```

example with bastion:

```
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name:  cluster-api
  namespace: default
spec:
  network:
    clusterName: cluster-api
    loadBalancer:
      loadbalancername: cluster-api-lb
      clusterName: cluster-api
      loadbalancertype: internet-facing
      subnetname: cluster-api-subnet
      securitygroupname: cluster-api-securitygroup-lb
    net:
      name: cluster-api-net
      clusterName: cluster-api-az
      ipRange: "10.0.0.0/16"
    internetService:
      name: cluster-api-igw
      clusterName: cluster-api
    controlPlaneSubnets:
      - cluster-api-subnet
    subnets:
    - name: cluster-api-subnet
      ipSubnetRange: "10.0.0.0/24"
      subregionName: eu-west-2a
    natServices:
    - name: cluster-api-nat
      clusterName: cluster-api
      publicipname: cluster-api-publicip
      subnetname: cluster-api-subnet
    publicIps:
      - name: cluster-api-publicip
        clusterName: cluster-api
    routeTables:
    - name: cluster-api-rtb
      subnets:
      - cluster-api-subnet
      routes:
      - name: cluster-api-nat
        targetName: cluster-api-nat
        targetType: nat
        destination: "0.0.0.0/0"
    securityGroups:
    - name: cluster-api-securitygroup-lb
      description: Cluster-api Load Balancer Security Group
      securityGroupRules:
      - name: cluster-api-securitygrouprule-calico-vxlan
        flow: Inbound
        ipProtocol: tcp
        ipRange: "0.0.0.0/0"
        fromPortRange: 6443
        toPortRange: 6443
    bastion:
      clusterName: cluster-api
      enable: true
      name: cluster-api-vm-bastion
      keypairName: cluster-api
      deviceName: /dev/sda1
      imageName: ubuntu-2004-2004-kubernetes-v1.22.11-2022-08-22
      rootDisk:
        rootDiskSize: 15
        rootDiskIops: 1000
        rootDiskType: io1
      subnetName: cluster-api-subnet-public
      subregionName: eu-west-2a
      securityGroupNames:
        - name: cluster-api-securitygroup-lb
      vmType: "tinav6.c4r8p2"

```

### loadBalancer

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `loadbalancername`| `OscClusterApi-1` | false | The Load Balancer  unique name 
| `subregionname` | `eu-west-2a` | false | The SubRegion Name where the Load Balancer will be created
| `listener` | `` | false | The Listener Spec
| `healthcheck` | `` | false | The healthcheck Spec


#### Listener

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `backendport`| `6443` | false | The port on which the backend vm will listen
| `backendprotocol` | `TCP` | false | The protocol ('HTTP'|'TCP') to route the traffic to the backend vm
| `loadbalancerport` | `6443` | false | The port on which the loadbalancer will listen
| `loadbalancerprotocol` | `TCP` | false | the routing protocol ('HTTP'|'TCP')

#### HealthCheck

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `checkinterval`| `30` | false | the time in second between two pings
| `healthythreshold` | `10` | false | the consecutive number of pings which are sucessful to consider the vm healthy
| `unhealthythreshold` | `5` | false | the consecutive number of pings which are failed to consider the vm unhealthy
| `port` | `6443` | false |  the HealthCheck port number
| `protocol` | `TCP` | false | The HealthCheck protocol ('HTTP'|'TCP')
| `timeout` | `5` | false | the Timeout to consider VM unhealthy


### Bastion

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `clusterName`| `cluster-api` | false | The cluster name
| `enable`| `false` | false | Enable to have bastion
| `name` | `cluster-api-vm-bastion` | false | The name of the bastion
| `imageName` | `tcp` | false |  the omi
| `keypairName` | `cluster-api` | false |  The keypair name used to access bastion
| `deviceName` | `/dev/sda1` | false |  The device name
| `rootDiskSize` | `15` | false |  The Root Disk Size
| `rootDiskIops` | `1000` | false |  The Root Disk Iops (only for io1)
| `rootDiskType` | `io1` | false |  The Root Disk Type (io1, gp2, standard)
| `subnetName` | `cluster-api-subnet-public` | false |  The Subnet associated to your bastion
| `subregionName` | `eu-west-2a` | false | The subregionName used for bastion and volume
| `securityGroupNames` | `cluster-api-securitygroup-lb` | false | The securityGroupName which is associated with bastion
| `vmType` | `tinav6.c2r4p2` | false |  The vmType use for the bastion

### Net

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-net` | false | the tag name associated with the Net
| `ipRange` | `172.19.95.128/25` | false | Net Ip range with CIDR notation
| `clusterName` | `cluster-api` | false | Name of the cluster
| `subregionName` | `eu-west-2a` | false | The subregionName used for vm and volume

### controlPlaneSubnets

List of subnet to spread controlPlane nodes


### Subnet

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-subnet` | false | The tag name associated with the Subnet
| `ipSubnetRange` | `172.19.95.192/27` | false | Subnet Ip range with CIDR notation

### publicIps

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-publicip` | false | The tag name associated with the Public Ip

### internetService

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-internetservice` | false | The tag name associated with the Internet Service
| `clusterName` | `cluster-api` | false | Name of the cluster


### natService

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-natservice` | false | The tag name associated with the Nat Service
| `publicIpName` | `cluster-api-publicip` | false | The Public Ip tag name associated wtih a Public Ip
| `subnetName`| `cluster-api-subnet` | false | The subnet tag name associated with a Subnet
| `clusterName` | `cluster-api` | false | Name of the cluster

### natServices

List of natServices 

You can have either list of natService (natServices) or one natService (natService)

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-natservice` | false | The tag name associated with the Nat Service
| `publicIpName` | `cluster-api-publicip` | false | The Public Ip tag name associated wtih a Public Ip
| `subnetName`| `cluster-api-subnet` | false | The subnet tag name associated with a Subnet
| `clusterName` | `cluster-api` | false | Name of the cluster

### routeTables

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-routetable` | false | The tag name associated with the Route Table
| `subnetName` | `cluster-api-subnet` | false | The subnet tag name associated with a Subnet
| `route` | `` | false | The route configuration

### route

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-route` | false | The tag name associated with the Route
| `targetName` | `cluster-api-internetservice` | false |  The tag name associated with the target resource type
| `targetType` | `gateway` | false |  The target resource type which can be Internet Service (gateway) or Nat Service (nat-service)
| `destination` | `0.0.0.0/0` | false |  the destination match Ip range with CIDR notation


### securityGroup

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-securitygroup` | false | The tag name associate with the security group
| `description` | `Security Group with cluster-api` | false | The description of the security group
| `securityGroupRules` | `` | false | The securityGroupRules configuration



### securityGroupRule

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-securitygrouprule` | false | The tag name associate with the security group
| `flow` | `Inbound` | false | The flow of the security group (inbound or outbound)
| `ipProtocol` | `tcp` | false |  The ip protocol name (tcp, udp, icmp or -1)
| `ipRange` | `46.231.147.5/32` | false |  The ip range of the security group rule
| `fromPortRange` | `6443` | false |  The beginning of the port range
| `toPortRange` | `6443` | false |  The end of the port range

## machine infrastructure controller OscCluster
example:
```
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscMachineTemplate
metadata:
  name: "cluster-api-md-0"
  namespace: default
  annotations:
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size: "5"
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size: "0"
spec:
  template:
    spec:
      node:
        clusterName: cluster-api
        image:
          name: ubuntu-2004-2004-kubernetes-v1.22.11-2022-08-22
        keypair:
          name: cluster-api
        vm:
          clusterName: cluster-api
          name: cluster-api-vm-kw
          keypairName: cluster-api
          deviceName: /dev/sda1
          rootDisk:
            rootDiskSize: 30
            rootDiskIops: 1500
            rootDiskType: io1
          subnetName: cluster-api-subnet-kw
          subregionName: eu-west-2a
          securityGroupNames:
            - name: cluster-api-securitygroups-kw
          vmType: "tinav6.c2r4p2"
```

### OscImage

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `` | false | The image name you will use

### OscKeypair

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `keypairName`| `cluster-api-keypair` | false | The keypairname you will use
| `destroyKeypair`| `false` | false | Destroy keypair at the end

### OscVm

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `clusterName`| `cluster-api` | false | The cluster name
| `name` | `cluster-api-vm-kw` | false | The name of the vm
| `keypairName` | `cluster-api` | false |  The keypair name used to access vm
| `deviceName` | `cluster-api` | false |  The device path to mount root volumes
| `rootDiskSize` | `30` | false |  The Root Disk Size
| `rootDiskIops` | `1500` | false |  The Root Disk Iops (only for io1)
| `rootDiskType` | `io1` | false |  The Root Disk Type (io1, gp2, standard)
| `rootDiskType` | `io1` | false |  The Root Disk Type (io1, gp2, standard)
| `subnetName` | `cluster-api-subnet-kw` | false |  The Subnet associated to your vm
| `subregionName` | `eu-west-2a` | false | The subregionName used for vm and volume
| `securityGroupNames` | `cluster-api-securitygroups-kw` | false | The securityGroupName which is associated with vm
| `vmType` | `tinav6.c2r4p2` | false |  The vmType use for the vm
| `imageName` | `ubuntu-2004-2004-kubernetes-v1.22.11-2022-08-22` | false |  The vmType use for the vm