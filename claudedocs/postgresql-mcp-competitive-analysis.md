# PostgreSQL MCP Competitive Analysis

## Executive Summary

This analysis examines how competing MCP servers handle database credentials, with focus on PostgreSQL implementations. The goal is to identify best practices for integrating secretctl with database MCP servers while maintaining AI-Safe Access security principles.

**Key Finding**: Most PostgreSQL MCP servers have significant security weaknesses. secretctl's AI-Safe Access design can provide superior security through credential injection without AI exposure.

---

## Industry Statistics (2025)

Source: [Astrix Security Research - State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)

| Metric | Value | Risk Level |
|--------|-------|------------|
| MCP servers requiring credentials | 88% | - |
| Using static API keys/PATs | 53% | HIGH |
| Credentials via environment variables | 79% | MEDIUM |
| Using OAuth (modern delegation) | 8.5% | LOW |

**Critical Issue**: Few MCP clients store server configuration securely. Credentials passed via MCP tools are stored in chat history.

---

## PostgreSQL MCP Server Implementations

### 1. Official Anthropic PostgreSQL MCP
**Source**: [modelcontextprotocol/servers](https://github.com/modelcontextprotocol/servers)

| Aspect | Implementation |
|--------|----------------|
| Credential Storage | Connection string in config |
| Security Model | Basic environment variables |
| AI Exposure | Connection string visible in config |

**Weakness**: No native secrets management integration.

---

### 2. Postgres MCP Pro (crystaldba)
**Source**: [crystaldba/postgres-mcp](https://github.com/crystaldba/postgres-mcp)

| Aspect | Implementation |
|--------|----------------|
| Credential Storage | Config or environment variables |
| Security Features | Read/write access control |
| AI Exposure | Credentials may pass through LLM |

**Weakness**: Credentials stored in chat history when passed through LLM.

---

### 3. hthuong09/postgres-mcp
**Source**: [hthuong09/postgres-mcp](https://github.com/hthuong09/postgres-mcp)

| Aspect | Implementation |
|--------|----------------|
| Credential Storage | Environment variables or .env files |
| Security Features | Passwords stripped from resource URIs |
| AI Exposure | Reduced but not eliminated |

**Strength**: Passwords automatically removed from URIs.
**Weakness**: Still relies on environment variable storage.

---

### 4. HenkDz/postgresql-mcp-server
**Source**: [HenkDz/postgresql-mcp-server](https://github.com/HenkDz/postgresql-mcp-server)

| Aspect | Implementation |
|--------|----------------|
| Credential Storage | CLI args, env vars, or per-tool config |
| Security Features | SQL injection prevention, parameterized queries |
| AI Exposure | Flexible but still exposed |

**Strength**: SQL injection prevention.
**Weakness**: Multiple credential paths increase attack surface.

---

### 5. 1Password-enabled MySQL MCP Server
**Source**: [chrisdail/mcp-mysql-1password-server](https://lobehub.com/mcp/chrisdail-mcp-mysql-1password-server)

| Aspect | Implementation |
|--------|----------------|
| Credential Storage | 1Password vault |
| Security Features | AI never sees passwords |
| AI Exposure | **NONE** - credentials injected at runtime |

**Strength**: Best-in-class security model. Credentials stored in 1Password, injected at runtime, AI model never sees passwords.

---

## Best Practices Summary

Source: [WorkOS Guide - Best Practices for MCP Secrets Management](https://workos.com/guide/best-practices-for-mcp-secrets-management)

### 1. Ephemeral/Dynamic Credentials
```
Create temporary credentials valid for 5 minutes
Execute query with limited permissions
Automatically terminate access after use
```

### 2. Per-Tool-Call Connections
```
Create connection for each tool call (not on server start)
Allows tool listing even if not configured
Trades latency for security and reliability
```

### 3. Least Privilege Access
```
Fine-grained roles per MCP integration
Avoid "root" tokens
Read-only permissions where possible
```

### 4. Enterprise Secret Management
```
HashiCorp Vault, AWS Secrets Manager, Azure Key Vault
Strict access controls and audit logging
Encryption at rest and in transit
```

---

## 1Password's Approach

Source: [1Password Blog - Securing MCP Servers](https://1password.com/blog/securing-mcp-servers-with-1password-stop-credential-exposure-in-your-agent)

### `op run` Credential Injection

```bash
# Instead of hardcoding in config:
{
  "env": {
    "POSTGRES_URL": "postgres://user:password@host/db"  # BAD
  }
}

# Use 1Password reference:
{
  "env": {
    "POSTGRES_URL": "op://vault/item/field"  # GOOD
  }
}
```

**How it works**:
1. `op run` resolves `op://` references
2. Decrypts secrets in memory
3. Sets as environment variables for that process
4. Secrets disappear when process exits

### Security Principles
- Credentials should NOT be exchanged over non-deterministic AI channels
- Credentials should be injected on behalf of agent, without handing them over
- Human users should authorize access to sensitive data

---

## secretctl Integration Design Options

### Option A: secretctl as Credential Injector for Existing PostgreSQL MCP

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐
│ Claude Code │────▶│  secretctl  │────▶│ postgres-mcp │
│   (AI)      │     │   run       │     │   (Query)    │
└─────────────┘     └─────────────┘     └──────────────┘
                          │
                    Inject env vars
                    (AI never sees)
```

**Implementation**:
```bash
# secretctl stores PostgreSQL credentials
secretctl set POSTGRES_URL

# Launch postgres-mcp with injected credentials
secretctl run -k "POSTGRES_*" -- npx @anthropic/postgres-mcp
```

**Pros**: Uses existing postgres-mcp, minimal development
**Cons**: Requires process wrapping, two-tool setup

---

### Option B: secretctl with Database Connection Tools

```
┌─────────────┐     ┌─────────────────────────────────┐
│ Claude Code │────▶│          secretctl MCP          │
│   (AI)      │     │  ┌───────────┐ ┌─────────────┐  │
└─────────────┘     │  │ secret_*  │ │ db_query    │  │
                    │  │ (vault)   │ │ db_execute  │  │
                    │  └───────────┘ └─────────────┘  │
                    └─────────────────────────────────┘
                              │
                         Internal credential
                         resolution (no exposure)
```

**New MCP Tools**:
- `db_query(connection_name, sql)` - Execute SELECT
- `db_execute(connection_name, sql)` - Execute INSERT/UPDATE/DELETE
- `db_list_connections()` - List configured database connections

**Pros**: Single tool, integrated experience
**Cons**: Significant development, scope expansion

---

### Option C: secretctl + Dedicated DB MCP (Recommended)

```
┌─────────────┐
│ Claude Code │
│   (AI)      │
└──────┬──────┘
       │
       ├────────────────────────────────────┐
       │                                    │
       ▼                                    ▼
┌─────────────────┐              ┌─────────────────┐
│  secretctl MCP  │              │  postgres-mcp   │
│  (credentials)  │──inject──▶   │  (operations)   │
└─────────────────┘              └─────────────────┘
       │
  AI-Safe Access
  (AI never sees
   plaintext)
```

**Workflow**:
1. secretctl stores database credentials securely
2. User configures postgres-mcp to read from secretctl
3. AI uses postgres-mcp for queries
4. Credentials flow through environment injection, never through AI

**Implementation**: Create launcher script or config pattern

---

## Recommendation

### For secretctl v1.x (Short-term)

**Approach**: Document the `secret_run` pattern for database MCP servers

```bash
# Store credentials
echo "postgres://user:pass@host:5432/db" | secretctl set db/postgres/dev

# Launch postgres-mcp with injected credentials
secretctl run -k "db/postgres/dev" -e POSTGRES_URL -- npx @anthropic/postgres-mcp
```

**Deliverables**:
- Documentation: "Using secretctl with Database MCP Servers"
- Example configurations for common setups
- Best practices guide

### For secretctl v2.x (Long-term)

**Approach**: Native integration with popular database MCP servers

Potential features:
- `secretctl mcp-launch postgres` - Launcher command with auto-injection
- Connection profile management (`secretctl db add`, `secretctl db list`)
- Integration with 1Password-style `op://` reference syntax

---

## Security Comparison Matrix

| Feature | Raw Config | Env Vars | 1Password | secretctl |
|---------|-----------|----------|-----------|-----------|
| Credential Storage | Plaintext | Plaintext | Encrypted | Encrypted |
| AI Exposure | YES | Possible | NO | NO |
| Audit Trail | NO | NO | YES | YES |
| Auto-Rotation | NO | NO | Manual | Manual |
| Ephemeral Creds | NO | NO | NO | Possible |
| MCP Integration | Native | Native | `op run` | `secret_run` |

---

## Conclusion

secretctl's AI-Safe Access design aligns with industry best practices (1Password, HashiCorp Vault) for AI-era credential management. The key insight from competitive analysis:

> **Credentials should be injected on behalf of the agent, without handing them over.**

secretctl can achieve this through the existing `secret_run` command, positioning it as the "1Password for developers" in the MCP ecosystem.

---

## Sources

- [Astrix Security - State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [WorkOS - Best Practices for MCP Secrets Management](https://workos.com/guide/best-practices-for-mcp-secrets-management)
- [1Password - Securing MCP Servers](https://1password.com/blog/securing-mcp-servers-with-1password-stop-credential-exposure-in-your-agent)
- [1Password - Where MCP Fits and Where It Doesn't](https://1password.com/blog/where-mcp-fits-and-where-it-doesnt)
- [Infisical - Managing Secrets in MCP Servers](https://infisical.com/blog/managing-secrets-mcp-servers)
- [Docker - Top 5 MCP Server Best Practices](https://www.docker.com/blog/mcp-server-best-practices/)
- [crystaldba/postgres-mcp](https://github.com/crystaldba/postgres-mcp)
- [hthuong09/postgres-mcp](https://github.com/hthuong09/postgres-mcp)
- [HenkDz/postgresql-mcp-server](https://github.com/HenkDz/postgresql-mcp-server)
