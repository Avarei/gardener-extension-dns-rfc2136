# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-FileCopyrightText: 2024 metal-stack Authors
#
# SPDX-License-Identifier: Apache-2.0

EXTENSION_PREFIX            := gardener-extension
NAME                        := dns-rfc2136
IMAGE_TAG                   := $(or ${GITHUB_TAG_NAME}, latest)
REGISTRY                    := ghcr.io/avarei
IMAGE_PREFIX                := $(REGISTRY)
REPO_ROOT                   := $(shell dirname "$(realpath $(lastword $(MAKEFILE_LIST)))")
HACK_DIR                    := $(REPO_ROOT)/hack
HOSTNAME                    := $(shell hostname)
LD_FLAGS                    := "-w -X github.com/avarei/$(EXTENSION_PREFIX)-$(NAME)/pkg/version.Version=$(IMAGE_TAG)"
IGNORE_OPERATION_ANNOTATION := false
GO_VERSION                  := 1.22
GOLANGCI_LINT_VERSION       := v1.56.2

EXTENSION_NAMESPACE	:= garden
GARDEN_KUBECONFIG ?=

ifeq ($(CI),true)
  DOCKER_TTY_ARG=""
else
  DOCKER_TTY_ARG=t
endif

export GO111MODULE := on

TOOLS_DIR := hack/tools
-include vendor/github.com/gardener/gardener/hack/tools.mk

#########################################
# Rules for local development scenarios #
#########################################

.PHONY: start
start:
	@LEADER_ELECTION_NAMESPACE=$(EXTENSION_NAMESPACE) GARDEN_KUBECONFIG=$(GARDEN_KUBECONFIG) go run \
		-ldflags $(LD_FLAGS) \
		./cmd/$(EXTENSION_PREFIX)-$(NAME) \
		--config-file=./example/00-componentconfig.yaml \
		--ignore-operation-annotation=$(IGNORE_OPERATION_ANNOTATION) \
		--leader-election=false \
		--gardener-version="v1.39.0" \
		--heartbeat-namespace=$(EXTENSION_NAMESPACE) \
		--heartbeat-renew-interval-seconds=30 \
		--metrics-bind-address=:8080 \
		--health-bind-address=:8081

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: build
build:
	go build -ldflags $(LD_FLAGS) -tags netgo ./cmd/$(EXTENSION_PREFIX)-$(NAME)

.PHONY: install
install: revendor $(HELM)
	@LD_FLAGS="-w -X github.com/gardener/$(EXTENSION_PREFIX)-$(NAME)/pkg/version.Version=$(VERSION)" \
	$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install.sh ./...

.PHONY: docker-image
docker-image:
	@docker build --no-cache \
		--tag $(IMAGE_PREFIX)/$(EXTENSION_PREFIX)-$(NAME):$(IMAGE_TAG) \
		--file Dockerfile --memory 6g .

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_PREFIX)/$(EXTENSION_PREFIX)-$(NAME):$(IMAGE_TAG)

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/...

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts

.PHONY: generate
generate: $(HELM) $(YQ)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/generate.sh ./charts/... ./cmd/... ./example/... ./pkg/...

.PHONY: generate-in-container
generate-in-container: revendor $(HELM)
#	echo $(shell git describe --abbrev=0 --tags) > VERSION
	docker run --rm -i$(DOCKER_TTY_ARG) \
		--env GOCACHE=/gocache \
		--mount type=tmpfs,destination=/gocache,tmpfs-mode=1777 \
		--user $$(id -u):$$(id -g) \
		--userns=keep-id \
		--volume $(PWD):/go/src/github.com/avarei/$(EXTENSION_PREFIX)-$(NAME):z \
		--workdir /go/src/github.com/avarei/$(EXTENSION_PREFIX)-$(NAME) \
		golang:$(GO_VERSION) \
		/usr/bin/make generate

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg

.PHONY: verify
verify: check format
