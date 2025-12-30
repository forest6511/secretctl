---
title: Core Concepts
description: Understand vaults, secrets, and how secretctl keeps your data safe.
sidebar_position: 4
---

# Core Concepts

## Vault

A vault is an encrypted SQLite database that stores all your secrets. By default, it's located at `~/.secretctl/vault.db`.

## Secrets

Secrets are key-value pairs stored in your vault. Keys use path-like notation for organization:

```
api/openai
db/production/password
aws/access-key
```

## Encryption

secretctl uses industry-standard encryption:

- **AES-256-GCM** for encrypting secret values
- **Argon2id** for deriving encryption keys from your master password

## AI-Safe Access (AI Security)

When using secretctl with AI agents (MCP), secrets are never exposed in plaintext:

- `secret_run`: Injects secrets as environment variables
- `secret_get_masked`: Returns masked values like `****WXYZ`
- No `secret_get` for plaintext via MCP

[Learn more about AI-Safe Access →](/docs/guides/mcp/security-model)
