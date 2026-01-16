.PHONY: test test-verbose test-coverage build clean help

# Detect platform
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Map to Go platform names
ifeq ($(UNAME_S),Darwin)
    OS := darwin
endif
ifeq ($(UNAME_S),Linux)
    OS := linux
endif

ifeq ($(UNAME_M),x86_64)
    ARCH := amd64
else ifeq ($(UNAME_M),arm64)
    ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
    ARCH := arm64
endif

# Platform-specific library settings
PLATFORM := $(OS)_$(ARCH)
REPO_ROOT := $(shell pwd)
LIB_DIR := $(REPO_ROOT)/lib/$(PLATFORM)
INCLUDE_DIR := $(REPO_ROOT)/include

# CGO flags for LanceDB
export CGO_ENABLED := 1
export CGO_CFLAGS := -I$(INCLUDE_DIR)

ifeq ($(OS),darwin)
    export CGO_LDFLAGS := -L$(LIB_DIR) -llancedb_go -framework Security -framework CoreFoundation -Wl,-rpath,$(LIB_DIR)
    export DYLD_LIBRARY_PATH := $(LIB_DIR):$(DYLD_LIBRARY_PATH)
else
    export CGO_LDFLAGS := -L$(LIB_DIR) -llancedb_go -Wl,-rpath,$(LIB_DIR)
    export LD_LIBRARY_PATH := $(LIB_DIR):$(LD_LIBRARY_PATH)
endif

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

test: ## Run all tests
	@echo "Running tests for platform: $(PLATFORM)"
	@echo "CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "CGO_LDFLAGS: $(CGO_LDFLAGS)"
	@echo "DYLD_LIBRARY_PATH: $(DYLD_LIBRARY_PATH)"
	@echo ""
	go test $$(go list ./... | grep -v /examples/)

test-verbose: ## Run tests with verbose output
	@echo "Running tests for platform: $(PLATFORM)"
	@echo "CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "CGO_LDFLAGS: $(CGO_LDFLAGS)"
	@echo "DYLD_LIBRARY_PATH: $(DYLD_LIBRARY_PATH)"
	@echo ""
	go test -v $$(go list ./... | grep -v /examples/)

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage for platform: $(PLATFORM)"
	@echo "CGO_CFLAGS: $(CGO_CFLAGS)"
	@echo "CGO_LDFLAGS: $(CGO_LDFLAGS)"
	@echo "DYLD_LIBRARY_PATH: $(DYLD_LIBRARY_PATH)"
	@echo ""
	go test -coverprofile=coverage.out $$(go list ./... | grep -v /examples/)
	go tool cover -func=coverage.out
	@echo ""
	@echo "For detailed HTML coverage report, run: go tool cover -html=coverage.out"

test-integration: ## Run only integration tests
	@echo "Running integration tests for platform: $(PLATFORM)"
	go test -v ./cmd/code-scout/... ./internal/chunker/integration_test.go

build: ## Build the project using build.sh
	./build.sh

clean: ## Clean build artifacts and test caches
	rm -rf dist/
	rm -f coverage.out
	go clean -testcache
	go clean -cache

.DEFAULT_GOAL := help
