
# Release

## Prerequisites

You should have run the full e2e suite.

```bash
CCM_OSC_REGION=eu-west-2 KUBECONFIG=~/.kube/config IMG={repository host}/outscale/cluster-api-outscale-controllers IMG_UPGRADE_FROM=ami-29e3ca13 IMG_UPGRADE_TO=ami-92d61a16 E2E_FOCUS=e2e make e2etest
```

## Versioning
Please use this semantic version:
- Pre-release: `v0.1.1-alpha.1`
- Major release: `v1.0.0`
- Minor release: `v0.1.0`
- Patch release: `v0.1.1`

## Update metadata.yaml
Major/minor versions should register their contract version in metadata.yaml:
```yaml
apiVersion: clusterctl.cluster.x-k8s.io/v1alpha3
releaseSeries:
...
  - major: 1
    minor: 5
    contract: v1beta1
```

## Create a release branch
Create a new branch for each major/minor release.

```bash
export RELEASE_VERSION=1.2.3
git checkout release-${RELEASE_VERSION}
git fetch origin
git rebase origin/release-${RELEASE_VERSION}
```

For patch releases, use the minor release branch.

Push the release branch:
```bash
git push origin release-${RELEASE_VERSION}
```

## Create a release tag
Create & push tag:
```bash
export RELEASE_TAG=v1.2.3
git tag -s ${RELEASE_TAG} -m "🔖 ${RELEASE_TAG}"
git push origin ${RELEASE_TAG}
```

This will trigger the [release github action][release].
It will publish the docker image, and will create the new release.

## Test locally
If you want to test locally what is done by github action:
```bash
make release
make release-changelog
```

<!-- References -->
[release]: https://github.com/outscale-dev/cluster-api-provider-outscale/blob/main/.github/workflows/release.yml 
