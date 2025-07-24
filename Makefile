# ---- config ------------------------------------------------------------------

.DEFAULT_GOAL := help

SHELL := /usr/bin/env bash -o pipefail

PACKAGES := $(shell go list ./...)

MAX_COL := 100

# ---- meta --------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---- tooling -----------------------------------------------------------------

.PHONY: install-tools
install-tools: ## Go install external tools
	go install github.com/segmentio/golines@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

# ---- formatting / linting ----------------------------------------------------

.PHONY: fmt
fmt: ## Run gofmt
	go fmt ./...

.PHONY: golines
golines: ## Reflow long lines to $(MAX_COL) cols
	golines -w -m $(MAX_COL) .

.PHONY: vet
vet: ## Run go vet 
	go vet $(PACKAGES)

.PHONY: staticcheck
staticcheck: ## Run staticcheck (honnef.co)
	staticcheck $(PACKAGES)

.PHONY: lint
lint: fmt golines vet staticcheck ## Run all linters/formatters

# ---- tests / build -----------------------------------------------------------

.PHONY: test
test: ## Run unit tests
	go test -count=1 $(PACKAGES)

.PHONY: test-race
test-race: ## Run tests with the race detector
	go test -race -count=1 $(PACKAGES)

.PHONY: build
build: ## Build all packages
	go build ./...

.PHONY: tidy
tidy: ## Ensure go.mod/go.sum are tidy
	go mod tidy

# ---- pipelines ---------------------------------------------------------------

.PHONY: ci 
ci: tidy lint test-race 