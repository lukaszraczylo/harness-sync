.PHONY: build test lint clean install

BIN := harness-sync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/harness-sync

test:
	go test ./...

lint:
	go vet ./...

install: build
	install -m 0755 $(BIN) $$HOME/.local/bin/$(BIN)

clean:
	rm -f $(BIN)
