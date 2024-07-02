.DEFAULT_GOAL := build

.PHONY: generate fmt vet build buildTest install test clean clean-all

generate:
	go generate ./...

fmt: generate
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
	CGO_ENABLED=0 go test -count 1 -v `go list ./... | grep -v internal/cache/...` && CGO_ENABLED=0 go test -p 1 -count 1 -v ./internal/cache/...

clean:
	go clean

clean-all:
	go clean -i ./...
