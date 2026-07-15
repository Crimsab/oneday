APP=oneday
BENCH=oneday-benchmark
ASCII_BENCH=oneday-ascii-benchmark
BUILD_DIR=build
LDFLAGS=$(shell bash ./scripts/build-ldflags.sh)
DOCS_VENV?=.venv-docs

.PHONY: test coverage vet verify docs-install docs-prepare docs-build docs-serve docs-check qa-matrix qa-matrix-auto universal-release-check release-check friend-safe-check build build-bench build-ascii-bench build-cross all

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	cd gateway/web && bun run test:coverage

vet:
	go vet ./...

verify: docs-check test vet qa-matrix-auto

docs-check:
	bun scripts/check-docs.ts

docs-install:
	python3 -m venv $(DOCS_VENV)
	$(DOCS_VENV)/bin/python -m pip install --disable-pip-version-check -r requirements-docs.txt

docs-prepare:
	bun scripts/prepare-docs-site.ts

docs-build: docs-prepare
	$(DOCS_VENV)/bin/mkdocs build --strict

docs-serve: docs-prepare
	$(DOCS_VENV)/bin/mkdocs serve --dev-addr 127.0.0.1:8000

qa-matrix:
	./scripts/qa-matrix.sh

qa-matrix-auto:
	./scripts/qa-matrix.sh --automated-only

release-check:
	./scripts/release-gate.sh

universal-release-check:
	./scripts/universal-release-gate.sh

friend-safe-check:
	./scripts/friend-safe-check.sh

build:
	go build -ldflags "$(LDFLAGS)" -o ./$(APP) ./cmd/oneday

build-bench:
	go build -ldflags "$(LDFLAGS)" -o ./$(BENCH) ./cmd/oneday-benchmark

build-ascii-bench:
	go build -ldflags "$(LDFLAGS)" -o ./$(ASCII_BENCH) ./cmd/oneday-ascii-benchmark

build-cross:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/oneday-linux-amd64 ./cmd/oneday
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/oneday-windows-amd64.exe ./cmd/oneday

all: verify build build-bench build-ascii-bench build-cross
