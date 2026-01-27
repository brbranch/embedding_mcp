VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
BINARY := mcp-memory

.PHONY: all build test clean release-dry-run install

all: test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/mcp-memory

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist/

release-dry-run:
	goreleaser release --snapshot --clean

install: build
	cp $(BINARY) $(GOPATH)/bin/

.DEFAULT_GOAL := build
