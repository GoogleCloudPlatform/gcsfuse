.DEFAULT_GOAL := build

.PHONY: generate fmt vet build install test clean clean-all

generate:
	go generate ./...

fmt: generate
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build main.go

install: fmt
	go install -v ./...

test: fmt
	CGO_ENABLED=0 go test -p 1 -count 1 -v ./...

clean:
	go clean

clean-all:
	go clean -i ./...

