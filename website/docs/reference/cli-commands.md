---
title: CLI Commands
description: Complete CLI command reference for secretctl.
sidebar_position: 1
---

# CLI Commands Reference

Complete reference for all secretctl CLI commands.

## Global Options

```bash
secretctl [command] --help    # Show help for any command
```

---

## init

Initialize a new secret vault.

```bash
secretctl init
```

Creates a new encrypted vault at `~/.secretctl/vault.db`. You will be prompted to set a master password (minimum 8 characters).

**Example:**

```bash
$ secretctl init
Enter master password: ********
Confirm master password: ********
Vault initialized successfully.
```

---

## set

Store a secret value from standard input, or create multi-field secrets.

```bash
secretctl set [key] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--field name=value` | Add a field to the secret (repeatable) |
| `--binding ENV=field` | Add environment binding (repeatable) |
| `--sensitive name` | Mark a field as sensitive (repeatable) |
| `--notes string` | Add notes to the secret |
| `--tags string` | Comma-separated tags (e.g., `dev,api`) |
| `--url string` | Add URL reference to the secret |
| `--expires string` | Expiration duration (e.g., `30d`, `1y`) |

**Examples:**

```bash
# Basic usage (single value from stdin)
echo "sk-your-api-key" | secretctl set OPENAI_API_KEY

# Multi-field secret
secretctl set db/prod \
  --field host=db.example.com \
  --field port=5432 \
  --field user=admin \
  --field password=secret123 \
  --sensitive password

# With environment bindings
secretctl set db/prod \
  --field host=db.example.com \
  --field password=secret123 \
  --binding PGHOST=host \
  --binding PGPASSWORD=password \
  --sensitive password

# With metadata
echo "mypassword" | secretctl set DB_PASSWORD \
  --notes="Production database" \
  --tags="prod,db" \
  --url="https://console.example.com"

# With expiration
echo "temp-token" | secretctl set TEMP_TOKEN --expires="30d"
```

---

## get

Retrieve a secret value or specific fields.

```bash
secretctl get [key] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--field name` | Get a specific field value |
| `--fields` | List all field names (no values) |
| `--show-metadata` | Show metadata with the secret |

**Examples:**

```bash
# Get secret value only (legacy single-value)
secretctl get API_KEY

# Get a specific field from multi-field secret
secretctl get db/prod --field host

# List all field names
secretctl get db/prod --fields

# Get secret with metadata
secretctl get API_KEY --show-metadata
```

---

## delete

Delete a secret from the vault.

```bash
secretctl delete [key]
```

**Example:**

```bash
secretctl delete OLD_API_KEY
```

---

## list

List all secret keys in the vault.

```bash
secretctl list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--tag string` | Filter by tag |
| `--expiring string` | Show secrets expiring within duration (e.g., `7d`) |

**Examples:**

```bash
# List all secrets
secretctl list

# Filter by tag
secretctl list --tag=prod

# Show expiring secrets
secretctl list --expiring=7d
```

---

## run

Execute a command with secrets injected as environment variables.

```bash
secretctl run [flags] -- command [args...]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-k, --key stringArray` | Secret keys to inject (glob pattern supported) |
| `-t, --timeout duration` | Command timeout (default: `5m`) |
| `--env string` | Environment alias (e.g., `dev`, `staging`, `prod`) |
| `--env-prefix string` | Environment variable name prefix |
| `--no-sanitize` | Disable output sanitization |
| `--obfuscate-keys` | Obfuscate secret key names in error messages |

**Environment Variable Naming:**

Secret keys are transformed to environment variable names:

- `/` is replaced with `_`
- `-` is replaced with `_`
- Names are converted to UPPERCASE

| Secret Key | Environment Variable |
|------------|---------------------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |

**Examples:**

```bash
# Single secret
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" https://api.example.com

# Multiple secrets
secretctl run -k DB_HOST -k DB_USER -k DB_PASS -- psql

# Wildcard pattern (matches single level)
secretctl run -k "aws/*" -- aws s3 ls

# With timeout
secretctl run -k API_KEY --timeout=30s -- ./long-script.sh

# With environment alias
secretctl run --env=prod -k "db/*" -- ./deploy.sh

# With prefix
secretctl run -k API_KEY --env-prefix=APP_ -- ./app
```

**Output Sanitization:**

By default, command output is scanned for secret values. Any matches are replaced with `[REDACTED:key]`.

```bash
# If DB_PASSWORD contains "secret123"
$ secretctl run -k DB_PASSWORD -- echo "Password is $DB_PASSWORD"
Password is [REDACTED:DB_PASSWORD]
```

---

## export

Export secrets to `.env` or JSON format.

```bash
secretctl export [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-k, --key strings` | Keys to export (glob pattern supported) |
| `-f, --format string` | Output format: `env`, `json` (default: `env`) |
| `-o, --output string` | Output file path (default: stdout) |
| `--with-metadata` | Include metadata in JSON output |
| `--force` | Overwrite existing file without confirmation |

**Examples:**

```bash
# Export all secrets to stdout
secretctl export

# Export to .env file
secretctl export -o .env

# Export specific keys as JSON
secretctl export -k "aws/*" -f json -o config.json

# Export with metadata
secretctl export -f json --with-metadata -o secrets.json

# Pipe to another command
secretctl export -f json | jq '.DB_HOST'
```

---

## import

Import secrets from `.env` or JSON files.

```bash
secretctl import [file] [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--on-conflict string` | How to handle existing keys: `skip`, `overwrite`, `error` (default: `error`) |
| `--dry-run` | Preview what would be imported without making changes |

**Examples:**

```bash
# Import from .env file
secretctl import .env

# Import from JSON file
secretctl import config.json

# Preview changes without importing
secretctl import .env --dry-run

# Skip existing keys
secretctl import .env --on-conflict=skip

# Overwrite existing keys
secretctl import .env --on-conflict=overwrite
```

**Supported Formats:**

- `.env` files: Standard KEY=VALUE format
- JSON files: Object with key-value pairs `{"KEY": "value"}`

---

## generate

Generate cryptographically secure random passwords.

```bash
secretctl generate [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-l, --length int` | Password length (8-256, default: 24) |
| `-n, --count int` | Number of passwords to generate (1-100, default: 1) |
| `-c, --copy` | Copy first password to clipboard |
| `--exclude string` | Characters to exclude |
| `--no-uppercase` | Exclude uppercase letters |
| `--no-lowercase` | Exclude lowercase letters |
| `--no-numbers` | Exclude numbers |
| `--no-symbols` | Exclude symbols |

**Examples:**

```bash
# Generate default password (24 chars)
secretctl generate

# Generate 32-char password without symbols
secretctl generate -l 32 --no-symbols

# Generate 5 passwords
secretctl generate -n 5

# Generate and copy to clipboard
secretctl generate -c

# Exclude ambiguous characters
secretctl generate --exclude "0O1lI"
```

---

## audit

Manage audit logs.

### audit list

List audit log entries.

```bash
secretctl audit list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--limit int` | Maximum number of events to show (default: 100) |
| `--since string` | Show events since duration (e.g., `24h`) |

**Example:**

```bash
secretctl audit list --limit=50 --since=24h
```

### audit verify

Verify audit log HMAC chain integrity.

```bash
secretctl audit verify
```

**Example:**

```bash
$ secretctl audit verify
Audit log integrity verified. 1234 events checked.
```

### audit export

Export audit logs to JSON or CSV format.

```bash
secretctl audit export [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format string` | Output format: `json`, `csv` (default: `json`) |
| `-o, --output string` | Output file path (default: stdout) |
| `--since string` | Export events since duration (e.g., `30d`) |
| `--until string` | Export events until date (RFC 3339) |

**Example:**

```bash
secretctl audit export --format=csv -o audit.csv --since=30d
```

### audit prune

Delete old audit log entries.

```bash
secretctl audit prune [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--older-than string` | Delete logs older than duration (e.g., `12m` for 12 months) |
| `--dry-run` | Show what would be deleted without deleting |
| `-f, --force` | Skip confirmation prompt |

**Example:**

```bash
# Preview deletions
secretctl audit prune --older-than=12m --dry-run

# Delete with confirmation
secretctl audit prune --older-than=12m

# Delete without confirmation
secretctl audit prune --older-than=12m --force
```

---

## backup

Create an encrypted backup of the vault.

```bash
secretctl backup [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-o, --output string` | Output file path (required unless using `--stdout`) |
| `--stdout` | Output to stdout (for piping) |
| `--with-audit` | Include audit logs in backup |
| `--backup-password` | Use a separate backup password (prompted) |
| `--key-file string` | Encryption key file (32 bytes) |
| `-f, --force` | Overwrite existing file without confirmation |

**Examples:**

```bash
# Basic backup
secretctl backup -o vault-backup.enc

# Backup with audit logs
secretctl backup -o full-backup.enc --with-audit

# Backup to stdout (for piping to gpg, etc.)
secretctl backup --stdout | gpg --encrypt > backup.gpg

# Use separate backup password
secretctl backup -o backup.enc --backup-password

# Use key file for automation
secretctl backup -o backup.enc --key-file=backup.key

# Overwrite existing backup
secretctl backup -o backup.enc --force
```

---

## restore

Restore the vault from an encrypted backup.

```bash
secretctl restore <backup-file> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview what would be restored without making changes |
| `--verify-only` | Only verify backup integrity (no restore) |
| `--on-conflict string` | How to handle existing keys: `skip`, `overwrite`, `error` (default: `error`) |
| `--key-file string` | Decryption key file |
| `--with-audit` | Restore audit logs (overwrites existing) |
| `-f, --force` | Skip confirmation prompt |

**Examples:**

```bash
# Verify backup integrity
secretctl restore backup.enc --verify-only

# Preview restore without changes
secretctl restore backup.enc --dry-run

# Restore, skip existing keys
secretctl restore backup.enc --on-conflict=skip

# Restore, overwrite existing keys
secretctl restore backup.enc --on-conflict=overwrite

# Restore with audit logs
secretctl restore backup.enc --with-audit

# Use key file for decryption
secretctl restore backup.enc --key-file=backup.key
```

---

## security

Analyze the security health of your vault and get recommendations.

```bash
secretctl security [flags]
secretctl security [subcommand]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `duplicates` | List duplicate passwords (Free: top 3) |
| `expiring` | List secrets expiring soon |
| `weak` | List weak passwords (Free: top 3) |

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `-v, --verbose` | Show all details including suggestions |
| `--days int` | Expiration warning window in days (default: 30) |

**Score Components:**

The security score (0-100) is calculated from four components:

| Component | Max Points | Description |
|-----------|-----------|-------------|
| Password Strength | 25 | Average strength of password fields |
| Uniqueness | 25 | Percentage of unique passwords |
| Expiration | 25 | Percentage of non-expired secrets |
| Coverage | 25 | Field coverage for templated secrets (Phase 3, currently always 25) |

**Examples:**

```bash
# Show security score and top issues
secretctl security

# Show all components and suggestions
secretctl security --verbose

# Output in JSON format
secretctl security --json

# List duplicate passwords
secretctl security duplicates

# List secrets expiring within 7 days
secretctl security expiring --days=7

# List weak passwords
secretctl security weak
```

**Example Output:**

```
🔒 Security Score: 85/100 (Good)

Components:
  Password Strength: 20/25 ████████░░
  Uniqueness:        25/25 ██████████
  Expiration:        15/25 ██████░░░░
  Coverage:          25/25 ██████████

⚠️  Top Issues (2):
  1. [WEAK] "legacy-api": Password has insufficient strength
  2. [EXPIRING_SOON] "aws/temp": Expires in 5 days
```

Use `--verbose` to see actionable suggestions.

---

## mcp-server

Start the MCP server for AI coding assistant integration.

```bash
secretctl mcp-server
```

**Authentication:**

Set `SECRETCTL_PASSWORD` environment variable before starting:

```bash
SECRETCTL_PASSWORD=your-password secretctl mcp-server
```

**Available MCP Tools:**

| Tool | Description |
|------|-------------|
| `secret_list` | List secret keys with metadata (no values) |
| `secret_exists` | Check if a secret exists with metadata |
| `secret_get_masked` | Get masked secret value (e.g., `****WXYZ`) |
| `secret_run` | Execute command with secrets as environment variables |

**Policy Configuration:**

Create `~/.secretctl/mcp-policy.yaml` to configure allowed commands:

```yaml
version: 1
default_action: deny
allowed_commands:
  - aws
  - gcloud
  - kubectl
```

See [MCP Integration Guide](/docs/guides/mcp/) for detailed configuration.
