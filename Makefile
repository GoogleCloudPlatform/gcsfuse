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
BUILD_ARM ?= true
STAGINGVERSION ?=
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
	cd .. && gcloud builds submit --config gcsfuse/csi_driver_build.yml --project=$(PROJECT) --substitutions=_GCSFUSE_VERSION=$(GCSFUSE_VERSION),_BUILD_ARM=$(BUILD_ARM),_STAGINGVERSION=$(STAGINGVERSION),_USER=cloudbuild-prow-gob-internal-boskos
