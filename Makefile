SHELL := /usr/bin/env bash

# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.SUFFIXES:
.SECONDARY:

include Makefile.vars.mk

.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "(: ).*?## "}; {gsub(/\\:/,":",$$1)}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: export GOOS = linux
build: fmt vet ## Build the Go binary
	@go build .

.PHONY: fmt
fmt: ## Run 'go fmt' against code
	go fmt ./...

.PHONY: vet
vet: ## Run 'go vet' against code
	go vet ./...

.PHONY: lint
lint: fmt vet ## Invokes the fmt and vet targets
	@echo 'Check for uncommitted changes ...'
	git diff --exit-code

.PHONY: build\:docker
build\:docker: export CGO_ENABLED = 0
build\:docker:
build\:docker: build ## Build the docker image
	docker build . -t $(DOCKER_IMG) -t $(QUAY_IMG)

.PHONY: clean
clean: ## Clean the project
	@rm -rf fronius-exporter cover.out dist

.PHONY: test
test: ## Run unit tests
	@go test -coverprofile cover.out ./...

.PHONY: run
run: ## Run locally
	@go run . -v
