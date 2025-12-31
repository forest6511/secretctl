---
title: Field Names Reference
description: Standard field names and templates for multi-field secrets.
sidebar_position: 4
---

# Field Names Reference

secretctl supports multi-field secrets with predefined templates for common use cases. This reference documents the standard field names, sensitivity settings, and environment variable bindings.

## Templates Overview

| Template | Use Case | Fields | Default Bindings |
|----------|----------|--------|------------------|
| **Login** | Website credentials | 2 | None |
| **Database** | Database connections | 5 | PostgreSQL (`PGHOST`, etc.) |
| **API** | API credentials | 2 | `API_KEY`, `API_SECRET` |
| **SSH** | SSH authentication | 2 | None |

---

## Login Template

For storing website and service login credentials.

### Fields

| Field | Sensitive | Description |
|-------|-----------|-------------|
| `username` | No | Username or email address |
| `password` | Yes | Account password |

### Environment Bindings

No default bindings. Add custom bindings as needed:

```bash
secretctl set github/login \
  --field username=myuser \
  --field password=secret123 \
  --sensitive password \
  --binding GITHUB_USER=username \
  --binding GITHUB_TOKEN=password
```

### Desktop App

Select the **Login** template when creating a new secret to auto-configure these fields.

---

## Database Template

For storing database connection credentials. Pre-configured for PostgreSQL environment variables.

### Fields

| Field | Sensitive | Description |
|-------|-----------|-------------|
| `host` | No | Database server hostname |
| `port` | No | Database server port |
| `username` | No | Database username |
| `password` | Yes | Database password |
| `database` | No | Database name |

### Environment Bindings

| Environment Variable | Maps To |
|---------------------|---------|
| `PGHOST` | `host` |
| `PGPORT` | `port` |
| `PGUSER` | `username` |
| `PGPASSWORD` | `password` |
| `PGDATABASE` | `database` |

### CLI Example

```bash
# Create database secret with PostgreSQL bindings
secretctl set db/prod \
  --field host=db.example.com \
  --field port=5432 \
  --field username=admin \
  --field password=secret123 \
  --field database=myapp \
  --sensitive password \
  --binding PGHOST=host \
  --binding PGPORT=port \
  --binding PGUSER=username \
  --binding PGPASSWORD=password \
  --binding PGDATABASE=database

# Run psql with injected credentials
secretctl run -k db/prod -- psql
```

### MySQL/MariaDB

For MySQL, use custom bindings:

```bash
secretctl set db/mysql \
  --field host=mysql.example.com \
  --field port=3306 \
  --field username=root \
  --field password=secret \
  --field database=mydb \
  --sensitive password \
  --binding MYSQL_HOST=host \
  --binding MYSQL_TCP_PORT=port \
  --binding MYSQL_USER=username \
  --binding MYSQL_PWD=password \
  --binding MYSQL_DATABASE=database
```

---

## API Template

For storing API keys and secrets.

### Fields

| Field | Sensitive | Description |
|-------|-----------|-------------|
| `api_key` | Yes | API key or access token |
| `api_secret` | Yes | API secret or private key |

### Environment Bindings

| Environment Variable | Maps To |
|---------------------|---------|
| `API_KEY` | `api_key` |
| `API_SECRET` | `api_secret` |

### CLI Example

```bash
# Create API credentials
secretctl set stripe/live \
  --field api_key=sk_live_xxx \
  --field api_secret=whsec_xxx \
  --sensitive api_key \
  --sensitive api_secret \
  --binding STRIPE_API_KEY=api_key \
  --binding STRIPE_WEBHOOK_SECRET=api_secret

# Run with API credentials
secretctl run -k stripe/live -- ./process-webhooks.sh
```

### Service-Specific Bindings

Different services use different environment variable names. Examples:

**OpenAI:**
```bash
secretctl set openai/prod \
  --field api_key=sk-proj-xxx \
  --sensitive api_key \
  --binding OPENAI_API_KEY=api_key
```

**AWS:**
```bash
secretctl set aws/prod \
  --field api_key=AKIAIOSFODNN7EXAMPLE \
  --field api_secret=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --sensitive api_key \
  --sensitive api_secret \
  --binding AWS_ACCESS_KEY_ID=api_key \
  --binding AWS_SECRET_ACCESS_KEY=api_secret
```

---

## SSH Template

For storing SSH private keys and passphrases.

### Fields

| Field | Sensitive | Input Type | Description |
|-------|-----------|------------|-------------|
| `private_key` | Yes | `textarea` | SSH private key content (multi-line) |
| `passphrase` | Yes | `text` | Key passphrase (optional) |

:::tip Multi-line Input
The `private_key` field uses a textarea input in the Desktop App, making it easy to paste PEM-format SSH keys. The CLI also supports multi-line input for this field.
:::

### Environment Bindings

No default bindings. SSH keys are typically written to files rather than environment variables.

### CLI Example

```bash
# Store SSH key
secretctl set ssh/server1 \
  --field private_key="$(cat ~/.ssh/id_ed25519)" \
  --field passphrase=mypassphrase \
  --sensitive private_key \
  --sensitive passphrase

# Get specific field
secretctl get ssh/server1 --field private_key > /tmp/key
chmod 600 /tmp/key
ssh -i /tmp/key user@server1
rm /tmp/key
```

---

## Field Attributes

Each field has the following attributes:

| Attribute | Type | Description |
|-----------|------|-------------|
| `value` | string | The field's secret value |
| `sensitive` | boolean | Whether the value should be masked |
| `inputType` | string | UI input type: `"text"` (default) or `"textarea"` |
| `kind` | string | Reserved for Phase 3 schema validation (optional) |
| `aliases` | string[] | Alternative names for the field (optional) |
| `hint` | string | Helper text shown in UI (optional) |

### Input Type

The `inputType` attribute controls how the field is rendered in the Desktop App:

| Input Type | Use Case | Example Fields |
|------------|----------|----------------|
| `text` | Single-line values | `username`, `password`, `api_key` |
| `textarea` | Multi-line values | `private_key`, certificates, configs |

When using templates, the `inputType` is automatically set based on the field's typical content.

---

## Field Naming Conventions

### Rules

Field names must follow these rules:

- **Characters**: Lowercase letters, numbers, underscores only
- **Format**: `snake_case` (e.g., `api_key`, `private_key`)
- **Length**: Maximum 64 characters
- **Reserved**: Cannot start with underscore

### Valid Examples

```
username
password
api_key
api_secret
private_key
database_url
connection_string
```

### Invalid Examples

```
apiKey          # camelCase not allowed
API_KEY         # uppercase not allowed
api-key         # hyphens not allowed
api key         # spaces not allowed
_private        # cannot start with underscore
```

---

## Custom Templates

While secretctl provides 4 built-in templates, you can create any field structure using CLI flags:

```bash
# Custom OAuth credentials
secretctl set oauth/google \
  --field client_id=xxx.apps.googleusercontent.com \
  --field client_secret=GOCSPX-xxx \
  --field refresh_token=1//xxx \
  --sensitive client_secret \
  --sensitive refresh_token \
  --binding GOOGLE_CLIENT_ID=client_id \
  --binding GOOGLE_CLIENT_SECRET=client_secret \
  --binding GOOGLE_REFRESH_TOKEN=refresh_token
```

---

## MCP Integration

### Discovering Fields

AI agents can discover field structure using `secret_list_fields`:

```json
// Request
{"key": "db/prod"}

// Response
{
  "key": "db/prod",
  "fields": ["host", "port", "username", "password", "database"],
  "sensitive_fields": ["password"],
  "bindings": {
    "PGHOST": "host",
    "PGPORT": "port",
    "PGUSER": "username",
    "PGPASSWORD": "password",
    "PGDATABASE": "database"
  }
}
```

### Accessing Non-Sensitive Fields

AI agents can read non-sensitive field values via `secret_get_field`:

```json
// Request
{"key": "db/prod", "field": "host"}

// Response
{"key": "db/prod", "field": "host", "value": "db.example.com"}

// Request for sensitive field (blocked)
{"key": "db/prod", "field": "password"}

// Response
{"error": "field 'password' is marked as sensitive"}
```

### Running with Bindings

Use `secret_run_with_bindings` to execute commands with environment variables:

```json
// Request
{
  "key": "db/prod",
  "command": ["psql", "-c", "SELECT 1"]
}

// Response
{"exit_code": 0, "stdout": "...", "stderr": ""}
```

---

## Best Practices

### 1. Use Consistent Naming

Adopt a naming convention across your secrets:

```
service/environment/purpose
├── db/prod/main
├── db/staging/main
├── aws/prod/deploy
└── stripe/prod/webhook
```

### 2. Mark Sensitive Fields

Always mark password-like fields as sensitive:

```bash
--sensitive password
--sensitive api_key
--sensitive private_key
--sensitive secret
--sensitive token
```

### 3. Use Bindings for Automation

Define bindings when creating secrets for seamless `secret_run` integration:

```bash
secretctl run -k db/prod -- psql  # Works with pre-defined bindings
```

### 4. Document Custom Fields

For custom field structures, add notes:

```bash
secretctl set custom/service \
  --field token=xxx \
  --field endpoint=https://api.example.com \
  --sensitive token \
  --notes "Custom service: token=auth, endpoint=API URL"
```

---

## See Also

- [CLI Commands](/docs/reference/cli-commands) - Full CLI reference
- [MCP Tools](/docs/reference/mcp-tools) - MCP tool documentation
- [Desktop App Guide](/docs/guides/desktop) - Desktop app overview
