BIN_NAME := goomerang
BIN := bin/$(BIN_NAME)
DIST := dist
VERSION ?= dev
DOCKER_BUILD_PLATFORMS ?= linux/amd64,linux/arm64
DOCKER ?= $(shell which podman 2>/dev/null || echo docker)

.PHONY: all build run test test-cover lint fmt tidy clean \
        binary-linux-amd64 binary-linux-arm64 docker-build docker-run

all: fmt lint test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/goomerang

run: build
	./$(BIN)

binary-linux-amd64:
	mkdir -p $(DIST)/linux/amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	  go build -ldflags "-s -w -X main.version=$(VERSION)" \
	  -o $(DIST)/linux/amd64/$(BIN_NAME) ./cmd/goomerang

binary-linux-arm64:
	mkdir -p $(DIST)/linux/arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	  go build -ldflags "-s -w -X main.version=$(VERSION)" \
	  -o $(DIST)/linux/arm64/$(BIN_NAME) ./cmd/goomerang

docker-build: binary-linux-amd64 binary-linux-arm64
	$(DOCKER) build --build-arg TARGETPLATFORM=linux/$$(go env GOARCH) -t $(BIN_NAME) .

docker-run:
	$(DOCKER) run -v ./goomerang.yml:/goomerang.yml -p 8080:8080 $(BIN_NAME)

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
	rm -rf bin $(DIST) coverage.out coverage.html
