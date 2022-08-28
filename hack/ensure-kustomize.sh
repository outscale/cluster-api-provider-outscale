#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

BIN_ROOT="./bin"

goarch="$(go env GOARCH)"
goos="$(go env GOOS)"
if [ "$goos" != "linux" ]; then
	echo "OS '$OSTYPE' not supported. Aborting." >&2
	exit 1
fi

check_kustomize_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/kustomize")" ]; then
		echo 'kustomize is not found installing'
		echo 'Installing Kustomize' && install_kustomize
	fi
}

verify_kustomize_version() {

	local kustomize_version
	kustomize_version="$(${BIN_ROOT}/kustomize version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"
	if [[ "${MINIMUM_KUSTOMIZE_VERSION}" != "${kustomize_version}" ]]; then
		cat <<EOF
Detected kustomize version: ${kustomize_version}.
Install ${MINIMUM_KUSTOMIZE_VERSION} of kustomize
EOF

		echo 'Installing Kustomize' && install_kustomize
	else
		cat <<EOF
Detected kustomize version: ${kustomize_version}.
Kustomize ${MINIMUM_KUSTOMIZE_VERSION} is already install.
EOF
	fi
}

install_kustomize() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		archive_name="kustomize-v${MINIMUM_KUSTOMIZE_VERSION}.tar.gz"
		curl -sLo "${BIN_ROOT}/${archive_name}" "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${MINIMUM_KUSTOMIZE_VERSION}/kustomize_v${MINIMUM_KUSTOMIZE_VERSION}_${goos}_${goarch}.tar.gz"
		tar -zvxf "${BIN_ROOT}/${archive_name}" -C "${BIN_ROOT}/"
		chmod +x "${BIN_ROOT}/kustomize"
		rm "${BIN_ROOT}/${archive_name}"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_kustomize_installed "$@"
verify_kustomize_version "$@"
