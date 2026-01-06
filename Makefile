# secretctl Makefile
# Common development tasks

.PHONY: all build test lint fmt vet clean install-tools pre-commit coverage help \
        build-ui test-ui test-e2e typecheck run dev-desktop snapshot

# Version (can be overridden: make build VERSION=1.0.0)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
all: fmt lint test build

# Build the CLI binary with version info
build:
	@echo "Building secretctl ($(VERSION))..."
	@go build -ldflags "-X main.version=$(VERSION)" -o bin/secretctl ./cmd/secretctl

# Build desktop app (requires Wails)
build-desktop:
	@echo "Building desktop app..."
	@cd desktop && wails build

# Build frontend UI only
build-ui:
	@echo "Building frontend UI..."
	@cd desktop/frontend && npm run build

# Run CLI directly (for development)
run:
	@go run ./cmd/secretctl $(ARGS)

# Run desktop app in development mode
dev-desktop:
	@echo "Starting desktop app in dev mode..."
	@cd desktop && wails dev

# Run all tests
test:
	@echo "Running tests..."
	@go test -race ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1
	@echo "Coverage report: go tool cover -html=coverage.out"

# Run short tests (for pre-commit)
test-short:
	@go test -short ./...

# Run frontend unit tests (Vitest)
test-ui:
	@echo "Running frontend unit tests..."
	@cd desktop/frontend && npm run test:unit

# Run E2E tests (Playwright)
test-e2e:
	@echo "Running E2E tests..."
	@cd desktop/frontend && npm run test:e2e

# Run TypeScript type check
typecheck:
	@echo "Running TypeScript type check..."
	@cd desktop/frontend && npm run typecheck

# Build snapshot release (for local testing)
snapshot:
	@echo "Building snapshot release..."
	@goreleaser build --snapshot --clean

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@goimports -local github.com/forest6511/secretctl -w .

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Tidy go modules
tidy:
	@echo "Tidying go modules..."
	@go mod tidy

# Run security scan
security:
	@echo "Running security scan..."
	@gosec -severity high -confidence high -exclude-dir=desktop ./...
	@govulncheck ./...

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@pip install pre-commit 2>/dev/null || echo "pip not available, skipping pre-commit"
	@echo "Done. Run 'pre-commit install' to enable git hooks."

# Setup pre-commit hooks
pre-commit:
	@pre-commit install
	@echo "Pre-commit hooks installed."

# Run pre-commit on all files
pre-commit-all:
	@pre-commit run --all-files

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out
	@rm -rf desktop/build/bin/

# Quick check before commit (fmt + lint + test-short)
check: fmt lint test-short
	@echo "All checks passed!"

# Full CI check (matching GitHub Actions)
ci: fmt vet lint test security
	@echo "CI checks passed!"

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build CLI binary with version info"
	@echo "  make build-desktop - Build desktop app (Wails)"
	@echo "  make build-ui      - Build frontend UI only"
	@echo "  make snapshot      - Build snapshot release (GoReleaser)"
	@echo ""
	@echo "Development:"
	@echo "  make run ARGS='...' - Run CLI directly"
	@echo "  make dev-desktop   - Run desktop app in dev mode"
	@echo ""
	@echo "Testing:"
	@echo "  make test          - Run Go tests"
	@echo "  make test-short    - Run short Go tests"
	@echo "  make test-ui       - Run frontend unit tests (Vitest)"
	@echo "  make test-e2e      - Run E2E tests (Playwright)"
	@echo "  make coverage      - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt           - Format code (gofmt + goimports)"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make vet           - Run go vet"
	@echo "  make typecheck     - Run TypeScript type check"
	@echo "  make security      - Run security scans"
	@echo ""
	@echo "Utilities:"
	@echo "  make tidy          - Tidy go modules"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install development tools"
	@echo "  make pre-commit    - Setup pre-commit hooks"
	@echo ""
	@echo "Shortcuts:"
	@echo "  make check         - Quick pre-commit check"
	@echo "  make ci            - Full CI check"
