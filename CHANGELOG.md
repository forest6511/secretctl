# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2025-12-05

### Added
- **`mcp-server` command** - MCP server for AI coding assistant integration
  - `secret_list`: List secret keys with metadata (no values)
  - `secret_exists`: Check if a secret exists with metadata
  - `secret_get_masked`: Get masked secret value (e.g., `****WXYZ`)
  - `secret_run`: Execute command with secrets as environment variables
  - Policy-based command allowlisting via `~/.secretctl/mcp-policy.yaml`

### Security
- **Option D+ design**: AI agents never receive plaintext secrets
- Default denied commands (env, printenv, set, export) always blocked
- Output sanitization in secret_run prevents secret leakage
- TOCTOU-safe policy file loading with symlink rejection
- Concurrent execution limiting (max 5 secret_run operations)

## [0.2.0] - 2025-12-03

### Added
- **`run` command** - Execute commands with secrets injected as environment variables
  - Wildcard key patterns (`-k "aws/*"`)
  - Output sanitization to prevent secret leakage in stdout/stderr
  - Configurable timeout (`--timeout`)
  - Environment variable prefix support (`--env-prefix`)
- **`export` command** - Export secrets to file or stdout
  - `.env` format (default)
  - JSON format (`--format=json`)
  - Key filtering with glob patterns (`-k "db/*"`)
- **`generate` command** - Generate secure random passwords
  - Configurable length (`-l`, default 24)
  - Character set options (`--no-symbols`, `--no-numbers`, etc.)
  - Multiple password generation (`-n`)
- **`audit export` command** - Export audit logs
  - JSON and CSV formats
  - Time range filtering (`--since`, `--until`)
- **`audit prune` command** - Delete old audit logs
  - Dry-run mode (`--dry-run`)
  - Configurable retention (`--older-than`)

### Security
- Output sanitization detects and redacts secrets in command output
- Path traversal protection for export file paths
- CSV injection prevention in audit export

## [0.1.0] - 2025-12-01

### Added
- Core vault implementation with AES-256-GCM encryption
- Argon2id key derivation for master password
- SQLite-based secret storage
- HMAC-chained audit logging with tamper detection
- Metadata support (notes, tags, URL, expiration)
- Master password strength validation
- File permission validation (0600/0700)
- CLI commands: init, lock, unlock, set, get, list, delete
- Audit commands: list, verify
