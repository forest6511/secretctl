---
title: Getting Started
description: Get up and running with secretctl in under 5 minutes.
sidebar_position: 1
---

# Getting Started with secretctl

**secretctl** is the simplest AI-ready secrets manager. It provides secure, local-first credential management with native support for AI agent integration via MCP (Model Context Protocol).

## Why secretctl?

- **Local-first**: Your secrets never leave your machine
- **AI-ready**: Built-in MCP server with Option D+ security (AI agents never see plaintext)
- **Simple**: Single binary, no server required
- **Secure**: AES-256-GCM encryption with Argon2id key derivation

## Quick Links

- [Installation](/docs/getting-started/installation) - Install secretctl on your system
- [Quick Start](/docs/getting-started/quick-start) - Create your first secret in 5 minutes
- [Core Concepts](/docs/getting-started/concepts) - Understand vaults, secrets, and encryption

## Choose Your Path

### CLI Users

If you prefer the command line:

```bash
# Install
brew install forest6511/tap/secretctl

# Initialize vault
secretctl init

# Add your first secret
secretctl secret add api/openai --value "sk-..."
```

[Continue with CLI Guide →](/docs/guides/cli/)

### Desktop App Users

If you prefer a graphical interface:

1. Download from [GitHub Releases](https://github.com/forest6511/secretctl/releases)
2. Open the app and create your vault
3. Start managing secrets visually

[Continue with Desktop Guide →](/docs/guides/desktop/)

### AI/MCP Integration

If you want to use secretctl with Claude Code or other AI agents:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp", "serve"]
    }
  }
}
```

[Continue with MCP Guide →](/docs/guides/mcp/)
