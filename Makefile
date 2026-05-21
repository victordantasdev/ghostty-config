BINARY := ./dist/ghostty-config
PKG    := ./cmd/ghostty-config
GOFLAGS ?=

.PHONY: build dev install fmt vet tidy clean help

help:
	@echo "Targets:"
	@echo "  build    Compile $(BINARY) into ./$(BINARY)"
	@echo "  dev      Build and run with default flags"
	@echo "  install  go install the binary into GOBIN"
	@echo "  fmt      go fmt ./..."
	@echo "  vet      go vet ./..."
	@echo "  tidy     go mod tidy"
	@echo "  clean    Remove the built binary"

build:
	go build $(GOFLAGS) -o $(BINARY) $(PKG)

dev: build
	./$(BINARY)

install:
	go install $(GOFLAGS) $(PKG)

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
