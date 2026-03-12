VERSION ?= 0.1.0
BINARY := raff
BUILD_DIR := ./bin
LDFLAGS := -ldflags "-X github.com/rafftechnologies/raff-cli/internal/client.Version=$(VERSION)"

.PHONY: build install clean fmt lint test

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/raff

install: build
	cp $(BUILD_DIR)/$(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

test:
	go test ./... -v
