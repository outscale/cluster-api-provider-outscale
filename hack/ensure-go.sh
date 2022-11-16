#!/bin/bash
# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

MIN_GO_VERSION="${MIN_GO_VERSION:-1.18.7}"
version_greater_equal()
{
    printf '%s\n%s\n' "$1" "$2" | sort --check=quiet --version-sort
}

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
	if ! version_greater_equal "$go_version" "$MIN_GO_VERSION"; then
	
		cat << EOF
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
