GO ?= go
CGO_ENABLED ?= 1
CC ?= cc
FUZZ_TIME ?= 10s

GO_ENV := CGO_ENABLED=$(CGO_ENABLED) CC=$(CC)
GOLANGCI_LINT := $(GO_ENV) $(GO) tool golangci-lint

.DEFAULT_GOAL := precommit

.PHONY: lint test test-race bench fuzz mod-tidy precommit

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO_ENV) $(GO) test ./...

test-race:
	$(GO_ENV) $(GO) test -race ./...

bench:
	$(GO_ENV) $(GO) test -run '^$$' -bench . -benchmem ./...

fuzz:
	$(GO_ENV) $(GO) test -run '^$$' -fuzz '^FuzzEngineValidate$$' -fuzztime $(FUZZ_TIME) .

mod-tidy:
	$(GO) mod tidy -diff

precommit: mod-tidy lint test test-race
