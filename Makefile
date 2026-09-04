GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BIN_NAME := goomerang
BIN := bin/$(GOOS)/$(GOARCH)/$(BIN_NAME)

.PHONY: all build run test test-cover lint fmt tidy clean

all: fmt lint test build

build:
	mkdir -p bin/$(GOOS)/$(GOARCH)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN) ./cmd/goomerang

run: build
	./$(BIN)

test:
	go test -v ./...

test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	echo "Coverage report: coverage.html"

lint:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

clean:
	rm -rf bin coverage.out coverage.html
