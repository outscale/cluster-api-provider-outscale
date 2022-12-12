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

TARGET=target/x86_64-unknown-linux-musl/release/json-format
KUBERNETES_SERIES ?= v1.22
KUBERNETES_SEMVER ?= v1.22.1
INPUT ?= kubernetes.json
OUTPUT ?= overwrite-1.22.json
WORKING_BINARY_DIRECTORY ?= $(shell pwd)/target/x86_64-unknown-linux-musl/release
FORMAT ?= params
all: help

.PHONY: help
help:
	@echo "help:"
	@echo "- make build: build the binary of cluster-api"
	@echo "- make test: run all tests"

build: $(TARGET)

target/x86_64-unknown-linux-musl/release/json-format: src/*.rs
	cargo build --target x86_64-unknown-linux-musl --release

.PHONY: test
test: build cargo-test format-test integration-test
	@echo all tests OK

.PHONY: integration-test
integration-test:
	./run-integration-test.sh -f $(FORMAT) -k $(KUBERNETES_SERIES) -r $(KUBERNETES_SEMVER) -w $(WORKING_BINARY_DIRECTORY) -i $(INPUT) -o $(OUTPUT)

.PHONY: cargo-test
cargo-test:
	cargo test

.PHONY: format-test
format-test:
	cargo fmt --check
	cargo clippy

