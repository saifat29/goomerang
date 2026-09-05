BIN_NAME := goomerang
BIN := bin/$(BIN_NAME)

.PHONY: all build run test test-cover lint fmt tidy clean

all: fmt lint test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-X main.version=dev" -o $(BIN) ./cmd/goomerang

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
	goreleaser check

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

clean:
	rm -rf bin dist coverage.out coverage.html
