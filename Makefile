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

# Ensure Make is run with bash shell as some syntax below is bash-specific
# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.

# Image URL to use all building/pushing image targets
REGISTRY ?= outscale
IMAGE_NAME ?= cluster-api-provider-osc
TAG ?= dev
RELEASE_TAG ?= v0.1.0
IMG ?= $(REGISTRY)/$(IMAGE_NAME):$(TAG)
IMG_RELEASE ?= $(REGISTRY)/$(IMAGE_NAME):$(RELEASE_TAG)
OSC_ACCESS_KEY ?= access
OSC_SECRET_KEY ?= secret
OSC_CLUSTER ?= cluster-api
CLUSTER ?= cluster-api
LOG_TAIL ?= -1
CAPI_VERSION ?= v1.1.4
CAPI_NAMESPACE ?= capi-kubeadm-bootstrap-system
CAPO_NAMESPACE ?= cluster-api-provider-outscale-system
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
E2E_CONF_FILE_SOURCE ?= ${PWD}/test/e2e/config/outscale-ci.yaml
E2E_CONF_FILE ?= ${PWD}/test/e2e/config/outscale-ci-envsubst.yaml
MINIMUM_GOLANGCI_LINT_VERSION=1.49.0
MINIMUM_SHELLCHECK_VERSION=0.8.0
MODE=check
MINIMUM_BUILDIFIER_VERSION=5.1.0
MINIMUM_YAMLFMT_VERSION=0.3.0
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate/boilerplate.go.txt" paths="./..."

.PHONY: mock-generate
mock-generate: mockgen ## Generate mock
	go generate ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: format
format: gofmt gospace yamlspace yamlfmt  

.PHONY: gofmt
gofmt: ## Run gofmt
	find . -name "*.go" | grep -v "\/vendor\/" | xargs gofmt -s -w

.PHONY: gospace
gospace: ## Run to remove trailling space
	find . -name "*.go" -type f -print0 | xargs -0 sed -i 's/[[:space:]]*$$//'

.PHONY: yamlspace
yamlspace: ## Run to remove trailling space
	find . -name "*.yaml" -type f -print0 | xargs -0 sed -i 's/[[:space:]]*$$//'

.PHONY: yamlfmt
yamlfmt: install-yamlfmt
	find . -name "*.yaml" | grep -v "\/vendor\/" | xargs yamlfmt


.PHONY: checkfmt
checkfmt: ## check gofmt
	./check-gofmt

.PHONY: unit-test
unit-test:
	go test -v -coverprofile=covers.out  ./controllers
	go tool cover -func=covers.out -o covers.txt
	go tool cover -html=covers.out -o covers.html
	go test -v -coverprofile=apicovers.out  ./api/v1beta1
	go tool cover -func=apicovers.out -o apicovers.txt
	go tool cover -html=apicovers.out -o apicovers.html
	

.PHONY: cloud-init-secret
cloud-init-secret:
	kubectl create secret generic cluster-api-test --save-config --dry-run=client --from-file=${PWD}/testenv/value -o yaml | kubectl apply -f -

.PHONY: testenv
testenv: cloud-init-secret
	USE_EXISTING_CLUSTER=true go test -v -coverprofile=covers.out  ./testenv/ -ginkgo.v -ginkgo.progress -test.v

.PHONY: e2e-conf-file
e2e-conf-file: envsubst
	$(ENVSUBST) < $(E2E_CONF_FILE_SOURCE) > $(E2E_CONF_FILE)

.PHONY: e2etest
e2etest: envsubst e2e-conf-file
	@USE_EXISTING_CLUSTER=true IMG=${IMG} go test -v -coverprofile=covers.out  ./test/e2e -ginkgo.v -ginkgo.progress -test.v -e2e.artifacts-folder=${PWD}/artifact -e2e.config=$(E2E_CONF_FILE)

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: # Build docker image with the manager
	docker build -t ${IMG} .

.PHONY: docker-build-dev
docker-build-dev: ## Generate and Build docker image with the manager
	generate docker-build

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -


KUSTOMIZE := $(shell command -v kustomize 2> /dev/null)

.PHONY: deploy
deploy: envsubst ## Deploy controller to the K8s cluster specified in ~/.kube/config.
ifndef KUSTOMIZE
	cd config/default && $(LOCAL_KUSTOMIZE) edit set image controller=${IMG}
	$(LOCAL_KUSTOMIZE) build config/default | $(ENVSUBST) | kubectl apply -f -
else
	cd config/default && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(ENVSUBST) | kubectl apply -f -
endif

.PHONY: credential
credential: ## Set Credentials
	kubectl create namespace cluster-api-provider-outscale-system --dry-run=client -o yaml | kubectl apply -f -
	@kubectl create secret generic cluster-api-provider-outscale --from-literal=access_key=${OSC_ACCESS_KEY} --from-literal=secret_key=${OSC_SECRET_KEY} -n cluster-api-provider-outscale-system ||:

.PHONY: logs-capo
logs-capo: ## Get logs capo
	kubectl logs -l  control-plane=controller-manager -n ${CAPO_NAMESPACE}  --tail=${LOG_TAIL}

.PHONY: delete-capo
delete-capo: ## Delete Cluster api Outscale cluster
	kubectl delete cluster ${CLUSTER}
	kubectl delete osccluster ${OSC_CLUSTER}

.PHONY: force-delete-capo
force-delete-capo: ## Force delete Cluster api Outscale cluster (Remove finalizers)
	kubectl patch cluster ${CLUSTER} -p '{"metadata":{"finalizers":null}}' --type=merge
	kubectl patch osccluster ${OSC_CLUSTER}  -p '{"metadata":{"finalizers":null}}' --type=merge

.PHONY: logs-capi
logs-capi: ## Get logs capi
	kubectl logs -l  control-plane=controller-manager -n ${CAPI_NAMESPACE}  --tail=${LOG_TAIL}

.PHONY: capm
capm: envsubst ## Deploy controller to the K8s cluster specified in ~/.kube/config.
ifndef KUSTOMIZE
	cd config/default && $(LOCAL_KUSTOMIZE) edit set image controller=${IMG}
	$(LOCAL_KUSTOMIZE) build config/default | $(ENVSUBST)  > capm.yaml
else
	cd config/default && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(ENVSUBST)  > capm.yaml
endif

.PHONY: deploy-dev
deploy-dev: manifests deploy  ## Deploy controller to the K8s cluster specified in ~/.kube/config.

.PHONY: undeploy
undeploy: envsubst ## Undeploy controller to the K8s cluster specified in ~/.kube/config.
ifndef KUSTOMIZE
	cd config/default && $(LOCAL_KUSTOMIZE) edit set image controller=${IMG}
	$(LOCAL_KUSTOMIZE) build config/default | $(ENVSUBST)  | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
else
	cd config/default && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(ENVSUBST)  | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
endif

##@ Release

RELEASE_DIR := out
GH_ORG_NAME ?= outscale-dev
GH_REPO_NAME ?= cluster-api-provider-outscale
RELEASE_BINARY ?= cluster-api
GH_REPO ?= outscale-dev/$(GH_REPO_NAME)
GOARCH  := $(shell go env GOARCH)
GOOS    := $(shell go env GOOS)
RELEASE_TAG ?= v0.1.0

release_dir:
	mkdir -p $(RELEASE_DIR)/

.PHONY: clean-release
clean-release:
	rm -rf $(RELEASE_DIR)

.PHONY: release
release: clean-release release_dir check-release-tag release-templates release-manifests

.PHONY: release-manifests
release-manifests: kustomize release_dir envsubst
	cp metadata.yaml $(RELEASE_DIR)/metadata.yaml
ifndef KUSTOMIZE
	cd config/default && $(LOCAL_KUSTOMIZE) edit set image controller=${IMG_RELEASE}
	$(LOCAL_KUSTOMIZE) build config/default | $(ENVSUBST) > out/infrastructure-components.yaml
else
	cd config/default && $(KUSTOMIZE) edit set image controller=${IMG_RELEASE}
	$(KUSTOMIZE) build config/default | $(ENVSUBST) > out/infrastructure-components.yaml
endif
.PHONY: release-templates
release-templates:
	cp templates/cluster-template* $(RELEASE_DIR)/

.PHONY: release-binary
release-binary:
	docker run \
		--rm \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-v "$$(pwd):/workspace" \
		-w /workspace \
		golang:1.18 \
		go build -a -ldflags '$(LDFLAGS) -extldflags "-static"' \
		-o $(RELEASE_DIR)/$(notdir $(RELEASE_BINARY))-$(GOOS)-$(GOARCH) $(RELEASE_BINARY)

.PHONY: release-tag
release-tag:
	docker tag $(IMG) $(IMG_RELEASE)
	docker push $(IMG_RELEASE)

.PHONY: check-previous-release-tag
check-previous-release-tag: ## Check if the previous release tag is set
	@if [ -z "${PREVIOUS_RELEASE_TAG}" ]; then echo "PREVIOUS_RELEASE_TAG is not set"; exit 1; fi

.PHONY: check-release-tag
check-release-tag: ## Check if the release tag is set
	@if [ -z "${RELEASE_TAG}" ]; then echo "RELEASE_TAG is not set"; exit 1; fi

.PHONY: check-github-token
check-github-token:
	@if [ -z "${SECRET_GITHUB_TOKEN}" ]; then echo "GITHUB_TOKEN is not set"; exit 1; fi

.PHONY: gh-login
gh-login: gh check-github-token
	cat <<< "${SECRET_GITHUB_TOKEN}"  | $(GH)  auth login --with-token

GH = $(shell pwd)/bin/gh

.PHONY: release-changelog
release-changelog: gh release_dir check-release-tag check-previous-release-tag
	./hack/releasechangelog.sh -t $(RELEASE_TAG) -p ${PREVIOUS_RELEASE_TAG} -g ${GH} -o $(GH_ORG_NAME) -r $(GH_REPO_NAME) -i $(IMG_RELEASE)  > $(RELEASE_DIR)/CHANGELOG.md

.PHONY: create-gh-release
create-gh-release: gh
	$(GH) release create $(RELEASE_TAG) -d -F $(RELEASE_DIR)/CHANGELOG.md -t $(RELEASE_TAG) -R $(GH_REPO) $(RELEASE_DIR)/*.yaml

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	mkdir -p $(shell pwd)/bin
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)


.PHONY: install-golangcilint
install-golangcilint: ## Download golangci-lint
	MINIMUM_GOLANGCI_LINT_VERSION=$(MINIMUM_GOLANGCI_LINT_VERSION) ./hack/ensure-golangci-lint.sh

.PHONY: install-yamlfmt
install-yamlfmt: ## donwload yaml fmt
        MINIMUM_YAMLFMT_VERSION=$(MINIMUM_YAMLFMT_VERSION) ./hack/ensure-yamlfmt.sh

.PHONY: install-shellcheck
install-shellcheck: ## Download shellcheck
	MINIMUM_SHELLCHECK_VERSION=$(MINIMUM_SHELLCHECK_VERSION) ./hack/verify-shellcheck.sh
.PHONY: verify-boilerplate
verify-boilerplate: ## Verify boilerplate text exists in each file
	./hack/verify-boilerplate.sh

.PHONY: install-buildifier
install-buildifier: ## Download shellcheck
	MODE=${MODE_BUILDIFIER} MINIMUM_BUILDIFIER_VERSION=$(MINIMUM_BUILDIFIER_VERSION) ./hack/verify-buildifier.sh


LOCAL_CLUSTERCTL ?= $(shell pwd)/bin/clusterctl
.PHONY: install-clusterctl
install-clusterctl: ## Download clusterctl locally if necessary.
	@if [ ! -s ${LOCAL_CLUSTERCTL} ]; then \
		mkdir -p $(shell pwd)/bin; \
		wget -c https://github.com/kubernetes-sigs/cluster-api/releases/download/${CAPI_VERSION}/clusterctl-linux-amd64; \
		mv $(shell pwd)/clusterctl-linux-amd64 $(shell pwd)/bin/clusterctl; \
		chmod +x  $(LOCAL_CLUSTERCTL); \
	fi

.PHONY: deploy-clusterapi
deploy-clusterapi: install-clusterctl ## Deploy clusterapi
	$(LOCAL_CLUSTERCTL) init  --wait-providers -v 10

.PHONY: undeploy-clusterapi
undeploy-clusterapi:  ## undeploy clusterapi
	$(LOCAL_CLUSTERCTL) delete --all --include-crd  --include-namespace -v 10

LOCAL_KUSTOMIZE ?= $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	mkdir -p $(shell pwd)/bin
	$(call go-get-tool,$(LOCAL_KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	mkdir -p $(shell pwd)/bin
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

MOCKGEN = $(shell pwd)/bin/mockgen
.PHONY: mockgen
mockgen: ## Download mockgen locally if necessary.
	mkdir -p $(shell pwd)/bin
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@v1.6.0)

ENVSUBST = $(shell pwd)/bin/envsubst
.PHONY: envsubst
envsubst: ## Download envsubst
	mkdir -p $(shell pwd)/bin
	go build -tags=tools -o $(ENVSUBST) github.com/drone/envsubst/v2/cmd/envsubst

GH = $(shell pwd)/bin/gh
GH_VERSION ?= 2.14.4
.PHONY: gh
gh:
	@if [ ! -s ${GH} ]; then \
		mkdir -p $(shell pwd)/bin; \
		curl https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz -Lo gh_${GH_VERSION}_linux_amd64.tar.gz; \
		tar -zxvf gh_2.14.4_linux_amd64.tar.gz gh_2.14.4_linux_amd64/bin/gh  --strip-components 2 -C ${GH}; \
                mv gh ./bin; \
		rm -f gh_${GH_VERSION}_linux_amd64.tar.gz; \
	fi
	

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
