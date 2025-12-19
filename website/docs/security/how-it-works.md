---
title: How It Works
description: Security architecture and data flow in secretctl.
sidebar_position: 2
---

# How Security Works

This guide explains the security architecture of secretctl, including how secrets are protected, how different access methods work, and the multi-layer defense model.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        secretctl Architecture                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   Human Operators              AI Agents                            │
│   ═══════════════              ═════════                            │
│   CLI / Desktop App            MCP Server                           │
│        │                            │                               │
│        │ Full Access                │ Restricted (Option D+)        │
│        │                            │                               │
│        └──────────┬────────────────┘                               │
│                   │                                                 │
│                   ▼                                                 │
│            ┌─────────────┐                                          │
│            │   Vault     │                                          │
│            │  (SQLite)   │                                          │
│            │             │                                          │
│            │ AES-256-GCM │                                          │
│            │  Encrypted  │                                          │
│            └─────────────┘                                          │
│                   │                                                 │
│                   ▼                                                 │
│            ┌─────────────┐                                          │
│            │ Audit Log   │                                          │
│            │ HMAC Chain  │                                          │
│            └─────────────┘                                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Key Hierarchy

secretctl uses a three-tier key hierarchy for defense in depth:

```
┌─────────────────────────────────────────────────────────┐
│                    User Input                           │
│                  Master Password                        │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼ Argon2id (memory: 64MB, time: 3, threads: 4)
┌─────────────────────────────────────────────────────────┐
│                 Master Key (256-bit)                    │
│              ※ Memory only, never stored               │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼ AES-256-GCM encryption
┌─────────────────────────────────────────────────────────┐
│           Data Encryption Key (DEK) (256-bit)           │
│              ※ Stored encrypted                        │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼ AES-256-GCM encryption
┌─────────────────────────────────────────────────────────┐
│                Encrypted Secrets                        │
│              ※ Stored in SQLite                        │
└─────────────────────────────────────────────────────────┘
```

### Why This Design?

| Layer | Purpose |
|-------|---------|
| Master Password | User authentication, never stored |
| Master Key | Derived fresh each unlock, protects DEK |
| DEK | Actual encryption key, allows password rotation |
| Encrypted Secrets | Protected data at rest |

## Vault Structure

The vault directory contains:

```
~/.secretctl/
├── vault.db          # SQLite DB (values encrypted by app)
├── vault.salt        # Argon2 salt (128-bit)
└── vault.meta        # Metadata (version, created_at)
```

### File Permissions

All sensitive files use strict permissions:

| File | Permission | Reason |
|------|------------|--------|
| vault.db | 0600 | Contains encrypted secrets |
| vault.salt | 0600 | Required for key derivation |
| vault.meta | 0600 | Metadata protection |
| ~/.secretctl/ | 0700 | Directory access control |

## Access Control Model

### CLI vs MCP: Trust Levels

secretctl restricts functionality based on access method:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    secretctl Access Model                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   CLI (Human Operator)                MCP Server (AI Agent)         │
│   ════════════════════                ═════════════════════         │
│   Trust Level: FULL                   Trust Level: RESTRICTED       │
│                                                                     │
│   ✅ secretctl get KEY                ❌ secret_get (not implemented)│
│      → Plaintext OK                      → No plaintext to AI       │
│                                                                     │
│   ✅ secretctl set KEY                ❌ secret_set (not implemented)│
│      → Store via stdin                   → AI cannot set values     │
│                                                                     │
│   ✅ secretctl list                   ✅ secret_list                │
│      → Full key list                     → Key names only           │
│                                                                     │
│   ✅ secretctl run -- cmd             ✅ secret_run                 │
│      → Environment injection             → Masked output only       │
│                                                                     │
│   ✅ secretctl delete KEY             ❌ (not implemented)           │
│      → Delete OK                         → AI cannot delete         │
│                                                                     │
│   ✅ secretctl export                 ❌ (not implemented)           │
│      → Export OK                         → AI cannot export         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Design Rationale

| Access | Operator | Trust | Restrictions |
|--------|----------|-------|--------------|
| **CLI** | Human directly | Full | None (all features) |
| **Desktop** | Human directly | Full | None (all features) |
| **MCP** | AI agent | Restricted | No plaintext get/set/delete |

**Why the difference?**

- CLI users are humans who **own their secrets** and have the right to see them
- AI agents are **proxies** that don't need to know the actual values
- AI only needs the **result** of using secrets, not the secrets themselves

## Option D+: Access Without Exposure

The core security model for AI integration:

```
┌─────────────────────────────────────────────────────────────────────┐
│  Option D+: Access Without Exposure                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ❌ Traditional MCP Implementation:                                  │
│     AI → MCP Server → Plaintext Secret → AI → Use                   │
│     Problem: AI "knows" the secret                                  │
│                                                                     │
│  ✅ secretctl Option D+:                                            │
│     AI → "Run this command with this secret"                        │
│        → Secret injected directly (AI never sees it)                │
│        → Only the result returns to AI                              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### MCP Tool Capabilities

| Capability | Description | AI Response |
|------------|-------------|-------------|
| `reference_only` | Key existence check | `{"exists": true, "created_at": "..."}` |
| `env_inject` | Environment injection | `{"exit_code": 0, "stdout": "..."}` |
| `masked_return` | Masked value display | `{"masked_value": "sk-****7890"}` |

**Note**: `full` (plaintext return) is **abolished**. No MCP tool returns plaintext values.

## secret_run Security Layers

The `secret_run` command uses multi-layer defense:

```
AI → secret_run(keys, command)
        │
        ▼
┌─────────────────────────────────────────────────┐
│ Layer 1: Command Validation                      │
│   - Blocked commands list (env, printenv, etc.) │
│   - Additional AI restrictions                  │
├─────────────────────────────────────────────────┤
│ Layer 2: Execution Environment                   │
│   - Non-TTY mode forced                         │
│   - Temporary working directory                 │
│   - Timeout: 300 seconds                        │
├─────────────────────────────────────────────────┤
│ Layer 3: Environment Injection                   │
│   - Secrets exist only in process environment   │
│   - Values never passed to AI                   │
├─────────────────────────────────────────────────┤
│ Layer 4: Output Sanitization                     │
│   - Scan stdout/stderr for secrets              │
│   - Mask any detected values                    │
│   - Base64 encoding detection                   │
└─────────────────────────────────────────────────┘
        │
        ▼
AI ← {"exit_code": 0, "stdout": "...", "stderr": "..."}
     ※ Secret values never included
```

### Blocked Commands

Commands that could expose environment variables:

```go
var blockedCommands = []string{
    "env",
    "printenv",
    "set",
    "export",
    "declare",
    "cat /proc/*/environ",
}
```

### Output Sanitization

```go
// OutputSanitizer masks secrets in stdout/stderr
type OutputSanitizer struct {
    secrets map[string]string // key -> value
}

func (s *OutputSanitizer) Sanitize(output string) string {
    result := output
    for key, value := range s.secrets {
        // Plaintext detection
        if strings.Contains(result, value) {
            result = strings.ReplaceAll(result, value, "[REDACTED:"+key+"]")
        }
        // Base64 encoding detection
        encoded := base64.StdEncoding.EncodeToString([]byte(value))
        if strings.Contains(result, encoded) {
            result = strings.ReplaceAll(result, encoded, "[REDACTED:"+key+":base64]")
        }
    }
    return result
}
```

### Masking Format

Values are masked with fixed-length prefix for security:

```go
func maskValue(value string) string {
    if len(value) <= 8 {
        return "********"
    }
    // Show only last 4 characters
    return "********" + value[len(value)-4:]
}

// Example: "sk-proj-abc123xyz789" → "********z789"
```

Fixed-length masking prevents value length inference.

## Audit Logging

### HMAC Chain Integrity

```
┌─────────────────────────────────────────────────────────────────────┐
│  Audit Log HMAC Chain                                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Record 1          Record 2          Record 3                       │
│  ┌─────────┐       ┌─────────┐       ┌─────────┐                   │
│  │ data    │       │ data    │       │ data    │                   │
│  │ prev: Ø │──┐    │ prev: H1│──┐    │ prev: H2│                   │
│  │ hash: H1│  │    │ hash: H2│  │    │ hash: H3│                   │
│  └─────────┘  │    └─────────┘  │    └─────────┘                   │
│               │         ▲       │         ▲                        │
│               └─────────┘       └─────────┘                        │
│                                                                     │
│  H = HMAC-SHA256(id || action || key_hash || ... || prev, key)     │
│  key = HKDF-SHA256(master_key, "audit-log-v1")                     │
│                                                                     │
│  Tampering Detection: prev_hash mismatch → chain broken            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### What's Logged

**Recorded:**
- Operation type (get, set, delete, list, run)
- Key hash (not the key name itself for sensitive operations)
- Source (cli, mcp, ui)
- Timestamp (nanosecond precision)
- Result (success, denied, error)

**NOT Recorded (for security):**
- Secret values (plaintext or encrypted)
- Master password
- Raw user input

### Verification

```bash
# Verify audit log integrity
$ secretctl audit verify
✓ 15,234 records verified
✓ Chain integrity: OK
✓ No gaps detected

# If tampering detected
$ secretctl audit verify
✗ Chain broken at record id=01HQ5E7N8K...
  Expected prev: a3f2b1...
  Actual prev:   7c8d4e...
  ALERT: Possible tampering detected
```

## MCP Policy Engine

### Policy Configuration

```yaml
# ~/.secretctl/mcp-policy.yaml
policies:
  - name: "claude-desktop"
    agent_id: "claude-desktop-*"
    allowed_keys:
      - "api/*"           # Allow api/ prefix
      - "config/dev/*"
    denied_keys:
      - "*/prod/*"        # Deny prod secrets
    capabilities:
      - env_inject        # Environment injection
      - reference_only    # Existence check
      - masked_return     # Masked display

  - name: "untrusted-agent"
    agent_id: "*"
    allowed_keys: []      # Deny all
    capabilities:
      - reference_only    # Only check existence
```

### Policy Evaluation Order

1. Check `denied_keys` first (deny takes precedence)
2. Check `allowed_keys` for permission
3. Verify required `capabilities`
4. Log access attempt to audit

## Security Benefits

1. **No AI Plaintext Exposure**
   - Most important protection
   - Values never in AI model logs or training data

2. **Prompt Injection Resistance**
   - No secret values to extract via malicious prompts
   - Even if AI is compromised, secrets remain protected

3. **Multi-Layer Defense**
   - Command validation + environment isolation + output filtering
   - Defense in depth approach

4. **Full Auditability**
   - Every command logged with tamper detection
   - Alerts when sanitization triggers

5. **Least Privilege**
   - AI only gets minimum necessary information
   - Key names and masked values only

## Next Steps

- [Encryption Details](/docs/security/encryption) - Cryptographic specifications
- [MCP Tools Reference](/docs/reference/mcp-tools) - Complete MCP tool documentation
- [AI Agent Integration](/docs/use-cases/ai-agent-integration) - Practical MCP setup
