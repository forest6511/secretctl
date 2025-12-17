---
title: Quick Start
description: Create your first secret in under 5 minutes.
sidebar_position: 3
---

# Quick Start

Get up and running with secretctl in 5 minutes.

## 1. Initialize Your Vault

```bash
secretctl init
```

You'll be prompted to create a master password. This password encrypts all your secrets.

:::caution
Remember your master password! It cannot be recovered if lost.
:::

## 2. Add Your First Secret

```bash
secretctl secret add api/openai --value "sk-your-api-key"
```

## 3. Retrieve a Secret

```bash
secretctl secret get api/openai
```

## 4. Use Secrets in Commands

```bash
secretctl run -k "api/*" -- your-command
```

This injects secrets as environment variables without exposing them in your shell history.

## Next Steps

- [Managing Secrets](/docs/guides/cli/managing-secrets) - Learn all secret operations
- [Running Commands](/docs/guides/cli/running-commands) - Advanced environment injection
- [MCP Integration](/docs/guides/mcp/) - Use with AI agents
