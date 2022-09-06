#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

check_packer_installed() {
	if ! [ -x "$(command -v packer)" ]; then
		echo 'packer not found, installing'
		install_packer
	fi
}

verify_packer_version() {

	local packer_version
	packer_version="$(/usr/local/bin/packer --version)"
	if [[ "${MINIMUM_PACKER_VERSION}" != "${packer_version}" ]]; then
		cat <<EOF
Detected packer version: v${packer_version}.
Install v${MINIMUM_PACKER_VERSION} of packer.
EOF
		echo 'Installing Packer' && install_packer
	else
		cat <<EOF
Detected packer version: v${packer_version}.
Packer v${MINIMUM_PACKER_VERSION} is already installed.
EOF
	fi
}

install_packer() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		curl -sLo "packer.zip" "https://releases.hashicorp.com/packer/${MINIMUM_PACKER_VERSION}/packer_${MINIMUM_PACKER_VERSION}_linux_amd64.zip" && unzip packer.zip -d packer-bin && mv packer-bin/packer . && rm -rf packer.zip packer-bin
		copy_binary
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi

}

function copy_binary() {
	echo "Copy binary in /usr/local/bin which is protected as sudo"
	sudo mv packer /usr/local/bin/packer
	chmod +x "/usr/local/bin/packer"
	echo "Installation Finished"
}

check_packer_installed "$@"
verify_packer_version "$@"
