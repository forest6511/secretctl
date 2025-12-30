---
title: Architecture Overview
description: System architecture and component design of secretctl.
sidebar_position: 1
---

# Architecture Overview

This document describes the high-level architecture of secretctl, including component relationships, data flows, and security boundaries.

## System Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           secretctl Architecture                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐                       │
│  │    CLI      │   │  Desktop    │   │ MCP Server  │    User Interfaces    │
│  │  (Cobra)    │   │   (Wails)   │   │   (stdio)   │                       │
│  └──────┬──────┘   └──────┬──────┘   └──────┬──────┘                       │
│         │                 │                 │                               │
│         │    Full Trust   │    Full Trust   │   Restricted (AI-Safe Access)│
│         │                 │                 │                               │
│         └────────────────┼────────────────┘                               │
│                          │                                                  │
│                          ▼                                                  │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                         Vault Core (pkg/vault)                         │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │ │
│  │  │   Manager   │  │  Secrets    │  │   Audit     │  │   Policy    │  │ │
│  │  │             │──│   Store     │──│    Log      │  │   Engine    │  │ │
│  │  │ (lifecycle) │  │  (CRUD)     │  │  (HMAC)     │  │  (MCP only) │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                          │                                                  │
│                          ▼                                                  │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                       Crypto Layer (pkg/crypto)                        │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                   │ │
│  │  │   Argon2id  │  │  AES-256    │  │  HMAC-256   │                   │ │
│  │  │    (KDF)    │  │    (GCM)    │  │   (Chain)   │                   │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                   │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                          │                                                  │
│                          ▼                                                  │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Storage Layer (SQLite)                          │ │
│  │  ┌───────────────────────────────────────────────────────────────┐   │ │
│  │  │  vault.db (0600)  │  vault.salt (0600)  │  vault.meta (0600)  │   │ │
│  │  └───────────────────────────────────────────────────────────────┘   │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | Package |
|-----------|----------------|---------|
| **CLI** | Command-line interface, user interaction | `cmd/secretctl` |
| **Desktop** | Native GUI application | `desktop/` |
| **MCP Server** | AI agent integration, AI-Safe Access enforcement | `internal/mcp` |
| **Vault Manager** | Lifecycle (init, lock, unlock), session management | `pkg/vault` |
| **Secrets Store** | CRUD operations, metadata handling | `pkg/vault` |
| **Audit Log** | HMAC-chained event logging, integrity verification | `pkg/vault` |
| **Policy Engine** | MCP command allowlist, deny-by-default | `internal/mcp` |
| **Crypto Layer** | Key derivation, encryption, HMAC | `pkg/crypto` |
| **Storage Layer** | SQLite persistence, file permissions | `pkg/vault` |

## Data Flow Diagrams

### Secret Set Operation

```
User                    CLI                     Vault                   Crypto
  │                      │                        │                       │
  │  set KEY (stdin)     │                        │                       │
  │─────────────────────▶│                        │                       │
  │                      │  Store(key, value)     │                       │
  │                      │───────────────────────▶│                       │
  │                      │                        │  Encrypt(DEK, value)  │
  │                      │                        │──────────────────────▶│
  │                      │                        │                       │
  │                      │                        │◀──────────────────────│
  │                      │                        │      ciphertext       │
  │                      │                        │                       │
  │                      │                        │  INSERT INTO secrets  │
  │                      │                        │──────────▶ [SQLite]   │
  │                      │                        │                       │
  │                      │                        │  Log(SET, key_hash)   │
  │                      │                        │──────────▶ [Audit]    │
  │                      │◀───────────────────────│                       │
  │◀─────────────────────│      success           │                       │
  │       OK             │                        │                       │
```

### MCP secret_run Operation (AI-Safe Access)

```
AI Agent            MCP Server              Vault              Subprocess
    │                   │                     │                    │
    │ secret_run        │                     │                    │
    │ (keys, cmd)       │                     │                    │
    │──────────────────▶│                     │                    │
    │                   │                     │                    │
    │                   │  ValidatePolicy(cmd)│                    │
    │                   │────────────────────▶│                    │
    │                   │◀────────────────────│                    │
    │                   │     allowed         │                    │
    │                   │                     │                    │
    │                   │  GetSecrets(keys)   │                    │
    │                   │────────────────────▶│                    │
    │                   │◀────────────────────│                    │
    │                   │     values          │                    │
    │                   │                     │                    │
    │                   │  Exec(cmd, env=secrets)                  │
    │                   │─────────────────────────────────────────▶│
    │                   │                                          │
    │                   │◀─────────────────────────────────────────│
    │                   │     stdout, stderr                       │
    │                   │                     │                    │
    │                   │  Sanitize(output, secrets)               │
    │                   │     [REDACTED] replacement               │
    │                   │                     │                    │
    │◀──────────────────│                     │                    │
    │  sanitized output │                     │                    │
    │  (NO plaintext)   │                     │                    │
```

## Threat Model

### Trust Boundaries

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Trust Boundary 1                                │
│                        (User's Local Environment)                           │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                         │ │
│  │   TRUSTED ACTORS                    TRUSTED OPERATIONS                 │ │
│  │   ──────────────                    ──────────────────                 │ │
│  │   • Human user (CLI)                • Read plaintext secrets          │ │
│  │   • Human user (Desktop)            • Write/modify secrets            │ │
│  │                                      • Delete secrets                  │ │
│  │                                      • Export vault                    │ │
│  │                                                                         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                    │                                        │
│                                    │ AI-Safe Access Boundary                │
│                                    ▼                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                                                                         │ │
│  │   RESTRICTED ACTORS                 RESTRICTED OPERATIONS              │ │
│  │   ─────────────────                 ─────────────────────              │ │
│  │   • AI agents (MCP)                 • List key names (no values)      │ │
│  │   • Automated tools                 • Check existence                  │ │
│  │                                      • Get masked values               │ │
│  │                                      • Run commands (allowlisted)      │ │
│  │                                                                         │ │
│  │   DENIED OPERATIONS                                                    │ │
│  │   ─────────────────                                                    │ │
│  │   ✗ Read plaintext secrets                                            │ │
│  │   ✗ Write/modify secrets                                              │ │
│  │   ✗ Delete secrets                                                    │ │
│  │   ✗ Export vault                                                      │ │
│  │   ✗ Run blocked commands (env, printenv, etc.)                        │ │
│  │                                                                         │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Threat Categories and Mitigations

| Category | Threat | Mitigation | Implementation |
|----------|--------|------------|----------------|
| **T1: Secret Exposure** | AI extracts plaintext | No `secret_get` tool | `internal/mcp` |
| **T2: Output Leakage** | Command output contains secrets | Output sanitization | `pkg/vault/sanitizer.go` |
| **T3: Prompt Injection** | Malicious prompt requests data | Tool-level restrictions | MCP tool definitions |
| **T4: Command Injection** | Injected shell commands | Command allowlist | `mcp-policy.yaml` |
| **T5: Env Dump** | `env`/`printenv` reveals secrets | Blocked commands list | `internal/mcp/policy.go` |
| **T6: Brute Force** | Master password guessing | Argon2id (memory-hard) | `pkg/crypto` |
| **T7: Data at Rest** | Disk access to vault | AES-256-GCM encryption | `pkg/crypto` |
| **T8: Log Tampering** | Audit log modification | HMAC chain integrity | `pkg/vault/audit.go` |
| **T9: File Permission** | Unauthorized file access | 0600/0700 permissions | `pkg/vault` |

### STRIDE Analysis

| Threat Type | Applicable | Mitigation |
|-------------|------------|------------|
| **S**poofing | No | Local-only, no network auth |
| **T**ampering | Yes | HMAC chain audit logs, encrypted storage |
| **R**epudiation | Yes | Audit logging with timestamps |
| **I**nformation Disclosure | Yes | AI-Safe Access, output sanitization |
| **D**enial of Service | Limited | Timeout on commands (300s) |
| **E**levation of Privilege | Yes | Trust boundary enforcement |

## Key Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Key Hierarchy                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│    User Input: Master Password                                              │
│                      │                                                      │
│                      │  + Salt (128-bit, random)                            │
│                      ▼                                                      │
│    ┌─────────────────────────────────────────────────────────────────────┐ │
│    │  Argon2id                                                            │ │
│    │  ─────────                                                           │ │
│    │  Memory:  64 MiB                                                     │ │
│    │  Time:    3 iterations                                               │ │
│    │  Threads: 4                                                          │ │
│    │  Output:  256 bits                                                   │ │
│    └─────────────────────────────────────────────────────────────────────┘ │
│                      │                                                      │
│                      ▼                                                      │
│    Master Key (256-bit) ──────── NEVER STORED ────────                     │
│                      │                                                      │
│                      │  HKDF-SHA256 ("dek-v1")                              │
│                      ▼                                                      │
│    ┌─────────────────────────────────────────────────────────────────────┐ │
│    │  DEK (Data Encryption Key) - 256-bit                                 │ │
│    │  ─────────────────────────────────────                               │ │
│    │  • Stored encrypted in vault.db                                      │ │
│    │  • Allows password rotation without re-encrypting all secrets        │ │
│    └─────────────────────────────────────────────────────────────────────┘ │
│                      │                                                      │
│                      │  AES-256-GCM                                         │
│                      ▼                                                      │
│    ┌─────────────────────────────────────────────────────────────────────┐ │
│    │  Encrypted Secrets                                                   │ │
│    │  ─────────────────                                                   │ │
│    │  • Each secret encrypted with unique nonce                           │ │
│    │  • Stored as: nonce (12B) || ciphertext || auth_tag (16B)           │ │
│    └─────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│    Audit Log Key (256-bit)                                                  │
│    ────────────────────────                                                │
│    • Derived from Master Key via HKDF-SHA256 ("audit-log-v1")             │
│    • Used for HMAC chain computation                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Database Schema

```sql
-- Secrets table
CREATE TABLE secrets (
    id           TEXT PRIMARY KEY,
    key          TEXT UNIQUE NOT NULL,
    encrypted_value BLOB NOT NULL,
    notes        TEXT,
    url          TEXT,
    tags         TEXT,              -- JSON array
    expires_at   DATETIME,
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

-- Audit log table
CREATE TABLE audit_log (
    id           TEXT PRIMARY KEY,
    action       TEXT NOT NULL,      -- get, set, delete, list, run
    key_hash     TEXT,               -- SHA-256 of key (not plaintext)
    source       TEXT NOT NULL,      -- cli, mcp, ui
    success      BOOLEAN NOT NULL,
    error_msg    TEXT,
    timestamp    DATETIME NOT NULL,
    prev_hash    TEXT,               -- HMAC chain
    hash         TEXT NOT NULL       -- HMAC of this record
);

-- Metadata table
CREATE TABLE metadata (
    key          TEXT PRIMARY KEY,
    value        TEXT NOT NULL
);
```

## File Structure

```
~/.secretctl/
├── vault.db           # SQLite database (encrypted values)
│                      # Permission: 0600
│
├── vault.salt         # Argon2id salt (128 bits)
│                      # Permission: 0600
│
├── vault.meta         # Vault metadata (JSON)
│                      # Permission: 0600
│                      # Contains: version, created_at
│
└── mcp-policy.yaml    # MCP command policy (optional)
                       # Permission: 0600
                       # Validated: No symlinks allowed
```

## Security Boundaries Summary

| Interface | Trust Level | Can Read Plaintext | Can Modify | Can Delete |
|-----------|-------------|-------------------|------------|------------|
| CLI | Full | Yes | Yes | Yes |
| Desktop | Full | Yes | Yes | Yes |
| MCP | Restricted | No | No | No |

## Next Steps

- [Encryption Details](/docs/security/encryption) - Cryptographic specifications
- [MCP Security Model](/docs/guides/mcp/security-model) - AI agent restrictions
- [How Security Works](/docs/security/how-it-works) - Detailed security mechanisms
