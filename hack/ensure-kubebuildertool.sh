#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

BIN_ROOT="/usr/local"

goarch="$(go env GOARCH)"
goos="$(go env GOOS)"
if [ "$goos" != "linux" ]; then
	echo "OS '$OSTYPE' not supported. Aborting." >&2
	exit 1
fi

check_kubebuildertool_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/kubebuilder/bin/kube-apiserver")" ]; then
		echo 'kubebuilder is not found installing'
		echo 'Installing Kubebuilder' && install_kubebuildertool
	fi
}

verify_kubebuildertool_version() {

	local kubebuildertool_version
	kubebuildertool_version="$(${BIN_ROOT}/kubebuilder/bin/kube-apiserver --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"
	if [[ "${MINIMUM_KUBEBUILDERTOOL_VERSION}" != "${kubebuildertool_version}" ]]; then
		cat <<EOF
Detected kubebuildertool version: ${kubebuildertool_version}.
Install ${MINIMUM_KUBEBUILDERTOOL_VERSION} of kubebuildertool
EOF

		echo 'Installing KubebuilderTool' && install_kubebuildertool
	else
		cat <<EOF
Detected kubebuildertool version: ${kubebuildertool_version}.
kubebuildertool ${MINIMUM_KUBEBUILDERTOOL_VERSION} is already installed.
EOF
	fi
}

install_kubebuildertool() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		curl -sLo "${BIN_ROOT}/kubebuilder-tools" "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${MINIMUM_KUBEBUILDERTOOL_VERSION}-${goos}-${goarch}.tar.gz"
		tar -zvxf "${BIN_ROOT}/kubebuilder-tools" -C "${BIN_ROOT}"
		chmod +x "${BIN_ROOT}/kubebuilder/bin/etcd"
                chmod +x "${BIN_ROOT}/kubebuilder/bin/kubectl"
                chmod +x "${BIN_ROOT}/kubebuilder/bin/kube-apiserver"
		rm "${BIN_ROOT}/kubebuilder-tools"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_kubebuildertool_installed "$@"
verify_kubebuildertool_version "$@"
