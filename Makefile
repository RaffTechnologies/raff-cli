VERSION ?= 0.1.0
BINARY := raff
BUILD_DIR := ./bin
LDFLAGS := -ldflags "-X github.com/rafftechnologies/raff-cli/internal/commands.Version=$(VERSION)"

.PHONY: build install clean fmt lint test vet sync

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/raff

install: build
	cp $(BUILD_DIR)/$(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

fmt:
	go fmt ./...

lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

test:
	go test ./...

vet:
	go vet ./...

# Build, vet, test against the current raff-go.
sync: build vet test
