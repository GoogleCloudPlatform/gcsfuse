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

PROJECT ?= $(shell gcloud config get-value project 2>/dev/null)
.DEFAULT_GOAL := build

.PHONY: generate imports fmt vet build buildTest install test clean-gen clean clean-all build-csi

generate:
	go generate ./...

imports: generate
	goimports -w .

fmt: imports
	go mod tidy && go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build .

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

e2e-test:
	ZONE=$$(curl -H "Metadata-Flavor: Google" metadata.google.internal/computeMetadata/v1/instance/zone | awk -F'/' '{print $$NF}'); \
	echo $$ZONE; \
	REGION=$$(echo $$ZONE | sed 's/-[a-z]$$//'); \
	echo $$REGION; \
	tools/integration_tests/improved_run_e2e_tests.sh --bucket-location $$REGION
