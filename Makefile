.PHONY: build install clean test run release help

BINARY := osem
INSTALL_PATH := ~/.local/bin/$(BINARY)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'none')"

help:
	@echo "osem - OpenCode Session Manager"
	@echo ""
	@echo "Usage:"
	@echo "  make build      Build binary"
	@echo "  make install    Install to ~/.local/bin"
	@echo "  make clean      Remove binary"
	@echo "  make test       Run tests"
	@echo "  make run        Run directly"
	@echo "  make release    Build for multiple platforms"
	@echo ""

build:
	go build $(BUILD_FLAGS) -o $(BINARY) ./cmd/osem

install: build
	cp $(BINARY) $(INSTALL_PATH)
	chmod +x $(INSTALL_PATH)
	@echo "Installed to $(INSTALL_PATH)"

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go test -v ./...

run:
	go run ./cmd/osem

release: clean
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/osem-linux-amd64 ./cmd/osem
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/osem-linux-arm64 ./cmd/osem
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/osem-darwin-amd64 ./cmd/osem
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/osem-darwin-arm64 ./cmd/osem
	@echo "Release binaries in dist/"

version:
	@echo "osem $(VERSION)"