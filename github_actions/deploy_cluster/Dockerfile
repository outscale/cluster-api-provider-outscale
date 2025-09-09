FROM python:3-bookworm

RUN curl -fsSL https://packages.buildkite.com/helm-linux/helm-debian/gpgkey | gpg --dearmor > /usr/share/keyrings/helm.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/helm.gpg] https://packages.buildkite.com/helm-linux/helm-debian/any/ any main" > /etc/apt/sources.list.d/helm-stable-debian.list

RUN apt-get update && apt-get upgrade -y && apt-get install -y helm jq && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.2/clusterctl-linux-amd64 -o clusterctl && install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl

RUN curl -LO https://dl.k8s.io/release/v1.30.12/bin/linux/amd64/kubectl && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

RUN python3 -m venv /.venv && . /.venv/bin/activate && /.venv/bin/pip install https://docs.eu-west-2.oks.outscale.com/oks-cli/oks_cli-latest.zip

COPY main.sh cleanup.sh /

ENTRYPOINT ["/main.sh"]