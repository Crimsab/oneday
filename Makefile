APP=oneday
BENCH=oneday-benchmark
ASCII_BENCH=oneday-ascii-benchmark
BUILD_DIR=build

.PHONY: test vet build build-bench build-ascii-bench build-cross all

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o ./$(APP) ./cmd/oneday

build-bench:
	go build -o ./$(BENCH) ./cmd/oneday-benchmark

build-ascii-bench:
	go build -o ./$(ASCII_BENCH) ./cmd/oneday-ascii-benchmark

build-cross:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/oneday-linux-amd64 ./cmd/oneday
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/oneday-windows-amd64.exe ./cmd/oneday

all: test vet build build-bench build-ascii-bench build-cross
