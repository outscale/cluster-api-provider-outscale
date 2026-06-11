FROM debian:13@sha256:fe7312b5f05bf5f43fad76bcd8945642e4e47a68aefd1b73f447615899d0fac1

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN apt-get update && apt-get install -y curl gpg ca-certificates

RUN curl -fsSL --retry 5 --retry-all-errors https://packages.buildkite.com/helm-linux/helm-debian/gpgkey | gpg --dearmor > /usr/share/keyrings/helm.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/helm.gpg] https://packages.buildkite.com/helm-linux/helm-debian/any/ any main" > /etc/apt/sources.list.d/helm-stable-debian.list

RUN apt-get update && apt-get install -y helm=3.20.0-1 --no-install-recommends && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN curl -sSL --retry 5 --retry-all-errors https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.8/clusterctl-linux-amd64 -o clusterctl && install clusterctl /usr/local/bin/clusterctl
RUN curl -sSLO --retry 5 --retry-all-errors https://dl.k8s.io/release/v1.34.8/bin/linux/amd64/kubectl && install kubectl /usr/local/bin/kubectl
RUN curl -sSL --retry 5 --retry-all-errors https://github.com/outscale/octl/releases/download/v0.0.23/octl_Linux_x86_64 -o octl && install octl /usr/local/bin/octl

COPY main.sh cleanup.sh /

ENTRYPOINT ["/main.sh"]
