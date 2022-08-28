#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

PATH_BIN="/tmp"

check_helm_installed() {
	if ! [ -x "$(command -v helm)" ]; then
		echo 'helm not found, installing'
		install_helm
	fi
}

verify_helm_version() {

	local helm_version
	helm_version="$(helm version --template="{{ .Version }}")"
	if [[ "v${MINIMUM_HELM_VERSION}" != "${helm_version}" ]]; then
		cat <<EOF
Detected helm version: ${helm_version}.
Install ${MINIMUM_HELM_VERSION} of Helm.
EOF

		echo 'Installing Helm' && install_helm
	else
		cat <<EOF
Detected helm version: v${helm_version}.
Helm v${MINIMUM_HELM_VERSION} is already installed.
EOF
	fi
}

install_helm() {
	if ! [ -d "${PATH_BIN}" ]; then
		mkdir -p "${PATH_BIN}"
	fi
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
	echo "Done"
}

check_helm_installed "$@"
verify_helm_version "$@"
