---
title: Environment Aliases
description: Map secret keys to different environments.
sidebar_position: 5
---

# Environment Aliases

Environment aliases allow seamless switching between different secret prefixes (dev/staging/prod) without changing secret key patterns. This is particularly useful for AI assistants that need to work with different environments.

## Overview

Instead of hardcoding environment-specific paths in your secret requests:

```json
// Without aliases - must specify full path
{ "keys": ["prod/db/host", "prod/db/password"] }
```

You can use aliases to dynamically map keys:

```json
// With aliases - specify pattern and environment
{ "keys": ["db/*"], "env": "prod" }
```

## Configuration

Define aliases in your policy file (`~/.secretctl/mcp-policy.yaml`):

```yaml
version: 1
default_action: deny
allowed_commands:
  - kubectl
  - aws

env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
    - pattern: "api/*"
      target: "staging/api/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
    - pattern: "api/*"
      target: "prod/api/*"
```

## How It Works

1. Define aliases in your policy file under `env_aliases`
2. Use the `env` parameter in `secret_run` to select an alias
3. Key patterns are transformed before secret lookup

**Example:**

With the policy above:

```json
{
  "command": "kubectl",
  "args": ["apply", "-f", "deployment.yaml"],
  "keys": ["db/*"],
  "env": "prod"
}
```

The key pattern `db/*` is transformed to `prod/db/*`, so secrets like:

- `prod/db/host`
- `prod/db/password`
- `prod/db/username`

...will be injected as environment variables.

## Pattern Matching

| Pattern | Key | Result |
|---------|-----|--------|
| `db/*` | `db/host` | Matches suffix `host` |
| `api/*` | `api/v1/key` | Matches suffix `v1/key` |
| `special_key` | `special_key` | Exact match |

## CLI Usage

The `--env` flag is also available for the CLI `run` command:

```bash
# Use dev environment secrets
secretctl run --env=dev -k "db/*" -- ./app

# Use prod environment secrets
secretctl run --env=prod -k "api/*" -- kubectl apply -f deployment.yaml
```

## Use Cases

### Multi-Environment Deployments

```yaml
env_aliases:
  dev:
    - pattern: "k8s/*"
      target: "dev/k8s/*"
  staging:
    - pattern: "k8s/*"
      target: "staging/k8s/*"
  prod:
    - pattern: "k8s/*"
      target: "prod/k8s/*"
```

AI can then deploy to any environment:

```
"Deploy the app to staging using k8s/* secrets"
```

### Database Connections

```yaml
env_aliases:
  local:
    - pattern: "postgres/*"
      target: "local/postgres/*"
  cloud:
    - pattern: "postgres/*"
      target: "cloud/postgres/*"
```

### API Key Management

```yaml
env_aliases:
  test:
    - pattern: "stripe/*"
      target: "test/stripe/*"
  live:
    - pattern: "stripe/*"
      target: "live/stripe/*"
```

## Best Practices

1. **Use consistent naming** - Keep pattern names consistent across environments
2. **Document your aliases** - Add comments explaining each alias
3. **Limit production access** - Consider separate policy files for prod
4. **Test with dev first** - Verify commands work with dev before prod
