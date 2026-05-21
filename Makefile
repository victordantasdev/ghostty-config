BINARY     := ./dist/ghostty-config
PKG        := ./cmd/ghostty-config
COVER_OUT  := ./dist/coverage.out
COVER_HTML := ./dist/coverage.html
GOFLAGS    ?=

.PHONY: build dev install fmt vet tidy clean help \
	test test-race test-verbose cover cover-html cover-check \
	release-check release-snapshot release-dry-run

help:
	@echo "Targets:"
	@echo "  build              Compile $(BINARY) into ./$(BINARY)"
	@echo "  dev                Build and run with default flags"
	@echo "  install            go install the binary into GOBIN"
	@echo "  fmt                go fmt ./..."
	@echo "  vet                go vet ./..."
	@echo "  tidy               go mod tidy"
	@echo "  test               Run the full test suite"
	@echo "  test-race          Run tests with the race detector"
	@echo "  test-verbose       Run tests with verbose output"
	@echo "  cover              Run tests and print per-function coverage"
	@echo "  cover-html         Run tests and open an HTML coverage report"
	@echo "  cover-check        Fail if total coverage is below 100%"
	@echo "  clean              Remove the built binary and coverage artifacts"
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

test:
	go test ./...

test-race:
	go test -race ./...

test-verbose:
	go test -v ./...

cover:
	@mkdir -p $(dir $(COVER_OUT))
	go test ./... -coverprofile=$(COVER_OUT) -covermode=count
	go tool cover -func=$(COVER_OUT)

cover-html: cover
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "HTML report: $(COVER_HTML)"
	@command -v open >/dev/null 2>&1 && open $(COVER_HTML) || true

cover-check: cover
	@total=$$(go tool cover -func=$(COVER_OUT) | awk '/^total:/ {print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$total%"; \
	awk -v t=$$total 'BEGIN { if (t+0 < 100) exit 1 }' || \
		(echo "coverage below 100%"; exit 1)

clean:
	rm -rf ./dist

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean --skip=publish

release-dry-run:
	goreleaser release --skip=publish --clean
