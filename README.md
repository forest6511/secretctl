# secretctl

**The simplest AI-ready secrets manager.**

No infrastructure. No subscription. No complexity.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

---

## Why secretctl?

Managing secrets shouldn't require a PhD in DevOps. secretctl is a local-first secrets manager that:

- **Just works** — Single binary, no servers, no configuration files
- **Stays local** — Your secrets never leave your machine
- **Plays nice with AI** — Designed for the age of AI coding assistants (MCP-ready)
- **Respects your workflow** — CLI-first with optional Web UI

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

### Audit Log

```bash
# View recent audit events
secretctl audit list --limit=50

# Verify log integrity
secretctl audit verify
```

## Security

secretctl takes security seriously:

- **Zero-knowledge design** — Your master password is never stored or transmitted
- **AES-256-GCM encryption** — Industry-standard authenticated encryption
- **Argon2id key derivation** — Memory-hard protection against brute force
- **Secure file permissions** — Vault files are created with 0600 permissions
- **No network access** — Completely offline operation
- **Tamper-evident logs** — HMAC chain detects any log manipulation

For reporting security vulnerabilities, please see [SECURITY.md](SECURITY.md).

## Documentation

- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)

## License

Apache License 2.0 — See [LICENSE](LICENSE) for details.

---

Built with care for developers who value simplicity and security.
