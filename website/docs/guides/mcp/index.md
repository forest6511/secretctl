---
title: MCP Integration
description: Integrate secretctl with AI agents via MCP.
sidebar_position: 1
---

# MCP Integration

secretctl includes a built-in MCP (Model Context Protocol) server for secure AI agent integration.

## Quick Setup

Add to your Claude Code configuration:

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

## Security: Option D+

secretctl uses the **Option D+** security model - AI agents never receive plaintext secrets.

[Learn more about Option D+ →](/docs/guides/mcp/security-model)
