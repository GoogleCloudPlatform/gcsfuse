# Copyright 2024 Google LLC
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

CSI_VERSION ?= main
GCSFUSE_TAG ?= master
GCSFUSE_VERSION ?= $(shell HASH=$$(git rev-parse --short=6 HEAD 2>/dev/null); if [ -z "$$HASH" ]; then echo "unknown"; else if [ -n "$$(git status --porcelain)" ]; then echo "$$HASH-dirty"; else echo "$$HASH"; fi; fi)
GOLANG_VERSION := $(shell cat .go-version)
BUILD_ARM ?= true

# The following section is to set the value of STAGINGVERSION to be used in build-csi target.
# Define the mandatory prefix, needed to allow passing machine-type from gke csi driver to gcsfuse,
# bypassing the check at
# https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver/blob/15afd00dcc2cfe0f9753ddc53c81631ff037c3f2/pkg/csi_driver/utils.go#L532.
STAGINGVERSIONPREFIX := prow-gob-internal-boskos-
# Define the fallback logic in case uuidgen is not available.
# 1. Try 'uuidgen'.
# 2. If 'uuidgen' fails or is missing, construct: [GitHash][Dirty?]-[Epoch]
# Note: We use '=' so this shell command only executes if STAGINGVERSION was not provided.
_STAGINGVERSION_FALLBACK = $(shell \
	uuidgen 2>/dev/null || \
	echo "$$(git rev-parse --short HEAD)$$(git diff --quiet HEAD || echo '+')-$$(date +%s)" \
)
# Apply default if not provided by user
STAGINGVERSION ?= $(_STAGINGVERSION_FALLBACK)
# Enforce the prefix (Idempotent: removes prefix if present, then adds it)
override STAGINGVERSION := $(STAGINGVERSIONPREFIX)$(patsubst $(STAGINGVERSIONPREFIX)%,%,$(STAGINGVERSION))

# Active gcloud project
PROJECT ?= $(shell gcloud config get-value project 2>/dev/null)

# Cluster configuration derived from active kubectl context (gke_PROJECT_LOCATION_CLUSTERNAME)
CLUSTER_PROJECT ?= $(shell kubectl config current-context 2>/dev/null | awk -F'_' '{print $$2}')
CLUSTER_NAME ?= $(shell kubectl config current-context 2>/dev/null | awk -F'_' '{print $$NF}')
CLUSTER_LOCATION ?= $(shell kubectl config current-context 2>/dev/null | awk -F'_' '{print $$(NF-1)}')
OVERLAY ?= stable
.DEFAULT_GOAL := build

.PHONY: help generate imports fmt vet lint build buildTest install test clean-gen clean clean-all build-csi install-custom-csi uninstall-custom-csi install-managed-csi uninstall-managed-csi install-csi uninstall-csi

help:
	@echo "Usage: make [target] [VARIABLE=value...]"
	@echo ""
	@echo "Available Targets:"
	@echo "  build                  - Lint and compile gcsfuse binaries (default)"
	@echo "  install                - Install gcsfuse binaries locally to \$$GOPATH/bin"
	@echo "  test                   - Run unit tests"
	@echo "  lint                   - Run golangci-lint and go vet"
	@echo "  vet                    - Run go vet static analysis"
	@echo "  fmt                    - Format Go source code and run go mod tidy"
	@echo "  imports                - Organize Go imports and run go generate"
	@echo "  generate               - Run go generate for config files"
	@echo "  buildTest              - Compile test packages without executing tests"
	@echo "  clean                  - Remove build artifacts and caches"
	@echo "  clean-all              - Remove build artifacts and all installed binaries"
	@echo "  build-csi              - Build and stage CSI driver images via Google Cloud Build"
	@echo "  install-custom-csi     - Build and install custom CSI driver onto the target GKE cluster (alias: install-csi)"
	@echo "  uninstall-custom-csi   - Uninstall custom CSI driver from the target GKE cluster (alias: uninstall-csi)"
	@echo "  install-managed-csi    - Install GKE managed CSI driver on the target GKE cluster"
	@echo "  uninstall-managed-csi  - Uninstall GKE managed CSI driver from the target GKE cluster"
	@echo "  e2e-test               - Run end-to-end integration tests"
	@echo ""
	@echo "Configuration Options & Variables:"
	@echo "  PROJECT          - GCP project ID for Cloud Build and e2e test bucket creation"
	@echo "                     (default: from active gcloud config: $(PROJECT))"
	@echo "  CLUSTER_PROJECT  - GCP project ID hosting the target GKE cluster"
	@echo "                     (default: from active kubectl context: $(CLUSTER_PROJECT))"
	@echo "  CLUSTER_NAME     - Name of the target GKE cluster for CSI install/uninstall"
	@echo "                     (default: from active kubectl context: $(CLUSTER_NAME))"
	@echo "  CLUSTER_LOCATION - Zone/region of the target GKE cluster (e.g., us-central1-c)"
	@echo "                     (default: from active kubectl context: $(CLUSTER_LOCATION))"
	@echo "  CSI_VERSION      - Git branch or tag of gcs-fuse-csi-driver to use"
	@echo "                     (default: $(CSI_VERSION))"
	@echo "  GCSFUSE_TAG      - Git branch or tag of gcsfuse to build in CSI driver"
	@echo "                     (default: $(GCSFUSE_TAG))"
	@echo "  OVERLAY          - Kustomize overlay to deploy during CSI installation (e.g. stable, dev)"
	@echo "                     (default: $(OVERLAY))"
	@echo "  GCSFUSE_VERSION  - gcsfuse version string to embed in Cloud Build CSI driver build"
	@echo "                     (default: $(GCSFUSE_VERSION))"
	@echo "  GOLANG_VERSION   - Go compiler version used for container image builds"
	@echo "                     (default: $(GOLANG_VERSION))"
	@echo "  BUILD_ARM        - Build multi-arch ARM64 images alongside AMD64 (true/false)"
	@echo "                     (default: $(BUILD_ARM))"
	@echo "  STAGINGVERSION   - Image tag version used for staging built CSI driver images"
	@echo "                     (default: $(STAGINGVERSION))"
	@echo "  ARGS             - Additional arguments and flags passed to improved_run_e2e_tests.sh"
	@echo "                     (e.g. make e2e-test ARGS=\"--help\", make e2e-test ARGS=\"--run-package operations --presubmit\")"

generate:
	go generate ./...

imports: generate
	goimports -w .

fmt: imports
	go mod tidy && gofmt -s -w .

vet: fmt
	go vet ./...

lint: vet
	golangci-lint run -E=unused --timeout 3m0s --new-from-rev=master

build: lint
	go build ./...

buildTest: vet
	go test -run=PATTERN_THAT_DOES_NOT_MATCH_ANYTHING ./...

install: fmt
	go install -v ./...

test: fmt
	CGO_ENABLED=0 go test -timeout 5m -count 1 `go list ./... | grep -v internal/cache/...` && CGO_ENABLED=0 go test -timeout 5m -p 1 -count 1 ./internal/cache/...

clean-gen:
	rm -rf cfg/config.go cfg/config_test.go

clean: clean-gen
	go clean

clean-all: clean-gen
	go clean -i ./...

build-csi:
	@echo "--------------------------------------"
	@echo "Starting build for version: $(STAGINGVERSION)"
	@echo "--------------------------------------"
	# Actual build commands would go here...
	gcloud builds submit --config csi_driver_build.yml --project=$(PROJECT) --substitutions=_GOLANG_VERSION=$(GOLANG_VERSION),_CSI_VERSION=$(CSI_VERSION),_GCSFUSE_VERSION=$(GCSFUSE_VERSION),_BUILD_ARM=$(BUILD_ARM),_STAGINGVERSION=$(STAGINGVERSION)

install-custom-csi:
	@tools/scripts/csi/install_custom.sh "$(CLUSTER_PROJECT)" "$(CLUSTER_NAME)" "$(CLUSTER_LOCATION)" "$(GCSFUSE_TAG)" "$(CSI_VERSION)" "$(OVERLAY)"

install-csi: install-custom-csi

uninstall-custom-csi:
	@tools/scripts/csi/uninstall_custom.sh "$(CLUSTER_PROJECT)" "$(CLUSTER_NAME)" "$(CLUSTER_LOCATION)" "$(CSI_VERSION)"

uninstall-csi: uninstall-custom-csi

install-managed-csi:
	@tools/scripts/csi/install_managed.sh "$(CLUSTER_PROJECT)" "$(CLUSTER_NAME)" "$(CLUSTER_LOCATION)"

uninstall-managed-csi:
	@tools/scripts/csi/uninstall_managed.sh "$(CLUSTER_PROJECT)" "$(CLUSTER_NAME)" "$(CLUSTER_LOCATION)"

e2e-test:
	@tools/integration_tests/improved_run_e2e_tests.sh $(if $(PROJECT),--project-id $(PROJECT)) $(ARGS)
