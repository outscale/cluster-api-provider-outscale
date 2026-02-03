# Changelog

## [v1.3.0] - 2026-02-04

This version adds:
* reconciliation rules, allowing reconciliation of resources without changes,
* repulseServer/repulseCluster placement constraints for nodes.

No changes since v1.3.0-rc.1

## [v1.3.0-rc.1] - 2026-01-28

### ✨ Added
* ✨ feat: add reconciliation rules by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/718
* ✨ feat(OscMachine): handle tag keys having already the tags prefix by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/731
* ✨ feat(OscMachine): add repulseServer/repulseCluster by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/729
### 🐛 Fixed
* 🐛 fix(OscCluster/OscMachine): abort reconciliation if owner Cluster/Machine has been deleted by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/732
### 📦 Dependency updates
* ⬆️ deps(gomod): update module sigs.k8s.io/cluster-api to v1.10.9 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/709
* ⬆️ deps(dockerfile): update golang docker tag to v1.25.5 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/702
* ⬆️ deps(gomod): update module github.com/onsi/ginkgo/v2 to v2.27.3 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/708
* ⬆️ deps(gomod): update module github.com/onsi/gomega to v1.38.3 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/705
* ⬆️ deps(gomod): update module sigs.k8s.io/cluster-api/test to v1.10.9 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/710
* ⬆️ deps(dockerfile): update golang:1.25.5 docker digest to 31c1e53 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/716
* ⬆️ deps(dockerfile): update golang:1.25.5 docker digest to 0f406d3 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/717
* ⬆️ deps(dockerfile): update python:3-bookworm docker digest to 96b5670 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/727
* ⬆️ deps(gomod): update module github.com/onsi/ginkgo/v2 to v2.27.5 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/719
* ⬆️ deps(dockerfile): update golang docker tag to v1.25.6 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/728

## [v1.2.0] - 2025-12-24

### 📦 Dependency updates
* ⬆️ deps(gomod): update module github.com/spf13/pflag to v1.0.10 by @Open-Source-Bot in https://github.com/outscale/cluster-api-provider-outscale/pull/706

## [v1.2.0-rc.1] - 2025-12-17

No changes since v1.2.0-beta.1

## [v1.2.0-beta.1] - 2025-12-03
### ✨ Added
* ✨ feat: add multitenancy by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/632
* ✨ feat(airgap): add netpeering to workload vpc by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/648
* ⚡️ perfs: add concurrency + tuning config flags by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/654
* ✨ feat(OscMachine): snapshot volume source by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/653
* ✨ feat(airgap): add image preloading by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/659
* ✨ feat: watch for a single namespace by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/666
* ✨ feat(airgap): disable internet/nat services, configure net access points by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/668
* ✨ feat(OscCluster): disable load-balancer by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/672
* ✨ feat: allow inference vm types for vm types validation by @ddavid-numspot in https://github.com/outscale/cluster-api-provider-outscale/pull/686
### 🛠️ Changed / Refactoring
* 👷 build: build binary with Go 1.25 by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/673
* 📈 api: use dev user-agent for CI calls by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/687
### 📝 Documentation
* 📝 doc: updates (installation, maturity level) by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/643
* 📄 licensing: fix licenses/reuse headers by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/647
* 📝 docs: fix dead links by @outscale-rce in https://github.com/outscale/cluster-api-provider-outscale/pull/658
* 📝 doc: improve version compatibility requirements in main/upgrade docs by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/684
* 🔧 eim: remove unused calls in example EIM policy by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/689
### 🗑️ Removed
* ⚰️ cleanup: remove runc in templates, already installed by image-builder by @jfbus in https://github.com/outscale/cluster-api-provider-outscale/pull/649
### 📦 Dependency updates
* build(deps): bump github.com/stretchr/testify from 1.10.0 to 1.11.1 by @dependabot[bot] in https://github.com/outscale/cluster-api-provider-outscale/pull/620
* build(deps): bump github.com/outscale/osc-sdk-go/v2 from 2.29.0 to 2.31.0 by @dependabot[bot] in https://github.com/outscale/cluster-api-provider-outscale/pull/650
* build(deps): bump the k8s group across 1 directory with 8 updates by @dependabot[bot] in https://github.com/outscale/cluster-api-provider-outscale/pull/655

## New Contributors
* @ddavid-numspot made their first contribution in https://github.com/outscale/cluster-api-provider-outscale/pull/686
