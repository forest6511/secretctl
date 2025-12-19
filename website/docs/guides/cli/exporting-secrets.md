---
title: Exporting Secrets
description: Learn how to export secrets to .env files and JSON format for use with other tools.
sidebar_position: 4
---

# Exporting Secrets

The `export` command allows you to export secrets to `.env` files or JSON format for use with Docker, CI/CD pipelines, or other tools.

## Prerequisites

- [secretctl installed](/docs/getting-started/installation)
- [Secrets stored in vault](/docs/guides/cli/managing-secrets)

## Basic Usage

```bash
secretctl export [flags]
```

By default, exports all secrets to stdout in `.env` format.

## Export Formats

### .env Format (Default)

```bash
# Export all secrets to stdout
secretctl export
```

**Output:**
```
API_KEY=sk-abc123
DB_PASSWORD=mypassword
AWS_ACCESS_KEY=AKIAIOSFODNN7
```

### JSON Format

```bash
# Export as JSON
secretctl export --format=json
```

**Output:**
```json
{
  "API_KEY": "sk-abc123",
  "DB_PASSWORD": "mypassword",
  "AWS_ACCESS_KEY": "AKIAIOSFODNN7"
}
```

### JSON with Metadata

```bash
# Include metadata in JSON output
secretctl export --format=json --with-metadata
```

**Output:**
```json
{
  "API_KEY": {
    "value": "sk-abc123",
    "tags": ["api", "prod"],
    "url": "https://console.example.com",
    "notes": "Production API key",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  },
  "DB_PASSWORD": {
    "value": "mypassword",
    "tags": ["db"],
    "expires_at": "2025-06-15T00:00:00Z",
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z"
  }
}
```

## Export to File

### .env File

```bash
# Export to .env file
secretctl export -o .env

# Export to specific path
secretctl export -o config/.env.production
```

### JSON File

```bash
# Export to JSON file
secretctl export --format=json -o secrets.json

# With metadata
secretctl export --format=json --with-metadata -o config.json
```

### Overwrite Protection

By default, secretctl will not overwrite existing files:

```bash
$ secretctl export -o .env
Error: file .env already exists. Use --force to overwrite.

# Force overwrite
$ secretctl export -o .env --force
Exported 5 secrets to .env
```

## Filtering Secrets

### Export Specific Keys

```bash
# Export only specific keys
secretctl export -k API_KEY -k DB_PASSWORD
```

### Wildcard Patterns

```bash
# Export all AWS secrets
secretctl export -k "aws/*" -o aws.env

# Export all database secrets as JSON
secretctl export -k "db/*" --format=json -o db-config.json
```

### Multiple Patterns

```bash
# Export secrets matching multiple patterns
secretctl export -k "aws/*" -k "db/*" -o infra.env
```

## Practical Examples

### Docker Compose

Create a `.env` file for Docker Compose:

```bash
# Export production secrets
secretctl export -k "prod/*" -o .env
```

```yaml
# docker-compose.yml
services:
  app:
    image: myapp
    env_file:
      - .env
```

### CI/CD Pipelines

Export secrets for CI/CD configuration:

```bash
# Export and source in shell
eval $(secretctl export)
./deploy.sh

# Or export to file and use
secretctl export -o .env
source .env
./deploy.sh
```

### Kubernetes Secrets

Generate Kubernetes secret manifests:

```bash
# Export as JSON and convert to Kubernetes secret
secretctl export -k "app/*" --format=json | \
  jq -r 'to_entries | map("\(.key)=\(.value)") | .[]' | \
  kubectl create secret generic app-secrets --from-env-file=/dev/stdin
```

### Application Configuration

Export for application-specific config:

```bash
# Export database config
secretctl export -k "db/*" --format=json -o config/database.json

# Export with metadata for documentation
secretctl export --format=json --with-metadata -o secrets-inventory.json
```

### Backup Secrets

Create a backup of all secrets:

```bash
# Full backup with metadata
secretctl export --format=json --with-metadata -o backup-$(date +%Y%m%d).json
```

:::caution
Exported files contain plaintext secrets. Store them securely and avoid committing to version control.
:::

### Environment-Specific Exports

```bash
# Export development secrets
secretctl export -k "dev/*" -o .env.development

# Export staging secrets
secretctl export -k "staging/*" -o .env.staging

# Export production secrets
secretctl export -k "prod/*" -o .env.production
```

## Piping to Other Commands

### Process with jq

```bash
# Get specific value from JSON export
secretctl export --format=json | jq -r '.API_KEY'

# List all keys
secretctl export --format=json | jq -r 'keys[]'

# Filter by pattern
secretctl export --format=json | jq 'with_entries(select(.key | startswith("DB_")))'
```

### Create Derived Files

```bash
# Create .env.example with redacted values
secretctl export | sed 's/=.*/=CHANGEME/' > .env.example

# Create documentation
secretctl export --format=json --with-metadata | \
  jq -r 'to_entries[] | "- **\(.key)**: \(.value.notes // "No description")"'
```

### Integration with Other Tools

```bash
# Sync to another secrets manager
secretctl export --format=json | vault kv put secret/myapp -

# Load into environment for script
env $(secretctl export | xargs) ./my-script.sh
```

## Key Transformation

When exporting, secret keys are transformed to valid environment variable names:

| Secret Key | Environment Variable |
|------------|---------------------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |

## Security Best Practices

### Avoid Version Control

Add exported files to `.gitignore`:

```bash
# .gitignore
.env
.env.*
*.secrets
secrets.json
```

### Secure File Permissions

Exported files should have restricted permissions:

```bash
# Export with secure permissions
secretctl export -o .env && chmod 600 .env
```

### Clean Up After Use

Remove exported files when no longer needed:

```bash
# Use in CI/CD, then clean up
secretctl export -o .env
source .env
./deploy.sh
rm .env
```

### Avoid Logging Exports

Be careful with stdout exports in logged environments:

```bash
# Bad: May be logged
echo $(secretctl export)

# Better: Direct to file
secretctl export -o .env
```

## Troubleshooting

### "file already exists" Error

Use `--force` to overwrite:

```bash
secretctl export -o .env --force
```

### Empty Output

Check that secrets exist and match your filter:

```bash
# List all secrets
secretctl list

# Test filter pattern
secretctl list | grep "aws"
```

### Permission Denied

Ensure you have write permission to the output directory:

```bash
# Check permissions
ls -la $(dirname .env)

# Export to writable location
secretctl export -o /tmp/.env
```

### JSON Parse Errors

When piping JSON output, ensure it's valid:

```bash
# Validate JSON
secretctl export --format=json | jq .
```

## Next Steps

- [Password Generation](/docs/guides/cli/password-generation) - Generate secure passwords
- [Running Commands](/docs/guides/cli/running-commands) - Inject secrets as environment variables
- [CLI Commands Reference](/docs/reference/cli-commands) - Complete command reference
