# ContextSubstrate Makefile

BINARY     := ctx
MODULE     := github.com/contextsubstrate/ctx
CMD_PKG    := $(MODULE)/cmd/ctx

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

GO       := go
GOFLAGS  :=
TESTFLAGS := -race -count=1

.PHONY: all build install test coverage lint vet fmt tidy clean help

all: build ## Build the binary (default)

build: ## Compile the ctx binary
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) $(CMD_PKG)

install: ## Install ctx to $GOPATH/bin
	$(GO) install $(GOFLAGS) -ldflags '$(LDFLAGS)' $(CMD_PKG)

test: ## Run all tests
	$(GO) test $(TESTFLAGS) ./...

coverage: ## Run tests with coverage report
	$(GO) test $(TESTFLAGS) -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Check that code is formatted
	@test -z "$$(gofmt -l .)" || { echo "Run 'gofmt -w .' to fix formatting:"; gofmt -l .; exit 1; }

tidy: ## Tidy go.mod
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -f coverage.out

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
