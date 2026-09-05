GO ?= go
CGO_ENABLED ?= 1
CC ?= cc

GO_ENV := CGO_ENABLED=$(CGO_ENABLED) CC=$(CC)
GOLANGCI_LINT := $(GO_ENV) $(GO) tool golangci-lint

.DEFAULT_GOAL := precommit

.PHONY: lint test test-race mod-tidy precommit

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO_ENV) $(GO) test ./...

test-race:
	$(GO_ENV) $(GO) test -race ./...

mod-tidy:
	$(GO) mod tidy -diff

precommit: mod-tidy lint test test-race
