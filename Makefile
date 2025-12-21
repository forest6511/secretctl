# secretctl Makefile
# Common development tasks

.PHONY: all build test lint fmt vet clean install-tools pre-commit coverage help

# Default target
all: fmt lint test build

# Build the CLI binary
build:
	@echo "Building secretctl..."
	@go build -o bin/secretctl ./cmd/secretctl

# Build desktop app (requires Wails)
build-desktop:
	@echo "Building desktop app..."
	@cd desktop && wails build

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
	@rm -rf desktop/build/

# Quick check before commit (fmt + lint + test-short)
check: fmt lint test-short
	@echo "All checks passed!"

# Full CI check (matching GitHub Actions)
ci: fmt vet lint test security
	@echo "CI checks passed!"

# Help
help:
	@echo "Available targets:"
	@echo "  make build       - Build CLI binary"
	@echo "  make build-desktop - Build desktop app"
	@echo "  make test        - Run all tests"
	@echo "  make coverage    - Run tests with coverage report"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make fmt         - Format code with gofmt and goimports"
	@echo "  make vet         - Run go vet"
	@echo "  make tidy        - Tidy go modules"
	@echo "  make security    - Run security scans"
	@echo "  make install-tools - Install development tools"
	@echo "  make pre-commit  - Setup pre-commit hooks"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make check       - Quick pre-commit check"
	@echo "  make ci          - Full CI check"
