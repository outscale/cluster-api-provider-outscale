# Cluster API Provider Outscale (CAPOSC)

[![Project Incubating](https://docs.outscale.com/fr/userguide/_images/Project-Incubating-blue.svg)](https://docs.outscale.com/en/userguide/Open-Source-Projects.html)
[![](https://dcbadge.limes.pink/api/server/HUVtY5gT6s?style=flat&theme=default-inverted)](https://discord.gg/HUVtY5gT6s)

<p align="center">
  <img alt="Kubernetes Logo" src="https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg" width="120px">
</p>

---

## 🌐 Links

* 📘 Documentation: [Getting Started](./docs/src/topics/get-started-with-clusterctl.md)
* 🛠 Developer Guide: [Development](./docs/src/developers/developement.md)
* 🤝 Contribution Guide: [CONTRIBUTING.md](./CONTRIBUTING.md)
* 🌐 Cluster API website: [https://cluster-api.sigs.k8s.io](https://cluster-api.sigs.k8s.io)
* 💬 Join us on [Discord](https://discord.gg/YOUR_INVITE_CODE)

---

## 📄 Table of Contents

* [Overview](#-overview)
* [Project Status](#-project-status)
* [Requirements](#-requirements)
* [Installation](#-installation)
* [Usage](#-usage)
* [Development](#-development)
* [Contributing](#-contributing)
* [License](#-license)

---

## 🧭 Overview

**Cluster API Provider Outscale (CAPOSC)** enables Kubernetes-native declarative infrastructure management on the [OUTSCALE](https://www.outscale.com) Cloud using [Cluster API](https://cluster-api.sigs.k8s.io).

With CAPOSC, you can provision and manage Kubernetes clusters on OUTSCALE like any other Kubernetes resource—declaratively and at scale.

---

## 🚧 Project Status

This project is currently in **alpha**.
Features and APIs are subject to change. Use with caution in production environments.

---

## ✅ Requirements

* [Kubernetes 1.26+](https://kubernetes.io/)
* [clusterctl CLI](https://cluster-api.sigs.k8s.io/reference/clusterctl.html)
* An OUTSCALE account with API credentials
* Internet access for cluster provisioning

---

## ⚙️ Installation

Install `clusterctl` and initialize the management cluster with CAPOSC components.

Refer to the getting started guide:

```bash
kubectl create namespace capa-outscale-system
clusterctl init --infrastructure outscale
```

📘 See full instructions: [Getting Started with clusterctl](./docs/src/topics/get-started-with-clusterctl.md)

---

## 🚀 Usage

Once initialized, you can manage workload clusters declaratively using Kubernetes manifests.

Example:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: example-cluster
spec:
  ...
```

Apply with:

```bash
kubectl apply -f cluster.yaml
```

---

## 🛠 Development

To set up your environment for development or to build from source, follow the steps in the [Development Guide](./docs/src/developers/developement.md).

---

## 🤝 Contributing

We welcome community contributions!

Please read our [CONTRIBUTING.md](./CONTRIBUTING.md) guide to learn how to propose improvements, report issues, or open pull requests.

---

## 📜 License

**CAPOSC** is licensed under the BSD 3-Clause License.

© 2025 Outscale SAS

This project complies with the [REUSE Specification](https://reuse.software/).

See [LICENSES/](./LICENSES) directory for full license information.