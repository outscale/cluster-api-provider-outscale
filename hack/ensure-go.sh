#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

MIN_GO_VERSION="${MIN_GO_VERSION:-1.18.5}"

# Ensure the go tool exists and is a viable version.
verify_go_version() {
	if ! command -v go >/dev/null 2>&1; then
		cat <<EOF
Can't find 'go' in PATH, please fix and retry.
See http://golang.org/doc/install for installation instructions.
EOF
		return 2
	fi

	local go_version
	go_version="$(go version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"

	if [[ "${MIN_GO_VERSION}" != "${go_version}" ]]; then
		cat <<EOF
Detected go version: ${go_version}.
Kubernetes requires ${MIN_GO_VERSION} or greater.
Please install ${MIN_GO_VERSION} or later.
EOF
		return 2
	fi
}

verify_go_version
export GO111MODULE=on
export GOFLAGS="-mod=vendor"
