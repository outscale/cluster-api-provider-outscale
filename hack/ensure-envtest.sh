#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

BIN_ROOT="./bin"
check_envtest_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/envtest")" ]; then
		echo 'envtest is not found installing'
		echo 'Installing envtest' && install_envtest
	fi
}

verify_envtest_version() {

	local envtest_version
	envtest_version="$(${BIN_ROOT}/envtest list | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	if [[ "${MINIMUM_ENVTEST_VERSION}" != "${envtest_version}" ]]; then
		cat <<EOF
Detected envtest install kubernetes version: v${envtest_version}
Install envtest with kubernetes version v${MINIMUM_ENVTEST_VERSION}
EOF

		echo 'Installing envtest' && install_envtest
	else
		cat <<EOF
Detected envtest install kubernetes version: v${envtest_version}.
envtest with kubernetes version v${MINIMUM_ENVTEST_VERSION} is already installed.
EOF
	fi
}

install_envtest() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
		cp "$HOME/go/bin/setup-envtest" "${BIN_ROOT}/envtest"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_envtest_installed "$@"
verify_envtest_version "$@"
