# Kubernetes Cluster Api Provider Outscale

Kubernetes-native declarative infrastructure for Outscale

## What is the Cluster API Provider Outscale

The Cluster API is used to create, manage and configure cluster using a declarative Kubernetes-style APIs.

The Cluster API allows you to deploy kubernetes across multiple cloud provider. The cluster api provider outscale allows you a outscale deployment of kubernetes.

## Project Status

Thee project is Work in Progress,  in an Alpha state.


## Compatibility with Cluster API and Kubernetes Versions

The provider version has currently been tested with the following version of Cluster API:
|                                       | Cluster API v1 (v1.0) |
| ------------------------------------- | --------------------- |
| Outscale Provider v1beta1       (v0.1)| ✓                     |

The provider version has currently been tested with the following version of kubernetes:
|                 | Outscale Provider v1beta1 (v0.1) |
| --------------- | ------------------------------------- |
| Kubernetes 1.22 |  ✓                                    |

# Features

- Cluster infrastructure Outscale controller (https://cluster-api.sigs.k8s.io/developer/providers/cluster-infrastructure.html#cluster-infrastructure-provider-specification)

# To start using Outscale Cluster Api
Please look at [Deployment](docs/deploy.md)

# To start developping Outscale Cluster Api
Please look at [Development](docs/develop.md)

# Contribution
Please look at [Contribution](CONTRIBUTING.md)

# License

> Copyright Outscale SAS
>
> BSD-3-Clause
This project is compliant with [REUSE](https://reuse.software/).
