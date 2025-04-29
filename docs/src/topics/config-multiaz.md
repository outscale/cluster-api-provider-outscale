# Configuring Multi-AZ clusters

## Subnets

When setting `subregions`, three subnets are created per subregion:
* a public subnet, with the nat role (the first one having also loadbalancer and bastion roles),
* a controlplane subnet,
* a worker subnet.

You can define your own subnets instead. 

## Controlplane nodes

The list of failure domains used by the controlplane provider is based on the subregions of controlplane subnets.

You need either to set `subregions` or to create a controlpane subnet per subregion, and configure at least as many controlplane replicas as there are subregions.

## Worker nodes

A node pool (`MachineDeployment`/`OscMachineTemplate` pair) must be created per AZ.

## NAT

In automatic mode, CAPOSC expects to find a nat subnet for each subregion where controlplane/worker nodes will be deployed.

## Route tabes

In automatic mode, CAPOSC creates a route table per subnet, routing traffic through the NAT service in the same subregion.

## Using clusterctl

A clusterctl template may be used to build a multiaz cluster on 3 subregions (a, d and c):

```bash
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_REGION=<osc-region>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>

clusterctl generate cluster <cluster-name> --kubernetes-version <kubernetes-version> --control-plane-machine-count=<control-plane-machine-count> --worker-machine-count=<worker-machine-count> --flavor=multiaz > getstarted.yaml
```

> Note: The template adds <worker-machine-count> nodes per subregion.