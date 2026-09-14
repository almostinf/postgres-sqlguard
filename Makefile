GO ?= go
CGO_ENABLED ?= 1
CC ?= cc
FUZZ_TIME ?= 10s
NO_CGO_CC ?= /definitely/missing/postgres-sqlguard-cc
SIZE_PROBE_DIR ?= /tmp/postgres-sqlguard-size-probe
SIZE_PROBE_LDFLAGS ?= -s -w -buildid=
RELEASE_BENCH_DATE ?= 2026-09-14
RELEASE_BENCH_DIR ?= docs/benchmarks/$(RELEASE_BENCH_DATE)
RELEASE_BENCH_RAW_DIR ?= $(RELEASE_BENCH_DIR)/raw
RELEASE_BENCH_DATA ?= $(RELEASE_BENCH_DIR)/dataset.json
RELEASE_BENCH_CHART_DIR ?= docs/assets/benchmarks
RELEASE_BENCH_COUNT ?= 5
RELEASE_BENCH_TIME ?= 1s
COVERAGE_DIR ?= coverage
COVERAGE_PROFILE ?= $(COVERAGE_DIR)/coverage.out

GO_ENV := CGO_ENABLED=$(CGO_ENABLED) CC=$(CC)
GOLANGCI_LINT := $(GO_ENV) $(GO) tool golangci-lint

.DEFAULT_GOAL := precommit

.PHONY: lint lint-cgo lint-no-cgo lint-all \
	test test-cgo test-no-cgo test-all \
	test-race test-race-cgo \
	coverage coverage-cgo \
	bench bench-cgo bench-no-cgo bench-all \
	release-bench-cgo release-bench-no-cgo release-bench-footprint release-bench-capture \
	release-bench-charts release-bench-check \
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

coverage:
	mkdir -p "$(COVERAGE_DIR)"
	set -eu; raw_profile=$$(mktemp "$(COVERAGE_PROFILE).raw.XXXXXX"); \
		data_profile=$$(mktemp "$(COVERAGE_PROFILE).data.XXXXXX"); \
		tmp_profile=$$(mktemp "$(COVERAGE_PROFILE).tmp.XXXXXX"); \
		trap 'rm -f "$$raw_profile" "$$data_profile" "$$tmp_profile"' EXIT; \
		test_packages=$$($(GO_ENV) $(GO) list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...); \
		$(GO_ENV) $(GO) test -covermode=atomic -coverpkg=./... -coverprofile="$$raw_profile" $$test_packages; \
		awk 'NR == 1 { if ($$0 != "mode: atomic") exit 1; next } \
			{ key = $$1 " " $$2; covered[key] = covered[key] || ($$3 > 0) } \
			END { for (key in covered) print key, (covered[key] ? 1 : 0) }' "$$raw_profile" > "$$data_profile"; \
		{ printf '%s\n' 'mode: atomic'; LC_ALL=C sort "$$data_profile"; } > "$$tmp_profile"; \
		mv "$$tmp_profile" "$(COVERAGE_PROFILE)"

coverage-cgo:
	$(MAKE) coverage CGO_ENABLED=1 CC="$(CC)"

bench:
	$(GO_ENV) $(GO) test -run '^$$' -bench . -benchmem ./...

bench-cgo:
	$(MAKE) bench CGO_ENABLED=1 CC="$(CC)"

bench-no-cgo:
	$(MAKE) bench CGO_ENABLED=0 CC="$(NO_CGO_CC)"

bench-all: bench-cgo bench-no-cgo

release-bench-cgo:
	mkdir -p "$(RELEASE_BENCH_RAW_DIR)"
	CGO_ENABLED=1 CC="$(CC)" $(GO) test . -run '^$$' -bench '^BenchmarkEngineValidateByComplexity$$' -benchmem -benchtime="$(RELEASE_BENCH_TIME)" -count="$(RELEASE_BENCH_COUNT)" > "$(RELEASE_BENCH_RAW_DIR)/cgo.txt"

release-bench-no-cgo:
	mkdir -p "$(RELEASE_BENCH_RAW_DIR)"
	CGO_ENABLED=0 CC="$(NO_CGO_CC)" $(GO) test . -run '^$$' -bench '^BenchmarkEngineValidateByComplexity$$' -benchmem -benchtime="$(RELEASE_BENCH_TIME)" -count="$(RELEASE_BENCH_COUNT)" > "$(RELEASE_BENCH_RAW_DIR)/no-cgo.txt"

release-bench-footprint: size-probe
	mkdir -p "$(RELEASE_BENCH_RAW_DIR)"
	stat -f '%N %z bytes' "$(SIZE_PROBE_DIR)/sqlguard-cgo" > "$(RELEASE_BENCH_RAW_DIR)/cgo-footprint.txt"
	for run in 1 2 3 4 5; do /usr/bin/time -l "$(SIZE_PROBE_DIR)/sqlguard-cgo"; done 2>> "$(RELEASE_BENCH_RAW_DIR)/cgo-footprint.txt"
	stat -f '%N %z bytes' "$(SIZE_PROBE_DIR)/sqlguard-no-cgo" > "$(RELEASE_BENCH_RAW_DIR)/no-cgo-footprint.txt"
	for run in 1 2 3 4 5; do /usr/bin/time -l "$(SIZE_PROBE_DIR)/sqlguard-no-cgo"; done 2>> "$(RELEASE_BENCH_RAW_DIR)/no-cgo-footprint.txt"

release-bench-capture: release-bench-cgo release-bench-no-cgo release-bench-footprint

release-bench-charts:
	$(GO_ENV) $(GO) run ./internal/testutil/cmd/benchcharts -data "$(RELEASE_BENCH_DATA)" -out "$(RELEASE_BENCH_CHART_DIR)"

release-bench-check:
	tmp_dir=$$(mktemp -d); trap 'rm -rf "$$tmp_dir"' EXIT; \
		$(GO_ENV) $(GO) run ./internal/testutil/cmd/benchcharts -data "$(RELEASE_BENCH_DATA)" -out "$$tmp_dir"; \
		cmp "$(RELEASE_BENCH_CHART_DIR)/latency.svg" "$$tmp_dir/latency.svg"; \
		cmp "$(RELEASE_BENCH_CHART_DIR)/allocations.svg" "$$tmp_dir/allocations.svg"; \
		cmp "$(RELEASE_BENCH_CHART_DIR)/cold-rss.svg" "$$tmp_dir/cold-rss.svg"; \
		cmp "$(RELEASE_BENCH_CHART_DIR)/binary-size.svg" "$$tmp_dir/binary-size.svg"

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

# The Go race detector requires CGO on Linux; no-CGO concurrency stays covered by the regular test suite.
precommit-no-cgo: mod-tidy lint-no-cgo test-no-cgo

precommit-all: mod-tidy lint-all test-all test-race-cgo
