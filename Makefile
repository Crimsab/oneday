APP=oneday
BENCH=oneday-benchmark
ASCII_BENCH=oneday-ascii-benchmark
BUILD_DIR=build
LDFLAGS=$(shell bash ./scripts/build-ldflags.sh)

.PHONY: test vet verify qa-matrix qa-matrix-auto release-check friend-safe-check build build-bench build-ascii-bench build-cross all

test:
	go test ./...

vet:
	go vet ./...

verify: test vet qa-matrix-auto

qa-matrix:
	./scripts/qa-matrix.sh

qa-matrix-auto:
	./scripts/qa-matrix.sh --automated-only

release-check:
	./scripts/release-gate.sh

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
