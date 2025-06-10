# Variables
BINARY_NAME=gohook
BUILD_DIR=./build
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

all: build ## Build binaries for all target platforms

build: clean build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 ## Build for all major platforms

build-js: ## Build the web UI
	(cd ui && yarn && yarn build)

test: ## Run Go tests
	go test -v ./...

deps: ## Install Go dependencies
	go get -d -v ./...

clean: ## Clean up build artifacts
	rm -rf $(BUILD_DIR)

# Cross-compilation targets
build-linux-amd64:
	@echo "--> Building for linux/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

build-linux-arm64:
	@echo "--> Building for linux/arm64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

build-windows-amd64:
	@echo "--> Building for windows/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

build-darwin-amd64:
	@echo "--> Building for darwin/amd64..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='$(LD_FLAGS)' -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .

package-zip: ## Package all builds into zip files
	@echo "--> Packaging binaries into zip files..."
	@for f in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		zip -j "$$f.zip" "$$f"; \
	done
	@echo "Done. Find packages in $(BUILD_DIR)"

.PHONY: all build build-js test deps clean build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 package-zip

