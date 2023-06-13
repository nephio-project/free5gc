GO_VERSION ?= 1.20.5
GOLANG_CI_VER ?= v1.52
GOSEC_VER ?= 2.15.0
TEST_COVERAGE_FILE=lcov.info
TEST_COVERAGE_HTML_FILE=coverage_unit.html
TEST_COVERAGE_FUNC_FILE=func_coverage.out
ignore-not-found ?= false

# CONTAINER_RUNNABLE checks if tests and lint check can be run inside container.
PODMAN ?= $(shell podman -v > /dev/null 2>&1; echo $$?)
ifeq ($(PODMAN), 0)
CONTAINER_RUNTIME=podman
else
CONTAINER_RUNTIME=docker
endif
CONTAINER_RUNNABLE ?= $(shell $(CONTAINER_RUNTIME) -v > /dev/null 2>&1; echo $$?)

# Use microk8s if installed
ifeq (,$(shell command -v microk8s 2> /dev/null))
KUBECTL="kubectl"
else
KUBECTL="microk8s kubectl"
endif

REGISTRY ?= docker.io/nephio
PROJECT ?= free5gc-operator
TAG ?= latest

# Image URL to use all building/pushing image targets
ifeq (,$(REGISTRY))
IMG ?= $(PROJECT):$(TAG)
else
IMG ?= $(REGISTRY)/$(PROJECT):$(TAG)
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.27.1

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
# SHELL = /usr/bin/env bash -o pipefail
# .SHELLFLAGS = -ec

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
	$(CONTROLLER_GEN) rbac:roleName=free5gc-operator-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

.PHONY: unit
unit: ## Run unit tests against code.
ifeq ($(CONTAINER_RUNNABLE), 0)
	$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/library/golang:${GO_VERSION}-alpine3.17 \
	/bin/sh -c "go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}; \
	go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE}; \
	go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}"
else
	go test ./... -v -coverprofile ${TEST_COVERAGE_FILE}
	go tool cover -html=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_HTML_FILE}
	go tool cover -func=${TEST_COVERAGE_FILE} -o ${TEST_COVERAGE_FUNC_FILE}
endif

# Install link at https://golangci-lint.run/usage/install/ if not running inside a container
.PHONY: lint
lint: ## Run lint  against code.
ifeq ($(CONTAINER_RUNNABLE), 0)
	$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/golangci/golangci-lint:${GOLANG_CI_VER}-alpine golangci-lint run ./... -v --timeout 10m
else
	golangci-lint run ./... -v --timeout 10m
endif

# Install link at https://github.com/securego/gosec#install if not running inside a container
.PHONY: gosec
gosec: ## inspects source code for security problem by scanning the Go Abstract Syntax Tree
ifeq ($(CONTAINER_RUNNABLE), 0)
	$(CONTAINER_RUNTIME) run -it -v ${PWD}:/go/src -w /go/src docker.io/securego/gosec:${GOSEC_VER} ./...
else
	gosec ./...
endif

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# Broken
#
# KPTGEN_CFG_DIR = package/fn-config
# KPT_PKG_DIR = package/$(PROJECT)
#
# .PHONY: package
# package: controller-gen kpt kptgen $(KPTGEN_CFG_DIR)
# 	rm -rf ${KPT_PKG_DIR}/
# 	mkdir ${KPT_PKG_DIR}
# 	$(KPT) pkg init ${KPT_PKG_DIR} --description "${PROJECT} controller"
# 	$(CONTROLLER_GEN) crd paths="./..." output:crd:dir=${KPT_PKG_DIR}/crds
# 	$(KPTGEN) apply config ${KPT_PKG_DIR} --fn-config-dir ${KPTGEN_CFG_DIR}
# 	$(KPT) fn eval --image gcr.io/kpt-fn/set-image:v0.1.1 ${KPT_PKG_DIR} -- name=controller newName=${REGISTRY}/${PROJECT} newTag=${TAG}
# 	$(KPT) fn eval --image gcr.io/kpt-fn/set-namespace:v0.1.1 ${KPT_PKG_DIR} -- namespace=free5gc

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
#KPT ?= $(LOCALBIN)/kpt
#KPTGEN ?= $(LOCALBIN)/kptgen

## Tool Versions
KUSTOMIZE_VERSION ?= v5.0.1
CONTROLLER_TOOLS_VERSION ?= v0.11.4
#KPT_VERSION ?= main
#KPTGEN_VERSION ?= v0.0.9

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# .PHONY: kpt
# kpt: $(KPT) ## Download kpt locally if necessary.
# $(KPT): $(LOCALBIN)
# 	test -s $(LOCALBIN)/kpt || GOBIN=$(LOCALBIN) go install -v github.com/GoogleContainerTools/kpt@$(KPT_VERSION)

# .PHONY: kptgen
# kptgen: $(KPTGEN)
# $(KPTGEN): $(LOCALBIN)
# 	test -s $(LOCALBIN)/kptgen || GOBIN=$(LOCALBIN) go install -v github.com/henderiw-kpt/kptgen@$(KPTGEN_VERSION)
