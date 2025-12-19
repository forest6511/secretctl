---
title: Running Commands with Secrets
description: Learn how to inject secrets as environment variables when running commands.
sidebar_position: 3
---

# Running Commands with Secrets

The `run` command executes any command with secrets injected as environment variables. This keeps secrets out of your shell history and command line arguments.

## Prerequisites

- [secretctl installed](/docs/getting-started/installation)
- [Secrets stored in vault](/docs/guides/cli/managing-secrets)

## Basic Usage

```bash
secretctl run -k <key> -- <command> [args...]
```

The `--` separates secretctl flags from the command you want to run.

### Single Secret

```bash
# Inject API_KEY and run curl
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" https://api.example.com
```

### Multiple Secrets

```bash
# Inject multiple secrets
secretctl run -k DB_HOST -k DB_USER -k DB_PASS -- psql -h $DB_HOST -U $DB_USER
```

## Wildcard Patterns

Use glob patterns to inject multiple secrets at once.

### Single-Level Wildcard

The `*` matches secrets at a single level:

```bash
# aws/* matches aws/access_key, aws/secret_key
secretctl run -k "aws/*" -- aws s3 ls

# db/* matches db/host, db/password (but not db/prod/host)
secretctl run -k "db/*" -- ./connect.sh
```

:::info
Always quote wildcard patterns to prevent shell expansion: `-k "aws/*"` not `-k aws/*`
:::

### Pattern Examples

| Pattern | Matches | Does NOT Match |
|---------|---------|----------------|
| `aws/*` | `aws/access_key`, `aws/secret_key` | `aws/prod/key` |
| `db/*` | `db/host`, `db/password` | `db/prod/host` |
| `API_KEY` | `API_KEY` (exact match) | `API_KEY_DEV` |

## Environment Variable Naming

Secret keys are transformed to valid environment variable names:

- `/` is replaced with `_`
- `-` is replaced with `_`
- Names are converted to UPPERCASE

| Secret Key | Environment Variable |
|------------|---------------------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |
| `API_KEY` | `API_KEY` |

## Output Sanitization

By default, secretctl scans command output for secret values and redacts them:

```bash
$ secretctl run -k DB_PASSWORD -- echo "Password is $DB_PASSWORD"
Password is [REDACTED:DB_PASSWORD]
```

This prevents accidental secret leakage in logs or terminal output.

### Disable Sanitization

For debugging or when you need raw output:

```bash
secretctl run -k API_KEY --no-sanitize -- ./script.sh
```

:::caution
Use `--no-sanitize` carefully. Secrets may appear in terminal output, logs, or be captured by screen recording.
:::

## Command Timeout

Set a timeout to prevent long-running commands:

```bash
# Timeout after 30 seconds
secretctl run -k API_KEY --timeout=30s -- ./slow-script.sh

# Timeout after 5 minutes (default)
secretctl run -k API_KEY --timeout=5m -- ./deploy.sh
```

### Timeout Format

| Format | Duration |
|--------|----------|
| `30s` | 30 seconds |
| `5m` | 5 minutes |
| `1h` | 1 hour |

## Environment Aliases

Use environment aliases to map secrets to different environments without changing your scripts.

### Setup

First, configure aliases in `~/.secretctl/mcp-policy.yaml`:

```yaml
version: 1
default_action: allow

env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### Usage

```bash
# Uses dev/db/host, dev/db/password
secretctl run --env=dev -k "db/*" -- ./app

# Uses prod/db/host, prod/db/password
secretctl run --env=prod -k "db/*" -- ./app
```

This allows the same command to work across environments without modification.

## Environment Variable Prefix

Add a prefix to all injected environment variables:

```bash
# Variables become APP_API_KEY, APP_DB_PASSWORD
secretctl run -k API_KEY -k DB_PASSWORD --env-prefix=APP_ -- ./app
```

This is useful when your application expects a specific prefix.

## Practical Examples

### AWS CLI

```bash
# Store AWS credentials
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# Run AWS commands
secretctl run -k "aws/*" -- aws s3 ls
secretctl run -k "aws/*" -- aws ec2 describe-instances
```

### Docker

```bash
# Pass secrets to Docker build
secretctl run -k GITHUB_TOKEN -- docker build \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t myapp .

# Run container with secrets
secretctl run -k "db/*" -- docker run \
  -e DB_HOST=$DB_HOST \
  -e DB_PASSWORD=$DB_PASSWORD \
  myapp
```

### Node.js / npm

```bash
# Run npm scripts with secrets
secretctl run -k API_KEY -k DATABASE_URL -- npm start

# Run tests with test credentials
secretctl run -k "test/*" -- npm test
```

### Database Connections

```bash
# PostgreSQL
secretctl run -k DB_HOST -k DB_USER -k DB_PASSWORD -- \
  psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST/mydb"

# MySQL
secretctl run -k MYSQL_HOST -k MYSQL_USER -k MYSQL_PASSWORD -- \
  mysql -h $MYSQL_HOST -u $MYSQL_USER -p$MYSQL_PASSWORD
```

### CI/CD Scripts

```bash
# Deploy script
secretctl run -k DEPLOY_TOKEN -k AWS_ACCESS_KEY -k AWS_SECRET_KEY -- ./deploy.sh

# With timeout for long deployments
secretctl run -k "deploy/*" --timeout=30m -- ./full-deploy.sh
```

## Security Considerations

### Secrets in Shell History

The `run` command keeps secrets out of shell history:

```bash
# Good: Secret not in history
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" ...

# Bad: Secret visible in history
curl -H "Authorization: Bearer sk-abc123" ...
```

### Process Environment

Secrets are only available to the child process and are not visible in the parent shell:

```bash
# After this command, $API_KEY is NOT set in your shell
secretctl run -k API_KEY -- ./script.sh
echo $API_KEY  # Empty
```

### Blocked Commands

Certain commands are always blocked for security:

- `env` - Would expose all environment variables
- `printenv` - Would expose all environment variables
- `set` - Could expose variables
- `export` - Could leak to shell

## Troubleshooting

### "secret not found" Error

Verify the secret exists:

```bash
secretctl list
secretctl get MY_KEY
```

### "command not found" Error

Ensure the command is in your PATH or use the full path:

```bash
# Use full path
secretctl run -k API_KEY -- /usr/local/bin/myapp

# Or ensure PATH is correct
secretctl run -k API_KEY -- bash -c 'which myapp && myapp'
```

### Wildcards Not Matching

Remember that `*` only matches one level:

```bash
# This matches aws/key, not aws/prod/key
secretctl run -k "aws/*" -- ./script.sh

# Check what secrets exist
secretctl list | grep aws
```

### Timeout Issues

Increase timeout for long-running commands:

```bash
# Default is 5 minutes, increase if needed
secretctl run -k API_KEY --timeout=30m -- ./long-running-task.sh
```

## Next Steps

- [Exporting Secrets](/docs/guides/cli/exporting-secrets) - Export to .env or JSON files
- [MCP Integration](/docs/guides/mcp/) - Use secrets with AI coding assistants
- [CLI Commands Reference](/docs/reference/cli-commands) - Complete command reference
