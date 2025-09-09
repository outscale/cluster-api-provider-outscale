# Credentials and multitenancy

Credentials can be accessed in different ways:
* using the same credentials for all workload clusters, stored in a secret,
* ... or stored in a profile file,
* using different credentials for each clusters (multitenancy), stored in secrets,
* ... or stored in profile files. 

## Single tenant, using a secret

By default, CAPOSC credentials are read from a Kubernetes secret having 3 keys:
* access_key,
* secret_key,
* region.

```bash
export OSC_ACCESS_KEY=<your-access-key>
export OSC_SECRET_KEY=<your-secret-access-key>
export OSC_REGION=<your-region>

kubectl create secret generic cluster-api-provider-outscale --from-literal=access_key=$OSC_ACCESS_KEY --from-literal=secret_key=$OSC_SECRET_KEY --from-literal=region=$OSC_REGION  -n cluster-api-provider-outscale-system
```

## Single tenant, using a file

CAPOSC can also read credentials from a [profile file][profile file] stored in `/root/.osc/config.json`, using the `default` profile.

Profile files can be injected into the CAPOSC deployment using annotations, with the help of an agent (e.g., [Vault Agent Injector][Vault]).

> Note: Upgrading the infrastructure provider will reset the profile file, so you need to ensure it is re-injected afterward.

## Multitenant, using secrets

Each cluster can be deployed in its own Outscale account, the `OscCluster` resource storing the name of the secret with the credentials to use.

```yaml
spec:
    credentials:
        fromSecret: "foo-secret"
```

The secret follows the same structure as the standard secret but is stored in the same namespace as the cluster spec.

## Multitenant, using files

Either a single [profile file][profile file] can be used, storing one profile per account, or multiple files, each containing a single `default` profile.

Using a single file:
```yaml
spec:
    credentials:
        fromFile: "/root/.osc/config.json"
        profile: "foo"
```

Using multiple files:
```yaml
spec:
    credentials:
        fromFile: "/root/.osc/foo.json"
```

> Note: Upgrading the infrastructure provider will reset the profile files, so you must ensure they are re-injected afterward.

<!-- References -->
[profile file]: https://github.com/outscale/oapi-cli#-configuration
[Vault]: https://developer.hashicorp.com/vault/docs/deploy/kubernetes/injector
