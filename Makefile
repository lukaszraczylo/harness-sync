.PHONY: build test lint test-race cover tidy clean install

BIN := harness-sync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/harness-sync

test:
	go test ./...

lint:
	golangci-lint run ./...

test-race:
	go test -race -count=1 ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

tidy:
	go mod tidy
	gofmt -s -w .

install: build
	install -m 0755 $(BIN) $$HOME/.local/bin/$(BIN)

clean:
	rm -f $(BIN)
