BINARY := ./dist/ghostty-config
PKG    := ./cmd/ghostty-config
GOFLAGS ?=

.PHONY: build dev install fmt vet tidy clean help release-check release-snapshot release-dry-run

help:
	@echo "Targets:"
	@echo "  build              Compile $(BINARY) into ./$(BINARY)"
	@echo "  dev                Build and run with default flags"
	@echo "  install            go install the binary into GOBIN"
	@echo "  fmt                go fmt ./..."
	@echo "  vet                go vet ./..."
	@echo "  tidy               go mod tidy"
	@echo "  clean              Remove the built binary"
	@echo "  release-check      Validate .goreleaser.yaml"
	@echo "  release-snapshot   Build all platforms locally (no publish, no git tag needed)"
	@echo "  release-dry-run    Full release pipeline without publishing"

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
	rm -rf ./dist

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish

release-dry-run:
	goreleaser release --skip=publish --clean
