OS = darwin freebsd linux openbsd
ARCHS = 386 arm amd64 arm64

.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

all: build release release-windows

build: deps ## Build the project
	go build

build-js:
	(cd ui && NODE_OPTIONS="${NODE_OPTIONS}" yarn build)

release: clean deps ## Generate releases for unix systems
	@for arch in $(ARCHS);\
	do \
		for os in $(OS);\
		do \
			echo "Building $$os-$$arch"; \
			mkdir -p build/gohook-$$os-$$arch/; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o build/gohook-$$os-$$arch/gohook; \
			tar cz -C build -f build/gohook-$$os-$$arch.tar.gz gohook-$$os-$$arch; \
		done \
	done

release-windows: clean deps ## Generate release for windows
	@for arch in $(ARCHS);\
	do \
		echo "Building windows-$$arch"; \
		mkdir -p build/gohook-windows-$$arch/; \
		GOOS=windows GOARCH=$$arch go build -o build/gohook-windows-$$arch/gohook.exe; \
		tar cz -C build -f build/gohook-windows-$$arch.tar.gz gohook-windows-$$arch; \
	done

test: deps ## Execute tests
	go test ./...

deps: ## Install dependencies using go get
	go get -d -v -t ./...

clean: ## Remove building artifacts
	rm -rf build
	rm -f gohook
