# This file is managed by greposync.
# Do not modify manually.
# Adjust variables in `.sync.yml`.
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
build: fmt vet $(BIN_FILENAME) ## Build the Go binary

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

.PHONY: build.docker
build.docker: export CGO_ENABLED = 0
build.docker: build ## Build the docker image
	docker build --tag $(LOCAL_IMG)	.

.PHONY: clean
clean: ## Clean the project
	@rm -rf $(BIN_FILENAME) cover.out dist

.PHONY: test
test: ## Run unit tests
	@go test -coverprofile cover.out -v ./...

.PHONY: run
run: ## Run locally
	@go run . -v

###
### Assets
###

.PHONY: $(BIN_FILENAME)
$(BIN_FILENAME):
	@go build -o $(BIN_FILENAME) .
