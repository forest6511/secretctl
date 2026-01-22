# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.8] - 2026-01-23

### Added
- **Competitor Import** - Migrate secrets from other password managers (#168)
  - 1Password CSV export import
  - Bitwarden JSON export import
  - LastPass CSV export import
  - Automatic field mapping (username, password, url, notes)
  - Support for `--preserve-case` and `--tag` options
- **Folder Organization** - Organize secrets into hierarchical folders (#167)
  - CLI commands: `folder create`, `folder list`, `folder delete`, `folder rename`, `folder move`, `folder info`
  - MCP tools: `folder_list`, `folder_create`, `folder_move_secret`
  - Folder customization with icons and colors
  - Path-based secret organization (e.g., `work/aws/api-key`)

### Fixed
- MCP server version string updated from hardcoded 0.6.0 to match release version (#173)

## [0.8.7] - 2026-01-22

### Added
- **Security Dashboard** - Analyze vault security health with comprehensive scoring (#169)
  - Overall security score (0-100) based on password strength, uniqueness, and expiration
  - Duplicate password detection using privacy-preserving HMAC comparison
  - Weak password detection with NIST-compliant strength analysis
  - CLI commands: `secretctl security`, `security duplicates`, `security weak`, `security expiring`
- **Dark Mode** - Full dark theme support in Desktop app (#169)
  - System preference detection with manual override
  - Consistent theming across all components
- **Keyboard Shortcuts** - Quick navigation and actions in Desktop app (#169)
  - `Ctrl+N` / `Cmd+N`: New secret
  - `Ctrl+S` / `Cmd+S`: Save
  - `Ctrl+F` / `Cmd+F`: Focus search
  - `Escape`: Cancel/close dialogs
- **Shell Completion** - Tab completion for all CLI commands (#169)
  - `secretctl completion bash`
  - `secretctl completion zsh`
  - `secretctl completion fish`
  - `secretctl completion powershell`
- **Internationalization (i18n)** - Multi-language support (#163)
  - Japanese language support for Desktop app and Website
  - Browser language auto-detection with English fallback

### Documentation
- Security Dashboard documentation (#170)
- Japanese README with language badges (#164)
- Rewritten README with pain-first messaging (#165)
- Google Search Console verification (#171)
- FUNDING.yml for GitHub Sponsors (#166)

## [0.8.6] - 2026-01-06

### Added
- **Desktop App** - Input type selector in Add Field dialog (#156)
  - "Single Line" option for passwords, API keys, usernames
  - "Multi-line" option for SSH keys, certificates, JSON
  - Segmented button UI with icons and dynamic help text
- **Desktop App** - Custom app icon with shield and key design (#154)

### Changed
- **Desktop App** - Enhanced color theme with Docker-style accents (#155)
  - Sky-500 primary color for header section
  - Improved button hover and active states
  - Enhanced template card hover effect

### Maintenance
- Consolidated research docs into `docs/internal/research/` (#153)

## [0.8.5] - 2026-01-04

### Fixed
- **Desktop App** - Textarea edit mode now supports password-like masking (#150)
  - Uses `-webkit-text-security: disc` for consistent UX with single-line inputs
  - Textarea defaults to visible (for SSH key verification), Input defaults to hidden
  - Eye button toggles visibility for both field types
- **Desktop App** - Fixed textarea becoming readonly after typing one character (#149)
  - Separated display masking from input blocking logic
  - Added regression tests to prevent future issues
- **Desktop App** - Fixed new SSH secrets not accepting textarea input (#147)
  - SSH key template now properly renders textarea for `private_key` field

### Added
- **Testing** - Behavior Matrix Tests for automatic UX consistency verification
  - Uses `describe.each()` pattern to test Input and Textarea with same scenarios
  - Ensures ADR-005 "Same UX as single-line Input" compliance

## [0.8.4] - 2025-12-31

### Added
- **Textarea Field Support** - Multi-line input for SSH keys, certificates, and other large text values
  - New `inputType` field attribute: `"text"` (default) or `"textarea"`
  - SSH template's `private_key` field now uses textarea input
  - Desktop App: Textarea component with proper masking for sensitive fields
  - CLI: Multi-line input support when `inputType` is `"textarea"`
- **Documentation** - Multi-Field Secrets section in Desktop Guide, SSH key instructions

## [0.7.0] - 2025-12-30

### Added
- **Multi-Field Secrets (Phase 2.5)** - Support for storing multiple key-value pairs in a single secret
  - New `Fields` map with name, value, and sensitivity settings per field
  - Database schema migration to v3 with backward compatibility
  - Well-known field names: `username`, `password`, `url`, `notes`, `host`, `port`, `database`, `api_key`

- **CLI Multi-Field Support**
  - `set --field name=value` for adding individual fields
  - `set --template login|database|api|ssh` for pre-defined templates
  - `set --binding field=ENV_VAR` for environment variable mappings
  - `get --field name` to retrieve a specific field
  - `get --fields` to list all field names

- **MCP Multi-Field Tools**
  - `secret_list` now includes `field_count` for each secret
  - `secret_list_fields` to list all field names for a secret
  - `secret_get_field` to retrieve a specific non-sensitive field value
  - `secret_get_masked` extended with `fields` map showing all fields with masking
  - `secret_run_with_bindings` for flexible environment variable injection

- **Desktop App Multi-Field UI**
  - Multi-field editing interface with add/remove field buttons
  - Template selection dropdown (Login, Database, API, SSH)
  - Sensitive field toggle per field
  - Field-level visibility controls

- **Documentation**
  - Well-known field names reference (`website/docs/reference/field-names.md`)
  - Updated MCP tools documentation with multi-field examples

### Security
- Sensitive field values are always masked in MCP responses
- Non-sensitive fields can be read via `secret_get_field` (e.g., username, host)
- `field_count` stored in plaintext for efficient querying (AI-Safe Access compliant)

## [0.6.0] - 2025-12-24

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
- **AI-Safe Access design**: AI agents never receive plaintext secrets
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

[Unreleased]: https://github.com/forest6511/secretctl/compare/v0.8.7...HEAD
[0.8.7]: https://github.com/forest6511/secretctl/compare/v0.8.6...v0.8.7
[0.8.6]: https://github.com/forest6511/secretctl/compare/v0.8.5...v0.8.6
[0.8.5]: https://github.com/forest6511/secretctl/compare/v0.8.4...v0.8.5
[0.8.4]: https://github.com/forest6511/secretctl/compare/v0.7.0...v0.8.4
[0.7.0]: https://github.com/forest6511/secretctl/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/forest6511/secretctl/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/forest6511/secretctl/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/forest6511/secretctl/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/forest6511/secretctl/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/forest6511/secretctl/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/forest6511/secretctl/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/forest6511/secretctl/releases/tag/v0.1.0
