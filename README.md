# secretctl

**The simplest AI-ready secrets manager.**

No infrastructure. No subscription. No complexity.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-website-blue)](https://forest6511.github.io/secretctl/)
[![Codecov](https://codecov.io/gh/forest6511/secretctl/branch/main/graph/badge.svg)](https://codecov.io/gh/forest6511/secretctl)

---

## Why secretctl?

Managing secrets shouldn't require a PhD in DevOps. secretctl is a local-first secrets manager that:

- **Just works** — Single binary, no servers, no configuration files
- **Stays local** — Your secrets never leave your machine
- **Plays nice with AI** — Designed for the age of AI coding assistants (MCP-ready)
- **Respects your workflow** — CLI-first with Desktop App

```
# That's it. You're done.
secretctl init
secretctl set API_KEY
secretctl get API_KEY
```

## Installation

### From Source

```bash
# Requires Go 1.24+
git clone https://github.com/forest6511/secretctl.git
cd secretctl
go build -o secretctl ./cmd/secretctl
```

### Binary Releases

Coming soon.

## Quick Start

### 1. Initialize your vault

```bash
secretctl init
# Enter your master password (min 8 characters)
```

### 2. Store a secret

```bash
echo "sk-your-api-key" | secretctl set OPENAI_API_KEY
```

### 3. Retrieve a secret

```bash
secretctl get OPENAI_API_KEY
```

### 4. List all secrets

```bash
secretctl list
```

### 5. Delete a secret

```bash
secretctl delete OPENAI_API_KEY
```

## Features

### Core

- **AES-256-GCM encryption** — Industry-standard authenticated encryption
- **Argon2id key derivation** — Memory-hard protection against brute force
- **SQLite storage** — Reliable, portable, no external dependencies
- **Audit logging** — HMAC-chained logs for tamper detection
- **AI-safe by design** — MCP integration never exposes plaintext secrets to AI agents

### Metadata Support

```bash
# Add notes and tags to secrets
secretctl set DB_PASSWORD --notes="Production database" --tags="prod,db"

# Add URL reference
secretctl set API_KEY --url="https://console.example.com/api-keys"

# Set expiration
secretctl set TEMP_TOKEN --expires="30d"

# Filter by tag
secretctl list --tag=prod

# Show expiring secrets
secretctl list --expiring=7d

# View full metadata
secretctl get API_KEY --show-metadata
```

### Run Commands with Secrets

Inject secrets as environment variables without exposing them in your shell history:

```bash
# Run a command with a single secret
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" https://api.example.com

# Use wildcards to inject multiple secrets
# Pattern aws/* matches aws/access_key, aws/secret_key (single level)
secretctl run -k "aws/*" -- aws s3 ls

# Output is automatically sanitized to prevent secret leakage
secretctl run -k DB_PASSWORD -- ./deploy.sh
# If deploy.sh prints DB_PASSWORD, it appears as [REDACTED:DB_PASSWORD]

# With timeout and prefix
secretctl run -k API_KEY --timeout=30s --env-prefix=APP_ -- ./app
```

> **Note**: Output sanitization uses exact string matching. Encoded secrets (Base64, hex) or partial matches are not detected.

### Export Secrets

Export secrets for use with Docker, CI/CD, or other tools:

```bash
# Export as .env file (default)
secretctl export -o .env

# Export specific keys as JSON
secretctl export --format=json -k "db/*" -o config.json

# Export to stdout for piping
secretctl export --format=json | jq '.DB_HOST'
```

### Generate Passwords

Create secure random passwords:

```bash
# Generate a 24-character password (default)
secretctl generate

# Generate a 32-character password without symbols
secretctl generate -l 32 --no-symbols

# Generate multiple passwords
secretctl generate -n 5
```

### Audit Log

```bash
# View recent audit events
secretctl audit list --limit=50

# Verify log integrity
secretctl audit verify

# Export audit logs
secretctl audit export --format=csv -o audit.csv

# Prune old logs (preview first)
secretctl audit prune --older-than=12m --dry-run
```

### AI Integration (MCP Server)

secretctl includes an MCP server for secure integration with AI coding assistants like Claude Code:

```bash
# Start MCP server (requires SECRETCTL_PASSWORD)
SECRETCTL_PASSWORD=your-password secretctl mcp-server
```

**Available MCP Tools:**
- `secret_list` — List secret keys with metadata (no values exposed)
- `secret_exists` — Check if a secret exists with metadata
- `secret_get_masked` — Get masked value (e.g., `****WXYZ`)
- `secret_run` — Execute commands with secrets as environment variables

**Configure in Claude Code** (`~/.claude.json`):
```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

**Policy Configuration** (`~/.secretctl/mcp-policy.yaml`):
```yaml
version: 1
default_action: deny
allowed_commands:
  - aws
  - gcloud
  - kubectl
```

> **Security**: AI agents never receive plaintext secrets. The `secret_run` tool injects secrets as environment variables, and output is automatically sanitized.

### Desktop App

secretctl includes a native desktop application built with Wails v2:

```bash
# Build the desktop app
cd desktop && wails build

# Or run in development mode
cd desktop && wails dev
```

**Features:**
- Native macOS/Windows/Linux application
- Create and unlock vaults with master password
- Full secret CRUD operations (Create, Read, Update, Delete)
- Search and filter secrets by key
- Copy secret values to clipboard (with auto-clear)
- Metadata support (URL, tags, notes)
- Password visibility toggle
- Auto-lock on idle timeout
- **Audit Log Viewer** — View and analyze all vault activity
  - Filter by action, source, key, and date range
  - Pagination for large log volumes
  - Chain integrity verification
  - Export to CSV/JSON formats
  - Detailed log entry modal
- Modern React + TypeScript + Tailwind CSS frontend

**Development:**
```bash
# Run E2E tests (Playwright)
cd desktop/frontend
npm run test:e2e

# Run with visible browser
npm run test:e2e:headed

# Run with Playwright UI
npm run test:e2e:ui
```

## Security

secretctl takes security seriously:

- **Zero-knowledge design** — Your master password is never stored or transmitted
- **AES-256-GCM encryption** — Industry-standard authenticated encryption
- **Argon2id key derivation** — Memory-hard protection against brute force
- **Secure file permissions** — Vault files are created with 0600 permissions
- **No network access** — Completely offline operation
- **Tamper-evident logs** — HMAC chain detects any log manipulation
- **Output sanitization** — Automatic redaction of secrets in command output

For reporting security vulnerabilities, please see [SECURITY.md](SECURITY.md).

## Documentation

📚 **[Full Documentation](https://forest6511.github.io/secretctl/)** — Getting started, guides, and reference

- [Getting Started](https://forest6511.github.io/secretctl/docs/getting-started/) - Installation and quick start
- [CLI Guide](https://forest6511.github.io/secretctl/docs/guides/cli/) - Command-line usage
- [MCP Integration](https://forest6511.github.io/secretctl/docs/guides/mcp/) - AI agent integration
- [Desktop App](https://forest6511.github.io/secretctl/docs/guides/desktop/) - Native application guide
- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)

## License

Apache License 2.0 — See [LICENSE](LICENSE) for details.

---

Built with care for developers who value simplicity and security.
