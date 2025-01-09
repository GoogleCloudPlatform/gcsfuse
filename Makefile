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

.DEFAULT_GOAL := build

.PHONY: generate imports fmt vet build buildTest install test clean-gen clean clean-all

generate:
	go generate ./...

imports: generate
	goimports -w .

fmt: imports
	go fmt ./...

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
	rm -rf cfg/config.go

clean: clean-gen
	go clean

clean-all: clean-gen
	go clean -i ./...
