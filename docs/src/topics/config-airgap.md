# Configuring an air-gapped cluster

You might want to configure an air-gapped cluster, i.e., a cluster without internet access.

An air-gapped cluster will need:
* an internal load-balancer,
* a net peering between the management VPC and the workload VPC, allowing access to the load-balancer and nodes,
* a disabled Internet Service and disabled NAT Services,
* preloaded images on the workload nodes,
* access to the Outscale API on the nodes.

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

## Disabling Internet Service and NAT Services

```yaml
network:
    disable:
    - internet
```

## Preloaded images

See [Preloading images](preload.md).

## Outscale API access

You need to add Net Access Point to all the Outscale services you use.

API and LBU access are required for the CCM and CSI to work.

The list of services you might meed:
* `api`: API access
* `directlink`: [Direct link](https://docs.outscale.com/en/userguide/About-DirectLink.html)
* `eim`: [Identity Management](https://docs.outscale.com/en/userguide/About-EIM.html)
* `kms`
* `lbu`: AWS LBU gateway
*  `oos`: [Object Storage](https://docs.outscale.com/en/userguide/About-OOS.html)

```yaml
network:
    netAccessPoints:
    - api
    - lbu
```

> The cluster needs to be able to call the Outscale services. CAPOSC is currently unable to configure SecurityGroupRules to services, you will need to have a Outbound rule allowing access to 0.0.0.0/0.