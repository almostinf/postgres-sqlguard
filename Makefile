GO ?= go
CGO_ENABLED ?= 1
CC ?= cc
FUZZ_TIME ?= 10s
NO_CGO_CC ?= /definitely/missing/postgres-sqlguard-cc
SIZE_PROBE_DIR ?= /tmp/postgres-sqlguard-size-probe
SIZE_PROBE_LDFLAGS ?= -s -w -buildid=

GO_ENV := CGO_ENABLED=$(CGO_ENABLED) CC=$(CC)
GOLANGCI_LINT := $(GO_ENV) $(GO) tool golangci-lint

.DEFAULT_GOAL := precommit

.PHONY: lint lint-cgo lint-no-cgo lint-all \
	test test-cgo test-no-cgo test-all \
	test-race test-race-cgo test-race-no-cgo test-race-all \
	bench bench-cgo bench-no-cgo bench-all \
	fuzz fuzz-cgo fuzz-no-cgo fuzz-all \
	size-probe size-probe-cgo size-probe-no-cgo size-probe-dir \
	mod-tidy precommit precommit-cgo precommit-no-cgo precommit-all

lint:
	$(GOLANGCI_LINT) run ./...

lint-cgo:
	$(MAKE) lint CGO_ENABLED=1 CC="$(CC)"

lint-no-cgo:
	$(MAKE) lint CGO_ENABLED=0 CC="$(NO_CGO_CC)"

lint-all: lint-cgo lint-no-cgo

test:
	$(GO_ENV) $(GO) test ./...

test-cgo:
	$(MAKE) test CGO_ENABLED=1 CC="$(CC)"

test-no-cgo:
	$(MAKE) test CGO_ENABLED=0 CC="$(NO_CGO_CC)"

test-all: test-cgo test-no-cgo

test-race:
	$(GO_ENV) $(GO) test -race ./...

test-race-cgo:
	$(MAKE) test-race CGO_ENABLED=1 CC="$(CC)"

test-race-no-cgo:
	$(MAKE) test-race CGO_ENABLED=0 CC="$(NO_CGO_CC)"

test-race-all: test-race-cgo test-race-no-cgo

bench:
	$(GO_ENV) $(GO) test -run '^$$' -bench . -benchmem ./...

bench-cgo:
	$(MAKE) bench CGO_ENABLED=1 CC="$(CC)"

bench-no-cgo:
	$(MAKE) bench CGO_ENABLED=0 CC="$(NO_CGO_CC)"

bench-all: bench-cgo bench-no-cgo

fuzz:
	$(GO_ENV) $(GO) test -run '^$$' -fuzz '^FuzzEngineValidate$$' -fuzztime $(FUZZ_TIME) .

fuzz-cgo:
	$(MAKE) fuzz CGO_ENABLED=1 CC="$(CC)"

fuzz-no-cgo:
	$(MAKE) fuzz CGO_ENABLED=0 CC="$(NO_CGO_CC)"

fuzz-all: fuzz-cgo fuzz-no-cgo

size-probe-dir:
	mkdir -p "$(SIZE_PROBE_DIR)"

size-probe-cgo: size-probe-dir
	CGO_ENABLED=1 CC="$(CC)" $(GO) build -trimpath -ldflags '$(SIZE_PROBE_LDFLAGS)' -o "$(SIZE_PROBE_DIR)/sqlguard-cgo" ./internal/testutil/cmd/sizeprobe

size-probe-no-cgo: size-probe-dir
	CGO_ENABLED=0 CC="$(NO_CGO_CC)" $(GO) build -trimpath -ldflags '$(SIZE_PROBE_LDFLAGS)' -o "$(SIZE_PROBE_DIR)/sqlguard-no-cgo" ./internal/testutil/cmd/sizeprobe

size-probe: size-probe-cgo size-probe-no-cgo

mod-tidy:
	$(GO) mod tidy -diff

precommit: mod-tidy lint test test-race

precommit-cgo: mod-tidy lint-cgo test-cgo test-race-cgo

precommit-no-cgo: mod-tidy lint-no-cgo test-no-cgo test-race-no-cgo

precommit-all: mod-tidy lint-all test-all test-race-all
