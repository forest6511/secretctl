---
title: MCP Integration
description: Integrate secretctl with AI agents via MCP.
sidebar_position: 1
---

# MCP Integration

secretctl includes a built-in MCP (Model Context Protocol) server for secure integration with AI coding assistants like Claude Code, Codex CLI, and other MCP-compatible tools.

## Overview

The MCP server enables AI assistants to work with your secrets **without ever seeing the actual secret values**. This is achieved through the [Option D+ security model](/docs/guides/mcp/security-model).

## Quick Start

### 1. Create Policy File (Required)

Before starting the MCP server, create a policy file to control which commands AI can execute:

```bash
mkdir -p ~/.secretctl
cat > ~/.secretctl/mcp-policy.yaml << 'EOF'
version: 1
default_action: deny
allowed_commands:
  - aws
  - gcloud
  - kubectl
EOF
```

### 2. Configure Your AI Tool

Add to your Claude Code configuration (`~/.claude.json`):

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

See [Claude Code Setup](/docs/guides/mcp/claude-code-setup) for detailed configuration.

## Features

- **Secure by Design**: AI agents never see plaintext secrets
- **Policy-Based Access Control**: Define which commands AI can execute
- **Output Sanitization**: Automatically redact leaked secrets from command output
- **Environment Aliases**: Switch between dev/staging/prod seamlessly

## Learn More

- [Security Model (Option D+)](/docs/guides/mcp/security-model) - How secrets are protected
- [Claude Code Setup](/docs/guides/mcp/claude-code-setup) - Detailed setup guide
- [Available Tools](/docs/guides/mcp/available-tools) - MCP tools reference
- [Environment Aliases](/docs/guides/mcp/env-aliases) - Multi-environment configuration
