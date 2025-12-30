---
title: AI Agent Integration
description: Use secretctl with Claude Code and other AI coding assistants through the MCP server.
sidebar_position: 3
---

# AI Agent Integration

secretctl provides first-class support for AI coding assistants through the Model Context Protocol (MCP). This guide covers integration with Claude Code and other MCP-compatible tools.

## Overview

### The Challenge

AI coding assistants need access to secrets for tasks like:
- Running tests that require API keys
- Deploying to cloud services
- Accessing databases
- Authenticating with external services

### The Problem with Traditional Approaches

Exposing raw secrets to AI agents creates risks:
- **Non-deterministic behavior** - LLMs may inadvertently expose secrets
- **Prompt injection** - Malicious prompts could extract secrets
- **Logging exposure** - Secrets may appear in conversation logs
- **No revocation** - Hard to invalidate exposed credentials

### secretctl's Solution: AI-Safe Access

secretctl follows the "Access Without Exposure" principle:

| Feature | Traditional | secretctl MCP |
|---------|-------------|---------------|
| Raw secret access | Yes | **No** |
| Command execution | Manual | **Automated** |
| Output sanitization | No | **Yes** |
| Audit logging | No | **Yes** |
| Policy control | No | **Yes** |

The AI agent can **use** secrets without **seeing** them.

## Claude Code Setup

### 1. Configure MCP Server

Add secretctl to your Claude Code configuration:

```json
// ~/.claude.json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

### 2. Create MCP Policy

Configure allowed operations:

```yaml
# ~/.secretctl/mcp-policy.yaml
version: 1
default_action: deny

# Commands the AI can execute with secrets
allowed_commands:
  - npm
  - node
  - python
  - go
  - cargo
  - aws
  - gcloud
  - kubectl
  - curl
  - ./deploy.sh
  - ./test.sh

# Commands never allowed
denied_commands:
  - rm
  - sudo
  - chmod
  - chown

# Environment aliases
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### 3. Set File Permissions

```bash
# Policy file must have secure permissions
chmod 600 ~/.secretctl/mcp-policy.yaml
```

### 4. Verify Setup

Restart Claude Code and verify the MCP server is connected.

## Available MCP Tools

### secret_list

List all secret keys (values not exposed):

```
Claude: "What secrets do I have available?"
→ Calls secret_list
→ Returns: ["API_KEY", "DATABASE_URL", "AWS_ACCESS_KEY"]
```

### secret_exists

Check if a specific secret exists:

```
Claude: "Do I have AWS credentials configured?"
→ Calls secret_exists("aws/access_key")
→ Returns: true/false
```

### secret_get_masked

Get metadata and masked value:

```
Claude: "Show me info about the API key"
→ Calls secret_get_masked("API_KEY")
→ Returns: {
    key: "API_KEY",
    maskedValue: "****xyz",
    tags: ["api", "prod"],
    notes: "OpenAI API key"
  }
```

### secret_run

Execute commands with secrets injected:

```
Claude: "Run the tests with the API key"
→ Calls secret_run({
    keys: ["API_KEY"],
    command: "npm test"
  })
→ Executes with API_KEY in environment
→ Output sanitized (secrets redacted)
```

## Practical Scenarios

### Running Tests

```
You: "Run the integration tests"

Claude:
1. Calls secret_list → finds TEST_API_KEY, TEST_DATABASE_URL
2. Calls secret_run({
     keys: ["TEST_API_KEY", "TEST_DATABASE_URL"],
     command: "npm run test:integration"
   })
3. Returns sanitized test output
```

### Deploying to AWS

```
You: "Deploy to production"

Claude:
1. Calls secret_exists("aws/*") → confirms AWS credentials exist
2. Calls secret_run({
     keys: ["aws/*"],
     command: "aws s3 sync ./dist s3://my-bucket"
   })
3. Returns deployment result (credentials never visible)
```

### Database Queries

```
You: "Check the user count in the database"

Claude:
1. Calls secret_run({
     keys: ["DATABASE_URL"],
     command: "psql $DATABASE_URL -c 'SELECT COUNT(*) FROM users'"
   })
2. Returns query result
```

### API Calls

```
You: "Fetch my GitHub repos"

Claude:
1. Calls secret_run({
     keys: ["GITHUB_TOKEN"],
     command: "curl -H 'Authorization: Bearer $GITHUB_TOKEN' https://api.github.com/user/repos"
   })
2. Returns repo list (token never exposed)
```

## Security Model

### What AI Agents CAN Do

- ✅ List available secret keys
- ✅ Check if secrets exist
- ✅ See masked values (last 4 characters)
- ✅ Read metadata (tags, notes, URLs)
- ✅ Execute allowed commands with secrets
- ✅ Receive sanitized output

### What AI Agents CANNOT Do

- ❌ Read plaintext secret values
- ❌ Execute denied commands
- ❌ Bypass output sanitization
- ❌ Access secrets outside policy
- ❌ Modify or delete secrets

### Output Sanitization

All command output is scanned for secret values:

```
# If API_KEY = "sk-abc123xyz"
# Original output: "Connected with key sk-abc123xyz"
# Sanitized output: "Connected with key [REDACTED:API_KEY]"
```

### Audit Logging

All MCP operations are logged:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "action": "secret.run",
  "source": "mcp",
  "keys": ["API_KEY"],
  "command": "npm test",
  "success": true
}
```

## Environment Aliases

Use environment aliases for context-aware secret mapping:

### Configuration

```yaml
# ~/.secretctl/mcp-policy.yaml
env_aliases:
  development:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
  production:
    - pattern: "db/*"
      target: "prod/db/*"
```

### Usage

```
You: "Run tests against staging"

Claude:
→ Calls secret_run({
    keys: ["db/*"],
    command: "npm test",
    env: "staging"
  })
→ Uses staging/db/* secrets automatically
```

## Policy Best Practices

### Principle of Least Privilege

Only allow commands the AI actually needs:

```yaml
# Good: Specific allowed commands
allowed_commands:
  - npm test
  - npm run build
  - ./deploy.sh

# Bad: Too permissive
allowed_commands:
  - "*"
```

### Block Dangerous Commands

Always deny potentially dangerous operations:

```yaml
denied_commands:
  - rm
  - sudo
  - chmod
  - chown
  - mv
  - dd
  - mkfs
```

### Environment Separation

Use separate secrets for different environments:

```yaml
env_aliases:
  dev:
    - pattern: "*"
      target: "dev/*"
  prod:
    - pattern: "*"
      target: "prod/*"
```

### Regular Audit Review

Monitor AI agent activity:

```bash
# Review MCP access
secretctl audit export | jq '.[] | select(.source == "mcp")'

# Check for failures
secretctl audit export | jq '.[] | select(.success == false)'
```

## Troubleshooting

### MCP Server Not Connecting

1. Verify secretctl path in configuration
2. Check SECRETCTL_PASSWORD is set
3. Ensure vault exists and is accessible
4. Check Claude Code logs for errors

### "Command not allowed" Error

Add the command to `allowed_commands` in policy:

```yaml
allowed_commands:
  - your-command
```

### "Secret not found" Error

1. Verify secret exists: `secretctl list`
2. Check key spelling (case-sensitive)
3. Verify pattern matches: `secretctl list | grep pattern`

### Output Not Sanitized

Sanitization only works for exact matches. If secrets appear:
1. Check the secret value is stored correctly
2. Verify the output format matches the stored value

### Permission Denied

```bash
# Check policy file permissions
ls -la ~/.secretctl/mcp-policy.yaml
# Should be: -rw------- (600)

chmod 600 ~/.secretctl/mcp-policy.yaml
```

## Comparison with Alternatives

### vs. Hardcoded Secrets

| Aspect | Hardcoded | secretctl MCP |
|--------|-----------|---------------|
| Security | ❌ Exposed in code | ✅ Never exposed |
| Audit | ❌ None | ✅ Full audit trail |
| Rotation | ❌ Code changes | ✅ Update vault only |

### vs. Environment Variables

| Aspect | Env Vars | secretctl MCP |
|--------|----------|---------------|
| AI visibility | ❌ Visible | ✅ Hidden |
| Output safety | ❌ Can leak | ✅ Sanitized |
| Policy control | ❌ None | ✅ Full control |

### vs. Other Secret Managers

| Aspect | Others | secretctl MCP |
|--------|--------|---------------|
| AI-native | ❌ Bolted on | ✅ Built-in |
| Local-first | ❌ Cloud dependent | ✅ Offline capable |
| Zero exposure | ❌ API returns secrets | ✅ Run-only model |

## Next Steps

- [Developer Workflows](/docs/use-cases/developer-workflows) - Local dev and CI/CD
- [MCP Tools Reference](/docs/reference/mcp-tools) - Complete tool specifications
- [Configuration](/docs/reference/configuration) - Policy configuration details
