---
title: From .env Files
description: Import secrets from .env files to secretctl.
sidebar_position: 2
---

# Migrate from .env Files

This guide shows you how to migrate your existing `.env` files to secretctl's encrypted vault.

## Why Migrate?

`.env` files have several security issues:

- **Stored in plaintext** - Anyone with file access can read your secrets
- **Often committed to git** - Easy to accidentally leak credentials
- **No access control** - Can't restrict who sees which secrets
- **No audit trail** - No record of when secrets were accessed or changed

secretctl solves these problems with:

- **AES-256-GCM encryption** - Secrets encrypted at rest
- **Master password protection** - Only authorized users can decrypt
- **Audit logging** - Full trail of secret access and modifications
- **AI-safe design** - Safe to use with Claude Code and other AI tools

## Quick Migration

### Step 1: Initialize Your Vault

If you haven't already:

```bash
secretctl init
```

### Step 2: Import Your .env File

```bash
# Preview what will be imported
secretctl import .env --dry-run

# Import all secrets
secretctl import .env
```

### Step 3: Verify the Import

```bash
secretctl list
```

### Step 4: Secure Your Old .env File

Once verified, remove or secure your old `.env` file:

```bash
# Option 1: Delete the file
rm .env

# Option 2: Add to .gitignore if you need it for local development
echo ".env" >> .gitignore
```

## Handling Conflicts

If you're importing into a vault that already has some secrets:

```bash
# Skip existing keys (keep vault values)
secretctl import .env --on-conflict=skip

# Overwrite existing keys (use .env values)
secretctl import .env --on-conflict=overwrite

# Stop on conflict (default behavior)
secretctl import .env --on-conflict=error
```

## Import from JSON

secretctl also supports JSON format:

```bash
# JSON file with key-value pairs
# {"API_KEY": "sk-xxx", "DB_PASSWORD": "secret"}
secretctl import config.json
```

## Multiple Environment Files

For projects with multiple environments:

```bash
# Import with environment prefixes
secretctl import .env.development --dry-run
secretctl import .env.production --dry-run

# Or import to separate "environments" using key prefixes
# Rename keys during import by editing the file first
```

## Working with Your Team

After migration, team members can:

1. Clone the project
2. Initialize their own vault: `secretctl init`
3. Import shared secrets (from a secure channel, not git)
4. Use `secretctl run` to inject secrets into commands

## Next Steps

- [Backup your vault](/docs/guides/cli/backup-restore) - Protect your encrypted secrets
- [Use with AI tools](/docs/getting-started/for-developers) - Set up Claude Code integration
- [Export when needed](/docs/reference/cli-commands#export) - Generate `.env` files for legacy tools
