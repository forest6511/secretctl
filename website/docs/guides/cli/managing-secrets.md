---
title: Managing Secrets
description: Learn how to add, view, update, and delete secrets using the secretctl CLI.
sidebar_position: 2
---

# Managing Secrets

This guide covers the core secret management operations: storing, retrieving, listing, and deleting secrets.

## Prerequisites

- [secretctl installed](/docs/getting-started/installation)
- [Vault initialized](/docs/getting-started/quick-start)

## Storing Secrets

Use the `set` command to store a secret. The value is read from standard input for security (avoiding shell history).

### Basic Usage

```bash
# Store a secret (value from stdin)
echo "sk-your-api-key" | secretctl set OPENAI_API_KEY
```

### With Metadata

Add context to your secrets with metadata:

```bash
# Add notes and tags
echo "mypassword" | secretctl set DB_PASSWORD \
  --notes="Production database credentials" \
  --tags="prod,database"

# Add a URL reference
echo "sk-xxx" | secretctl set API_KEY \
  --url="https://console.example.com/api-keys"

# Set expiration (e.g., 30 days, 1 year)
echo "temp-token" | secretctl set TEMP_TOKEN --expires="30d"
```

### Available Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--notes` | Add descriptive notes | `--notes="Production DB"` |
| `--tags` | Comma-separated tags | `--tags="prod,api"` |
| `--url` | Reference URL | `--url="https://..."` |
| `--expires` | Expiration duration | `--expires="30d"` or `--expires="1y"` |

## Retrieving Secrets

Use the `get` command to retrieve a secret value.

### Basic Usage

```bash
# Get secret value
secretctl get API_KEY
```

**Output:**
```
sk-abc123xyz789
```

### With Metadata

```bash
# Show metadata along with the value
secretctl get API_KEY --show-metadata
```

**Output:**
```
Key: API_KEY
Value: sk-abc123xyz789
Tags: api, prod
URL: https://console.example.com/api-keys
Notes: OpenAI API key for production
Created: 2025-01-15 10:30:00
Updated: 2025-01-15 10:30:00
```

## Listing Secrets

Use the `list` command to see all stored secrets.

### Basic Usage

```bash
# List all secret keys
secretctl list
```

**Output:**
```
API_KEY
DB_PASSWORD
AWS_ACCESS_KEY
AWS_SECRET_KEY
```

### Filter by Tag

```bash
# Show only secrets with specific tag
secretctl list --tag=prod
```

### Find Expiring Secrets

```bash
# Show secrets expiring within 7 days
secretctl list --expiring=7d

# Show secrets expiring within 30 days
secretctl list --expiring=30d
```

## Deleting Secrets

Use the `delete` command to remove a secret from the vault.

```bash
# Delete a secret
secretctl delete OLD_API_KEY
```

:::caution
Deletion is permanent. The secret cannot be recovered after deletion.
:::

## Updating Secrets

To update a secret, use the `set` command with the same key. The existing secret will be overwritten.

```bash
# Update an existing secret
echo "new-password" | secretctl set DB_PASSWORD

# Update with new metadata
echo "new-password" | secretctl set DB_PASSWORD \
  --notes="Updated 2025-01" \
  --tags="prod,rotated"
```

## Hierarchical Keys

Organize secrets using forward slashes to create a hierarchy:

```bash
# Store secrets with hierarchical keys
echo "access-key" | secretctl set aws/access_key
echo "secret-key" | secretctl set aws/secret_key
echo "host" | secretctl set db/prod/host
echo "password" | secretctl set db/prod/password
```

This enables powerful wildcard patterns with the `run` and `export` commands:

```bash
# Inject all AWS secrets
secretctl run -k "aws/*" -- aws s3 ls

# Export all production database secrets
secretctl export -k "db/prod/*" -o .env
```

## Best Practices

### Use Descriptive Keys

```bash
# Good: Clear, hierarchical naming
echo "xxx" | secretctl set github/personal_access_token
echo "xxx" | secretctl set aws/production/access_key

# Avoid: Ambiguous names
echo "xxx" | secretctl set token1
echo "xxx" | secretctl set key
```

### Add Metadata for Context

```bash
# Include notes for future reference
echo "xxx" | secretctl set STRIPE_API_KEY \
  --notes="Live key for production. Test key is in STRIPE_TEST_KEY" \
  --tags="stripe,payments,prod" \
  --url="https://dashboard.stripe.com/apikeys"
```

### Set Expiration for Temporary Secrets

```bash
# Temporary tokens should have expiration
echo "xxx" | secretctl set DEPLOY_TOKEN --expires="7d"

# Regular rotation reminders
echo "xxx" | secretctl set DB_PASSWORD --expires="90d"
```

### Use Tags for Organization

```bash
# Tag by environment
echo "xxx" | secretctl set API_KEY --tags="prod"
echo "xxx" | secretctl set API_KEY_DEV --tags="dev"

# Tag by service
echo "xxx" | secretctl set STRIPE_KEY --tags="stripe,payments"
echo "xxx" | secretctl set SENDGRID_KEY --tags="sendgrid,email"
```

## Troubleshooting

### "secret not found" Error

The specified key does not exist in the vault.

```bash
# List all secrets to verify the key
secretctl list

# Check for typos in the key name
secretctl get API_KEY  # correct
secretctl get api_key  # keys are case-sensitive
```

### "vault is locked" Error

The vault needs to be unlocked with your master password.

```bash
# Any command will prompt for the password
secretctl list
Enter master password: ********
```

### Input Not Being Read

Ensure you're piping the value correctly:

```bash
# Correct: pipe the value
echo "myvalue" | secretctl set MY_KEY

# Wrong: no pipe (will wait for input)
secretctl set MY_KEY
```

## Next Steps

- [Running Commands](/docs/guides/cli/running-commands) - Inject secrets as environment variables
- [Exporting Secrets](/docs/guides/cli/exporting-secrets) - Export to .env or JSON
- [CLI Commands Reference](/docs/reference/cli-commands) - Complete command reference
