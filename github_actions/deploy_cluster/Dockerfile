FROM python:3-bookworm

RUN apt update && apt upgrade -y

RUN curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.1/clusterctl-linux-amd64 -o clusterctl && install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl

RUN curl -LO https://dl.k8s.io/release/v1.30.12/bin/linux/amd64/kubectl && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

RUN python3 -m venv /.venv && . /.venv/bin/activate && /.venv/bin/pip install https://docs.eu-west-2.oks.outscale.com/oks-cli/oks_cli-latest.zip

COPY main.sh cleanup.sh /

ENTRYPOINT ["/main.sh"]