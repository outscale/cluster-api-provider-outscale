
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
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

.PHONY: testenv
testenv:
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

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	mkdir -p $(shell pwd)/bin
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

LOCAL_CLUSTERCTL ?= $(shell pwd)/bin/clusterctl
.PHONY: install-clusterctl
install-clusterctl: ## Download clusterctl locally if necessary.
	mkdir -p $(shell pwd)/bin
	wget -c https://github.com/kubernetes-sigs/cluster-api/releases/download/${CAPI_VERSION}/clusterctl-linux-amd64	
	mv $(shell pwd)/clusterctl-linux-amd64 $(shell pwd)/bin/clusterctl
	chmod +x  $(LOCAL_CLUSTERCTL)

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
