# Cluster-template
There are a relationship between controller
 
# Configuration

## cluster infrastructure controller OscCluster
exemple:
```
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: hello-osc
  namespace: default
spec:
  network:
    loadBalancer:
      loadbalancername: OscSdkExample-7
      subregionname: eu-west-2a
    net:
      name: cluster-api-net
      ipRange: "172.19.95.128/25"
    subnets:
      - name: cluster-api-subnet
        ipSubnetRange: "172.19.95.192/27"
    publicIps:
      - name: cluster-api-publicip
    internetService:
      name: cluster-api-internetservice
    natService:
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
| `backendport`| `OscClusterApi-1` | false | The port on which the backend vm will listen
| `backendprotocol` | `eu-west-2a` | false | The protocol ('HTTP'|'TCP') to route the traffic to the backend vm
| `loadbalancerport` | `` | false | The port on which the loadbalancer will listen
| `loadbalancerprotocol` | `` | false | the routing protocol ('HTTP'|'TCP')

#### HealthCheck

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `checkinterval`| `OscClusterApi-1` | false | the time in second between two pings
| `healthythreshold` | `eu-west-2a` | false | the consecutive number of pings which are sucessful to consider the vm healthy
| `port` | `` | false |  the HealthCheck port number
| `protocol` | `` | false | The HealthCheck protocol ('HTTP'|'TCP')
| `timeout` | `` | false | the Timeout to consider VM unhealthy

### Net

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-net` | false | the tag name associated with the Net
| `ipRange` | `172.19.95.128/25` | false | Net Ip range with CIDR notation


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


### natService

| Name |  Default | Required | Description
| --- | --- | --- | ---
| `name`| `cluster-api-natservice` | false | The tag name associated with the Nat Service
| `publicIpName` | `cluster-api-publicip` | false | The Public Ip tag name associated wtih a Public Ip
| `subnetName`| `cluster-api-subnet` | false | The subnet tag name associated with a Subnet

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


