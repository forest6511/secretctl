# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`backup` command** - Create encrypted vault backups (#83)
  - AES-256-GCM encryption with fresh salt per backup
  - HMAC-SHA256 integrity verification
  - Optional audit log inclusion (`--with-audit`)
  - Key file support for automation (`--key-file`)
  - Stdout output for piping (`--stdout`)
- **`restore` command** - Restore vault from encrypted backup (#83)
  - Integrity verification (`--verify-only`)
  - Dry run mode (`--dry-run`)
  - Conflict handling: skip, overwrite, error
  - Atomic restore prevents partial states
- **`import` command** - Import secrets from .env or JSON files (#91)
  - Support for `.env` and JSON formats
  - Conflict handling modes: `--skip`, `--overwrite`, `--error`
  - Preview mode with `--dry-run`
  - Import summary with counts

### Fixed
- **E2E test stability** - Resolved flaky tests AUDIT-002 and CORE-004 (#93)
  - Fixed strict mode violations in audit table header checks
  - Fixed custom ConfirmDialog interaction in delete tests
  - Improved CI workflow timeout handling

## [0.5.0] - 2025-12-08

### Added
- **Audit Log Viewer** - Comprehensive audit log viewing in desktop application
  - Filter logs by action, source, key, and date range
  - Pagination support for large log volumes (20 entries per page)
  - Real-time chain integrity verification with visual status
  - Export logs to CSV and JSON formats
  - Detailed log entry modal with full metadata
  - Statistics display (total, success, failure counts)
- Backend API enhancements for audit log search
  - `SearchAuditLogs` with filtering and pagination
  - `VerifyAuditLogs` for chain integrity verification
  - `GetAuditLogStats` for summary statistics

## [0.4.1] - 2025-12-08

### Added
- **Desktop App Secret CRUD** - Full secret management in desktop application
  - Create secrets with key, value, URL, tags, and notes
  - Read and view secret details with metadata
  - Update existing secrets
  - Delete secrets with confirmation dialog
  - Search and filter secrets by key
  - Copy secret values to clipboard with auto-clear feedback
- **E2E Test Coverage** - All E2E tests enabled and passing
  - 6 authentication tests (SEC-001, SEC-002)
  - 6 secret CRUD tests (CORE-001 to CORE-006)

### Fixed
- Playwright strict mode violations in E2E tests
- Dialog handling in delete confirmation tests

## [0.4.0] - 2025-12-06

### Added
- **Desktop App** - Native desktop application built with Wails v2
  - Cross-platform support (macOS, Windows, Linux)
  - Vault creation with master password validation
  - Vault unlock with password authentication
  - Secrets list view with metadata display
  - Password visibility toggle
  - Auto-lock on idle timeout (15 minutes)
  - Modern React + TypeScript + Tailwind CSS frontend
  - Playwright E2E test framework

### Changed
- Reorganized project structure with `desktop/` directory for Wails app

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
