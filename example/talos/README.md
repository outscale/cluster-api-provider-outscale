# Prerequisite

- [Terraform](https://www.terraform.io/downloads) (>= 0.14)
- [Outscale Access Key and Secret Key](https://docs.outscale.com/en/userguide/Creating-an-Access-Key.html)
- [Wireguard](https://www.wireguard.com)

# Requirements


Please clone the project:
```
git clone https://github.com/outscale-dev/cluster-api-provider-outscale
```

You can initialize the credentials which is used by the clusterctl with::
```
export OSC_ACCESS_KEY=<your-access-key>
export OSC_SECRET_KEY=<your-secret-access-key>
export OSC_REGION=<your-region>
make credential
./bin/clusterctl init --infrastructure outscale
```

Please create an your own omi using this docs (https://www.talos.dev/v1.6/talos-guides/install/cloud-platforms/aws/)

Pease apply talos:
```
kubectl apply -f talos.yaml
```

# Configuration

```
export TF_VAR_access_key_id="myaccesskey"
export TF_VAR_secret_key_id="mysecretkey"
export TF_VAR_region="eu-west-2"
```

By editing ['terraform.tfvars'](terraform.tfvars), you will set the net_id and control_plane_subnet_id.

WARNING: Make sure that `net_id` corresponds to your cluster and the omi bootstrapper-talos is currently only available on eu-west-2. You can create your own omi with [talos-aws](https://www.talos.dev/v1.5/talos-guides/install/cloud-platforms/aws/).

# Deploying bastion

This step will create bastion components as well as configuration files needed to initialize bastion.

```
terraform init
terraform apply
```

# Initialize Bastion

Get wireguard config
```
terraform refresh
terraform output
```
Save in /etc/wireguard/wg0.conf
Launch vpn:
```
wg-quick up wg0
```




# Deploy CCM

To initialize the cluster, please launch cloud-provider-outscale [Cloud Controller Manager (CCM)](https://github.com/outscale/cloud-provider-osc/blob/OSC-MIGRATION/deploy/README.md)


# Cleaning Up bastion

And then destroy the bastion: 
```
terraform destroy
```
