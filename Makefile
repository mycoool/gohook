# Variables
BINARY_NAME?=gohook
AGENT_BINARY_NAME?=gohook-agent
BUILD_DIR?=./build
SHELL := /bin/bash

# Default Go toolchain to local if GOTOOLCHAIN is set, otherwise extract from go.mod
ifdef GOTOOLCHAIN
	GO_VERSION=$(GOTOOLCHAIN)
else
	GO_VERSION=$(shell go mod edit -json | jq -r .Toolchain | sed -e 's/go//')
endif

.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: build ## Build server binary by default

build: clean build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 ## Build server for all major platforms

agent: AGENT=1
agent: clean build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 ## Build sync node agent binaries

build-js: ## Build the web UI
	(cd ui && yarn && yarn build)

check-go:
	golangci-lint run

check-js:
	(cd ui && yarn lint)
	(cd ui && yarn testformat)

test: ## Run Go tests
	go test -v ./...

deps: ## Install Go dependencies
	go get -d -v ./...

clean: ## Clean up build artifacts
	rm -rf $(BUILD_DIR)

ifeq ($(AGENT),1)
	OUTPUT_NAME=$(AGENT_BINARY_NAME)
	BUILD_TARGET=./cmd/nodeclient
else
	OUTPUT_NAME=$(BINARY_NAME)
	BUILD_TARGET=.
endif

GO_BUILD_CMD?=go build

# Cross-compilation targets
build-linux-amd64:
	@echo "--> Building for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO_BUILD_CMD) -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(OUTPUT_NAME)-linux-amd64$(EXT) $(BUILD_TARGET)

build-linux-arm64:
	@echo "--> Building for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO_BUILD_CMD) -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(OUTPUT_NAME)-linux-arm64$(EXT) $(BUILD_TARGET)

build-windows-amd64:
	@echo "--> Building for windows/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO_BUILD_CMD) -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(OUTPUT_NAME)-windows-amd64.exe $(BUILD_TARGET)

build-darwin-amd64:
	@echo "--> Building for darwin/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO_BUILD_CMD) -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(OUTPUT_NAME)-darwin-amd64$(EXT) $(BUILD_TARGET)

package-zip: ## Package all builds into zip files
	@echo "--> Packaging binaries into zip files..."
	@for f in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		zip -j "$$f.zip" "$$f"; \
	done
	@echo "Done. Find packages in $(BUILD_DIR)"

.PHONY: all build build-js test deps clean build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 package-zip
