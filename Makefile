# Makefile — waba-cli. `make verify` is the deterministic gate (CI/dev);
# `make accept` (= verify + the LLM judge) is the build-acceptance gate.
BINARY      := waba
MODULE      := github.com/jjuanrivvera/waba-cli
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
  -X $(MODULE)/internal/version.Version=$(VERSION) \
  -X $(MODULE)/internal/version.Commit=$(COMMIT) \
  -X $(MODULE)/internal/version.Date=$(DATE)
COVERAGE_MIN ?= 80
# spec-completeness: min % of the enumerated full API the manifest must wrap
API_COVERAGE_MIN ?= 90

.DEFAULT_GOAL := build

## --- build & run ---
build: ## build to bin/$(BINARY)
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/$(BINARY)
install: ## go install the binary
	CGO_ENABLED=0 go install -ldflags '$(LDFLAGS)' ./cmd/$(BINARY)
uninstall: ; rm -f "$$(go env GOPATH)/bin/$(BINARY)"
run: build ; ./bin/$(BINARY) $(ARGS)
dev: fmt vet build ## fast local cycle

## --- quality gate ---
fmt: ; gofmt -s -w .
vet: ; go vet ./...
lint: ; golangci-lint run ./... || (echo "golangci-lint missing or failed" >&2; exit 1)
tidy: ; go mod tidy
test: ; go test ./...
test-race: ; go test -race ./...
test-coverage: ; go test ./... -coverprofile=coverage.out -coverpkg=./...
cover-check: test-coverage ; ./scripts/cover-check.sh $(COVERAGE_MIN)
check: fmt vet lint test ## the local quality gate

## --- the acceptance gate (cliwright) ---
# verify == the DETERMINISTIC gate (build/test/lint/spec/coverage/DoD). Fast, repeatable,
# CI-safe, zero LLM/tokens — this is what CI and routine `make` runs should use.
verify: check spec-check spec-completeness cover-check ## deterministic gate; exit 0 == green
	./scripts/dod-check.sh $(BINARY)
# judge == the ONE non-deterministic gate (an LLM scores the few subjective DoD items).
judge: ## LLM-scored subjective gate (build-acceptance only; needs claude/codex)
	./scripts/judge.sh
# accept == the full build-acceptance gate. The /goal completion promise binds to THIS.
accept: verify judge ## full acceptance (verify + judge); exit 0 == done & high
spec-check: build ## built CLI surface ⊆ the spec-derived manifest (consistency)
	./scripts/spec-check.sh
spec-completeness: ## manifest must wrap ≥$(API_COVERAGE_MIN)% of the enumerated full API (completeness)
	./scripts/spec-completeness.sh api-manifest.json $(API_COVERAGE_MIN)

## --- docs & release ---
docs-gen: ; go run ./tools/gendocs
docs-serve: ; mkdocs serve
docs-build: ; mkdocs build
snapshot: ; goreleaser release --snapshot --clean --skip=sign,sbom,docker
setup-hooks: ; git config core.hooksPath .githooks && echo "hooks installed"
clean: ; rm -rf bin dist coverage.out

.PHONY: build install uninstall run dev fmt vet lint tidy test test-race \
        test-coverage cover-check check verify judge accept spec-check spec-completeness \
        docs-gen docs-serve docs-build snapshot setup-hooks clean
