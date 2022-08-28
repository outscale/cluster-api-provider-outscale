#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

BIN_ROOT="./bin"
check_envsubst_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/envsubst")" ]; then
		echo 'envsubst is not found installing'
		echo 'Installing envsubst' && install_envsubst
	fi
}

install_envsubst() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v github.com/drone/envsubst/v2/cmd/envsubst
		cp "$HOME/go/bin/envsubst" "${BIN_ROOT}/envsubst"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_envsubst_installed "$@"
