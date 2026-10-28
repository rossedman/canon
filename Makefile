SHELL := /usr/bin/env bash
# canon: run `make check` before pushing; it is what CI runs.

GO       ?= go
PKGS     := ./...
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X github.com/rossedman/canon/cmd.version=$(VERSION) -X github.com/rossedman/canon/cmd.commit=$(COMMIT) -X github.com/rossedman/canon/cmd.date=$(DATE)
FUZZTIME ?= 30s
MKDOCS   := uv run --no-project --with-requirements requirements-docs.txt mkdocs

.PHONY: build test cover cover-gate bench bench-smoke fuzz fuzz-smoke lint fmt vuln docs docs-build docs-serve check-cmds doccheck canon-lint release-check check clean

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/canon .

test:
	$(GO) test -race $(PKGS)

cover:
	$(GO) test -race -coverprofile=coverage.out $(PKGS)
	{ head -1 coverage.out; grep -v '^mode:' coverage.out | grep -vFf <($(GO) run ./tools/generated ./...) | grep -v '/examples/'; } > coverage.hand.out
	$(GO) tool cover -func=coverage.hand.out | tail -1

cover-gate: cover
	$(GO) tool cover -func=coverage.hand.out | awk '/^total:/ {sub("%","",$$3); if ($$3+0 < 90) {print "coverage " $$3 "% < 90%"; exit 1} else {print "coverage " $$3 "%"}}'

bench:
	mkdir -p bench
	$(GO) test -run='^$$' -bench=. -benchmem -count=6 $(PKGS) | tee bench/$(shell date +%Y%m%d-%H%M%S).txt
	@echo "compare runs with: benchstat bench/<old>.txt bench/<new>.txt"

bench-smoke:
	$(GO) test -run='^$$' -bench=. -benchtime=1x $(PKGS)

fuzz:
	./tools/fuzz.sh $(FUZZTIME)

fuzz-smoke:
	./tools/fuzz.sh 10s

lint:
	golangci-lint run
	shellcheck tools/*.sh

fmt:
	gofmt -w $$(git ls-files '*.go')

vuln:
	govulncheck ./...

check-cmds:
	./tools/check-cmds.sh

doccheck:
	$(GO) run ./tools/doccheck ./...

canon-lint:
	$(GO) run . lint

docs:
	$(GO) run ./tools/gendocs

docs-build: docs
	$(MKDOCS) build --strict

docs-serve: docs
	$(MKDOCS) serve

release-check:
	goreleaser check
	goreleaser release --snapshot --clean --skip=sign

check: build lint cover-gate check-cmds doccheck canon-lint bench-smoke fuzz-smoke docs-build release-check

clean:
	rm -rf bin dist site coverage.out coverage.hand.out
