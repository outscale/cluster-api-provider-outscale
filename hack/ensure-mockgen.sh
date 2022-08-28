#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

BIN_ROOT="./bin"
check_mockgen_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/mockgen")" ]; then
		echo 'mockgen is not found installing'
		echo 'Installing mockgen' && install_mockgen
	fi
}

verify_mockgen_version() {

	local mockgen_version
	mockgen_version="$(${BIN_ROOT}/mockgen --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	if [[ "${MINIMUM_MOCKGEN_VERSION}" != "${mockgen_version}" ]]; then
		cat <<EOF
Detected mockgen version: ${mockgen_version}
Install ${MINIMUM_MOCKGEN_VERSION} of mockgen
EOF

		echo 'Installing mockgen' && install_mockgen
	else
		cat <<EOF
Detected mockgen version: ${mockgen_version}.
mockgen ${MINIMUM_MOCKGEN_VERSION} is already installed.
EOF
	fi
}

install_mockgen() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v "github.com/golang/mock/mockgen@v${MINIMUM_MOCKGEN_VERSION}"
		cp "$HOME/go/bin/mockgen" "${BIN_ROOT}/mockgen"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_mockgen_installed "$@"
verify_mockgen_version "$@"
