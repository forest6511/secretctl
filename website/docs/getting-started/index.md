---
title: Getting Started
description: Get up and running with secretctl in under 5 minutes.
sidebar_position: 1
---

# Getting Started with secretctl

**secretctl** is the simplest AI-ready secrets manager. It provides secure, local-first credential management with native support for AI agent integration via MCP (Model Context Protocol).

## Choose Your Path

<div className="row">
  <div className="col col--6">
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>For Developers</h3>
      </div>
      <div className="card__body">
        <p>Integrate with Claude Code, automate CI/CD, use MCP for AI-safe secret access.</p>
        <ul>
          <li>MCP server setup</li>
          <li>Claude Code integration</li>
          <li>Environment injection</li>
          <li>API automation</li>
        </ul>
      </div>
      <div className="card__footer">
        <a className="button button--primary button--block" href="/docs/getting-started/for-developers">
          Developer Guide →
        </a>
      </div>
    </div>
  </div>
  <div className="col col--6">
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>For General Users</h3>
      </div>
      <div className="card__body">
        <p>Simple, secure password management with the Desktop App or basic CLI.</p>
        <ul>
          <li>Desktop App setup</li>
          <li>Password organization</li>
          <li>Backup and restore</li>
          <li>No technical knowledge required</li>
        </ul>
      </div>
      <div className="card__footer">
        <a className="button button--secondary button--block" href="/docs/getting-started/for-users">
          User Guide →
        </a>
      </div>
    </div>
  </div>
</div>

## Why secretctl?

- **Local-first**: Your secrets never leave your machine
- **AI-ready**: Built-in MCP server with AI-Safe Access (AI agents never see plaintext)
- **Simple**: Single binary, no server required
- **Secure**: AES-256-GCM encryption with Argon2id key derivation

## Quick Links

- [Installation](/docs/getting-started/installation) - Install secretctl on your system
- [Quick Start](/docs/getting-started/quick-start) - Create your first secret in 5 minutes
- [Core Concepts](/docs/getting-started/concepts) - Understand vaults, secrets, and encryption

## Quick Start (5 minutes)

### Option 1: Desktop App

1. Download from [GitHub Releases](https://github.com/forest6511/secretctl/releases)
2. Open the app and create your vault
3. Start managing secrets visually

[Continue with Desktop Guide →](/docs/guides/desktop/)

### Option 2: CLI

```bash
# Download and install
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl

# Initialize vault
secretctl init

# Add your first secret
echo "sk-..." | secretctl set OPENAI_API_KEY
```

[Continue with CLI Guide →](/docs/guides/cli/)

### Option 3: AI/MCP Integration

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

[Continue with MCP Guide →](/docs/guides/mcp/)
