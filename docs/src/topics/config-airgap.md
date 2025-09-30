# Configuring an air-gapped cluster

You might want to configure an air-gapped cluster, i.e., a cluster without internet access.

An air-gapped cluster will need:
* an internal load-balancer,
* a net peering between the management VPC and the workload VPC, allowing access to the load-balancer and nodes,
* a disabled Internet Service and disabled NAT Services,
* preloaded images on the workload nodes,
* access to the Outscale API on the workload nodes.

## Internal load-balancer

The load-balancer needs to be configured as internal:

```yaml
network:
    loadBalancer:
        loadbalancertype: "internal"
```

## NetPeering

Add a NetPeering beween the management VPC and the workload VPC:

```yaml
network:
    netPeering:
        enable: true
        managementCredentials:
            fromSecret: "osc-management"
```

By default, CAPOSC will:
* create a NetPeering using the OscCluster credentials,
* accept it using managementCredentials,
* add routes to all workload route tables to the management net,
* add routes to all management route tables to the workload net.

The management Net is identified by querying the metadata server from CAPOSC.

If the management node has no access to the metadata server, you need to configure the management net id:

```yaml
network:
    netPeering:
        enable: true
        managementCredentials:
            fromSecret: "osc-management"
        managementNetId: "vpc-xxx"
```

If you with to configure routes on a single management subnet, you can configure it:

```yaml
network:
    netPeering:
        enable: true
        managementCredentials:
            fromSecret: "osc-management"
        managementSubnetId: "subnet-xxx"
```

## Disabling Internet Service a NAT Services

TBD

## Pre-loaded images

TBD

## Outscale API access

TBD