
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
MINIMUM_KUBEBUILDERTOOL_VERSION=1.24.1
MINIMUM_ENVTEST_VERSION=1.23.3
E2E_CONF_FILE_SOURCE ?= ${PWD}/test/e2e/config/outscale-ci.yaml
E2E_CONF_FILE ?= ${PWD}/test/e2e/config/outscale-ci-envsubst.yaml
MINIMUM_CLUSTERCTL_VERSION=1.1.3
MIN_GO_VERSION=1.18.5
MINIMUM_TILT_VERSION=0.25.3
MINIMUM_PACKER_VERSION=1.8.1
MINIMUM_CONTROLLER_GEN_VERSION=0.8.0 
MINIMUM_GH_VERSION=2.12.1
MINIMUM_KIND_USE_VERSION=v0.14.0
MINIMUM_ENVTEST_VERSION=1.23.3
MINIMUM_HELM_VERSION=v3.9.4
MINIMUM_KUSTOMIZE_VERSION=4.5.1
MINIMUM_MOCKGEN_VERSION=1.6.0
MINIMUM_KUBECTL_VERSION=1.22.10
E2E_CONF_CLASS_FILE_SOURCE ?= ${PWD}/test/e2e/config/outscale.yaml
E2E_CONF_CLASS_FILE ?= ${PWD}/test/e2e/config/outscale-envsubst.yaml
E2E_CONF_CCM_FILE_SOURCE ?= ${PWD}/test/e2e/data/ccm/ccm.yaml.template
E2E_CONF_CCM_FILE ?= ${PWD}/test/e2e/data/ccm/ccm.yaml
E2E_CONF_CLUSTER_CLASS_FILE_SOURCE ?= ${PWD}/example/cluster-machine-template-with-clusterclass.yaml.tmpl
E2E_CONF_CLUSTER_CLASS_FILE ?= ${PWD}/example/cluster-machine-template-with-clusterclass.yaml
E2E_CLUSTER_CLASS_FILE_SOURCE ?= ${PWD}/example/clusterclass.yaml.tmpl
E2E_CLUSTER_CLASS_FILE ?= ${PWD}/example/clusterclass.yaml
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
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: mock-generate
mock-generate: mockgen ## Generate mock
	go generate ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: gofmt
gofmt: ## Run gofmt
	find . -name "*.go" | grep -v "\/vendor\/" | xargs gofmt -s -w

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

.PHONY: testtools
testtools:
	USE_EXISTING_CLUSTER=true go test -v -coverprofile=covers.out  ./testtools/ -ginkgo.v -ginkgo.progress -test.v

.PHONY: e2e-conf-file
e2e-conf-file: envsubst
	$(ENVSUBST) < $(E2E_CONF_FILE_SOURCE) > $(E2E_CONF_FILE)

.PHONY: e2etest
e2etest: envsubst e2e-conf-file
	USE_EXISTING_CLUSTER=true IMG=${IMG} go test -v -coverprofile=covers.out  ./test/e2e -test.timeout 30m -ginkgo.v -ginkgo.progress  -e2e.use-cni=false -e2e.use-existing-cluster=true -ginkgo.focus=".*basic.*" -e2e.use-ccm=false -test.v -e2e.validate-stack=false -e2e.artifacts-folder=${PWD}/artifact -e2e.config=$(E2E_CONF_FILE)

.PHONY: e2e-conf-class-file
e2e-conf-class-file: envsubst
	$(ENVSUBST) < $(E2E_CONF_CLASS_FILE_SOURCE) > $(E2E_CONF_CLASS_FILE)

.PHONY: e2etestexistingcluster
e2etestexistingcluster: envsubst e2e-conf-class-file ccm-file
	USE_EXISTING_CLUSTER=true IMG=${IMG} go test -v -coverprofile=covers.out  ./test/e2e -test.timeout 180m -e2e.use-existing-cluster=true -ginkgo.focus=".*feature.*" -ginkgo.v -ginkgo.progress -test.v -e2e.artifacts-folder=${PWD}/artifact -e2e.use-cni=true -e2e.use-ccm=true -e2e.validate-stack=true -e2e.config=$(E2E_CONF_CLASS_FILE)

.PHONY: e2etestkind
e2etestkind: envsubst e2e-conf-class-file ccm-file
	USE_EXISTING_CLUSTER=false IMG=${IMG} go test -v -coverprofile=covers.out  ./test/e2e -test.timeout 180m -e2e.use-existing-cluster=false -ginkgo.v -ginkgo.skip=".*feature.*|.*conformance.*|.*basic.*" -ginkgo.focus=".*first_upgrade.*|.*KCP.*remediation.*" -e2e.validate-stack=false  -ginkgo.progress -test.v -e2e.artifacts-folder=${PWD}/artifact -e2e.use-cni=true -e2e.use-ccm=true -e2e.config=$(E2E_CONF_CLASS_FILE) 

.PHONY: e2econformance
e2econformance: envsubst e2e-conf-class-file ccm-file
	USE_EXISTING_CLUSTER=true IMG=${IMG} go test -v -coverprofile=covers.out  ./test/e2e -test.timeout 180m -e2e.use-existing-cluster=true -ginkgo.focus=".*conformance.*" -ginkgo.v -ginkgo.progress -test.v -e2e.artifacts-folder=${PWD}/artifact -e2e.use-cni=true -e2e.use-ccm=true -e2e.validate-stack=false -e2e.config=$(E2E_CONF_CLASS_FILE)


.PHONY: ccm-file
ccm-file: envsubst
	$(ENVSUBST) < $(E2E_CONF_CCM_FILE_SOURCE) > $(E2E_CONF_CCM_FILE)

.PHONY: ccm-cni-ex
ccm-cni-ex: envsubst
	$(ENVSUBST) < $(E2E_CONF_CLUSTER_CLASS_FILE_SOURCE) > $(E2E_CONF_CLUSTER_CLASS_FILE)

.PHONY: cluster-class-ex
cluster-class-ex: envsubst
	$(ENVSUBST) < $(E2E_CLUSTER_CLASS_FILE_SOURCE) > $(E2E_CLUSTER_CLASS_FILE)

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
credential: envsubst ## Set Credentials
	kubectl create namespace cluster-api-provider-outscale-system --dry-run=client -o yaml | kubectl apply -f - ||:
	@kubectl create secret generic cluster-api-provider-outscale --from-literal=access_key=${OSC_ACCESS_KEY} --from-literal=secret_key=${OSC_SECRET_KEY} -n cluster-api-provider-outscale-system ||:
	@cat ./hack/ccm-secret.yaml | $(ENVSUBST)| kubectl apply -f - ||:

.PHONY: clusterclass
clusterclass: envsubst ## Set Clusterclass
	@cat ./example/cluster-class.yaml | $(ENVSUBST)| kubectl apply -f - 

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

.PHONY: create-capm
create-capm: kustomize controller-gen manifests generate capm

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
controller-gen: ## Download controller-gen
	MINIMUM_CONTROLLER_GEN_VERSION=${MINIMUM_CONTROLLER_GEN_VERSION} ./hack/ensure-controller-gen.sh 

LOCAL_CLUSTERCTL ?= $(shell pwd)/bin/clusterctl
.PHONY: install-clusterctl
install-clusterctl: ## Download clusterctl
	MINIMUM_CLUSTERCTL_VERSION=$(MINIMUM_CLUSTERCTL_VERSION) ./hack/ensure-clusterctl.sh

.PHONY: verify-go
verify-go:  ## Download go
	MIN_GO_VERSION=$(MIN_GO_VERSION) ./hack/ensure-go.sh

.PHONY: install-tilt
install-tilt: ## Download tilt
	MINIMUM_TILT_VERSION=$(MINIMUM_TILT_VERSION) ./hack/ensure-tilt.sh

.PHONY: install-packer
install-packer: ## Download packer
	MINIMUM_PACKER_VERSION=$(MINIMUM_PACKER_VERSION) ./hack/ensure-packer.sh

.PHONY: install-gh
install-gh: ## Download gh
	MINIMUM_GH_VERSION=$(MINIMUM_GH_VERSION) ./hack/ensure-gh.sh 

.PHONY: install-kind
install-kind: ## Download kind
	MINIMUM_KIND_VERSION=$(MINIMUM_KIND_USE_VERSION)  ./hack/ensure-kind.sh

.PHONY: install-kubectl
install-kubectl: ## Download kubectl
	MINIMUM_KUBECTL_VERSION=$(MINIMUM_KUBECTL_VERSION) ./hack/ensure-kubectl.sh

.PHONY: install-helm
install-helm: ## Downlload helm
	MINIMUM_HELM_VERSION=$(MINIMUM_HELM_VERSION) ./hack/ensure-helm.sh

.PHONY: deploy-clusterapi
deploy-clusterapi: install-clusterctl ## Deploy clusterapi
	$(LOCAL_CLUSTERCTL) init  --wait-providers -v 10

.PHONY: undeploy-clusterapi
undeploy-clusterapi:  ## undeploy clusterapi
	$(LOCAL_CLUSTERCTL) delete --all --include-crd  --include-namespace -v 10

.PHONY: install-kubebuildertool
install-kubebuildertool: ## Download kubebuildertool
	MINIMUM_KUBEBUILDERTOOL_VERSION=$(MINIMUM_KUBEBUILDERTOOL_VERSION) ./hack/ensure-kubebuildertool.sh


.PHONY: install-kind
install-kind: ## Download kind
        MINIMUM_KIND_VERSION=$(MINIMUM_KIND_VERSION) ./hack/ensure-kind.sh

LOCAL_KUSTOMIZE ?= $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download Kustomize
	MINIMUM_KUSTOMIZE_VERSION=$(MINIMUM_KUSTOMIZE_VERSION) hack/ensure-kustomize.sh

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	MINIMUM_ENVTEST_VERSION=$(MINIMUM_ENVTEST_VERSION) ./hack/ensure-envtest.sh

MOCKGEN = $(shell pwd)/bin/mockgen
.PHONY: mockgen
mockgen: ## Download mockgen locally if necessary.
        MINIMUM_MOCKGEN_VERSION=$(MINIMUM_MOCKGEN_VERSION) ./hack/ensure-mockgen.sh

ENVSUBST = $(shell pwd)/bin/envsubst
.PHONY: envsubst
envsubst: ## Download envsubst
	./hack/ensure-envsubst.sh 

GH = $(shell pwd)/bin/gh
.PHONY: gh
gh: ## Download gh
	MINIMUM_GH_VERSION=$(MINIMUM_GH_VERSION) ./hack/ensure-gh.sh
	

install-dev-prerequisites: ## Install clusterctl, controller-gen, envsubst, envtest, gh, go, helm, kind, kubectl, kustomize, packer, til 
	@echo "Start install all depencies"
	$(MAKE) install-clusterctl
	$(MAKE) controller-gen
	$(MAKE) verify-go
	$(MAKE) envsubst
	$(MAKE) mockgen
	$(MAKE) envtest
	$(MAKE) install-helm
	$(MAKE) install-kind
	$(MAKE) install-kubectl
	$(MAKE) kustomize
	$(MAKE) install-tilt
	@echo "Finished to install all dependencies"

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
