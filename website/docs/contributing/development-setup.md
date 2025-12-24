---
title: Development Setup
description: Set up your development environment for contributing to secretctl.
sidebar_position: 2
---

# Development Setup

This guide helps you set up a development environment for contributing to secretctl.

## Prerequisites

### Required

- **Go 1.24+** - [Download](https://go.dev/dl/)
- **Git** - For version control
- **Make** (optional) - For build automation

### For Desktop App Development

- **Node.js 18+** - For frontend development
- **Wails v2** - [Installation](https://wails.io/docs/gettingstarted/installation)

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/forest6511/secretctl.git
cd secretctl
```

### 2. Install Dependencies

```bash
# Go dependencies
go mod download

# Verify the build
go build ./...
```

### 3. Run Tests

```bash
# All tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 4. Run Linting

```bash
# Install golangci-lint if needed
# https://golangci-lint.run/usage/install/

golangci-lint run ./...
```

## Project Structure

```
secretctl/
├── cmd/                    # CLI commands (Cobra)
│   └── secretctl/
├── internal/               # Internal packages
│   ├── cli/               # CLI utilities
│   ├── config/            # Configuration handling
│   └── mcp/               # MCP server implementation
├── pkg/                    # Public packages
│   ├── audit/             # Audit logging
│   ├── backup/            # Backup and restore
│   ├── crypto/            # Cryptographic operations
│   ├── secret/            # Secret types
│   └── vault/             # Vault operations
├── desktop/               # Wails desktop app
│   ├── frontend/          # React + TypeScript
│   └── *.go               # Go backend
└── website/               # Documentation site
```

## Building

### CLI Binary

```bash
# Development build
go build -o bin/secretctl ./cmd/secretctl

# Run locally
./bin/secretctl --help
```

### Desktop App

```bash
cd desktop

# Development mode (hot reload)
wails dev

# Production build
wails build
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

- Write clear, readable code
- Follow existing patterns
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/vault/...

# Run with verbose output
go test -v ./...
```

### 4. Lint and Format

```bash
# Format code
gofmt -w .
goimports -w .

# Run linter
golangci-lint run ./...
```

### 5. Commit and Push

Use [Conventional Commits](https://www.conventionalcommits.org/):

```bash
git add .
git commit -m "feat: add password strength indicator"
git push origin feature/your-feature-name
```

## Testing Tips

### Table-Driven Tests

secretctl uses table-driven tests extensively:

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid", "SecurePass123!", false},
        {"too short", "short", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Integration Tests

For tests that need a real vault:

```go
func TestVaultOperations(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir()

    // Initialize vault
    v := vault.New(tmpDir)
    err := v.Create("test-password-123!")
    require.NoError(t, err)

    // Your test logic...
}
```

## Security Considerations

When contributing security-related code:

- **Never log secrets** - No passwords, keys, or sensitive data in logs
- **Use crypto/rand** - Never use `math/rand` for security purposes
- **Wipe sensitive data** - Use `crypto.SecureWipe()` for passwords and keys
- **Handle errors** - Never ignore errors from crypto operations

## Getting Help

- Read existing code for patterns and conventions
- Check [issues](https://github.com/forest6511/secretctl/issues) for context
- Open a [discussion](https://github.com/forest6511/secretctl/discussions) for questions

## Next Steps

- [Contributing Guidelines](/docs/contributing) - Full contribution guide
- [Architecture Overview](/docs/architecture) - System design
- [Security Design](/docs/security/encryption) - Cryptographic details
