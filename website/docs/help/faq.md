---
title: FAQ
description: Frequently asked questions about secretctl.
sidebar_position: 1
---

# Frequently Asked Questions

## General

### What is secretctl?

secretctl is a local-first, single-binary secrets manager designed for developers. It provides CLI, desktop app, and AI (MCP) integration while keeping your secrets encrypted and never exposing them to external services.

### Why another secrets manager?

Unlike cloud-based solutions, secretctl:
- Stores secrets locally (no external servers)
- Works offline
- Provides AI-safe secret injection (secrets never exposed to LLMs)
- Ships as a single binary with no dependencies

### Is secretctl free?

Yes, secretctl is open source and free for personal and commercial use.

## Security

### What encryption does secretctl use?

secretctl uses:
- **AES-256-GCM** for authenticated encryption
- **Argon2id** for key derivation (OWASP recommended parameters: 64MB memory, 3 iterations)
- **SQLite** with encrypted storage

### Why can't I get plaintext secrets via MCP?

This is intentional. The MCP server follows the "Option D+" security model where AI agents never receive plaintext secrets. Instead:
- `secret_run` injects secrets as environment variables
- `secret_get_masked` returns masked values (e.g., `****WXYZ`)
- Output is sanitized to prevent accidental secret exposure

This aligns with 1Password's "Access Without Exposure" philosophy.

### Where are my secrets stored?

All data is stored locally in `~/.secretctl/`:
- `vault.db` - Encrypted SQLite database
- `audit/` - Directory with monthly JSONL audit logs (e.g., `2025-01.jsonl`)

### Can I backup my vault?

Yes, you can backup the `~/.secretctl/` directory. The backup will contain encrypted data, so your master password is still required to access secrets.

## CLI Usage

### How do I get started?

```bash
# Initialize a new vault
secretctl init

# Add a secret
secretctl set my-api-key

# Retrieve a secret
secretctl get my-api-key

# Run a command with secrets injected
secretctl run -k "api/*" -- ./my-app
```

### What's the difference between `get` and `run`?

- `get` outputs the secret value directly (for human use)
- `run` injects secrets as environment variables to a subprocess (for automation)

Use `run` when you want to pass secrets to programs without exposing them in command history or logs.

### Can I use wildcards?

Yes, wildcards work with `run`, `export`, and `delete`:

```bash
# Inject all secrets matching aws/*
secretctl run -k "aws/*" -- ./deploy.sh

# Export all database secrets
secretctl export -k "db/*" -f env
```

## Desktop App

### How do I install the desktop app?

Currently, pre-built binaries are not yet available. Build from source:

```bash
cd desktop
wails build
```

See the [Desktop App Guide](/docs/guides/desktop) for details.

### Does the desktop app share secrets with CLI?

Yes, both use the same vault at `~/.secretctl/`. Secrets created in one are immediately available in the other.

### Why does the vault auto-lock?

For security, the vault automatically locks after 15 minutes of inactivity. Any mouse or keyboard activity resets the timer.

## MCP Integration

### How do I use secretctl with Claude Code?

Add the MCP server to your Claude Code configuration:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"]
    }
  }
}
```

### What MCP tools are available?

| Tool | Description |
|------|-------------|
| `secret_list` | List all secret keys with metadata |
| `secret_exists` | Check if a secret exists |
| `secret_get_masked` | Get masked value (e.g., `****WXYZ`) |
| `secret_run` | Run command with secrets as env vars |
| `secret_list_fields` | List field names for multi-field secrets |
| `secret_get_field` | Get non-sensitive field values only |
| `secret_run_with_bindings` | Run with predefined environment bindings |

### Why is there no `secret_get` MCP tool?

By design, secretctl never exposes plaintext secrets to AI agents. Use `secret_run` to inject secrets into subprocesses instead.

## Troubleshooting

### "vault not initialized" error

Run `secretctl init` to create a new vault with a master password.

### "decryption failed" error

This usually means an incorrect master password. The password is set during `secretctl init` and is required for all operations.

### How do I reset my vault?

If you've forgotten your master password, you'll need to delete the vault and start over:

```bash
rm -rf ~/.secretctl
secretctl init
```

**Warning**: This permanently deletes all stored secrets.

### Audit log shows "chain broken" warning

This indicates the audit log may have been tampered with. While secrets remain secure, you should investigate the cause.

## Roadmap

### What features are available?

**Phase 2.5 (Multi-Field Secrets)** - ✅ Shipped:
- Store multiple fields per secret (e.g., username + password + host)
- Pre-defined templates for common secret types (Login, Database, API, SSH)
- Field-level sensitivity control for MCP integration
- Environment variable bindings for `secret_run`

**Using multi-field secrets in CLI:**

```bash
# Create a multi-field secret
secretctl set db/prod --field host=db.example.com --field user=myuser --field password=secret123

# Get a specific field
secretctl get db/prod --field host

# Run with bindings
secretctl run -k db/prod -- ./my-app
```

**Using templates in Desktop App:**
1. Click "Add Secret"
2. Select a template (Login, Database, API, SSH)
3. Fields and bindings are auto-configured
4. Fill in the values and save

See the [project roadmap](https://github.com/forest6511/secretctl) for the latest development status.
