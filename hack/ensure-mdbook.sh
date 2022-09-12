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

check_mdbook_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/mdbook")" ]; then
		echo "mdbook is not found installing"
		echo "Installing mdbook" && install_mdbook
	fi
}

verify_mdbook_version() {

	local mdbook_version
	mdbook_version="$(${BIN_ROOT}/mdbook --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"
	if [[ "${MINIMUM_MDBOOK_VERSION}" != "${mdbook_version}" ]]; then
		cat <<EOF
Detected mdbook version: ${mdbook_version}.
Install ${MINIMUM_MDBOOK_VERSION} of mdbook.
EOF
                echo "Installing mdbook" && install_mdbook

	else
		cat <<EOF
Detected mdbook version: ${mdbook_version}.
Kustomize ${MINIMUM_MDBOOK_VERSION} is already installied.
EOF
	fi
}

install_mdbook() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		curl -sLo "${BIN_ROOT}/mdbook.tar.gz" "https://github.com/rust-lang/mdBook/releases/download/v${MINIMUM_MDBOOK_VERSION}/mdbook-v${MINIMUM_MDBOOK_VERSION}-x86_64-unknown-linux-gnu.tar.gz"
		echo "https://github.com/rust-lang/mdBook/releases/download/v${MINIMUM_MDBOOK_VERSION}/mdbook-v${MINIMUM_MDBOOK_VERSION}-x86_64-unknown-linux-gnu.tar.gz"
		tar -zvxf "${BIN_ROOT}/mdbook.tar.gz" -C "${BIN_ROOT}/"
		chmod +x "${BIN_ROOT}/mdbook"
		rm "${BIN_ROOT}/mdbook.tar.gz"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_mdbook_installed "$@"
verify_mdbook_version "$@"
