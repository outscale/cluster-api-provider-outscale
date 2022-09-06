#!/bin/bash

check_kind_installed() {
	if ! [ -x "$(command -v kind)" ]; then
		echo 'kind not found, installing'
		install_kind
	fi
}

verify_kind_version() {

	local kind_version
	kind_version="v$(kind version -q)"
	if [[ "v${MINIMUM_KIND_VERSION}" != "${kind_version}" ]]; then
		cat <<EOF
Detected kind version: v${kind_version}.
Install v${MINIMUM_KIND_VERSION} of kind.
EOF
		echo 'Installing  kind' && install_kind
	else
		cat <<EOF
Detected kind version: ${kind_version}.
Kind v${MINIMUM_KIND_VERSION} is already installed.
EOF
	fi
}

install_kind() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		curl -sLo "kind" "https://github.com/kubernetes-sigs/kind/releases/download/${MINIMUM_KIND_VERSION}/kind-linux-amd64"
		copy_binary
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi

}

function copy_binary() {
	echo "Copy binary in /usr/local/bin which is protected as sudo"
	sudo mv kind /usr/local/bin/kind
	chmod +x "/usr/local/bin/kind"
	echo "Installation Finished"
}

check_kind_installed "$@"
verify_kind_version "$@"
