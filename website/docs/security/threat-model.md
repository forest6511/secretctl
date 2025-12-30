---
title: Threat Model
description: Visual threat model showing what secretctl protects against.
sidebar_position: 4
---

# Threat Model

This document provides a visual overview of secretctl's threat model, showing what attacks are mitigated and what is explicitly out of scope.

## Visual Threat Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        secretctl THREAT MODEL v1                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ATTACKERS                         DEFENSES                    ASSETS       │
│  ═════════                         ════════                    ══════       │
│                                                                             │
│  ┌─────────────┐                                           ┌─────────────┐ │
│  │ L1: Network │─────────┐                                 │   Secrets   │ │
│  │   Observer  │         │                                 │  (API Keys, │ │
│  │             │         ▼                                 │  Passwords) │ │
│  │ • Passive   │    ┌──────────┐                           └──────┬──────┘ │
│  │   sniffing  │    │localhost │    ┌───────────────────┐        │        │
│  │ • MITM      │    │  only    │───▶│ NO NETWORK ACCESS │        │        │
│  └─────────────┘    └──────────┘    └───────────────────┘        │        │
│                          ✓ PROTECTED                              │        │
│                                                                   │        │
│  ┌─────────────┐                                                  ▼        │
│  │ L2: Local   │─────────┐                           ┌───────────────────┐ │
│  │  Malicious  │         │                           │   ENCRYPTED VAULT │ │
│  │    App      │         ▼                           │                   │ │
│  │             │    ┌──────────────┐                 │  ┌─────────────┐  │ │
│  │ • File read │    │ AES-256-GCM  │                 │  │ vault.db    │  │ │
│  │ • Vault     │    │ + Argon2id   │                 │  │ (0600 perm) │  │ │
│  │   theft     │    │ (64MB/3iter) │                 │  └─────────────┘  │ │
│  └─────────────┘    └──────────────┘                 │         │         │ │
│                          ✓ PROTECTED                  │         ▼         │ │
│                                                      │  ┌─────────────┐  │ │
│  ┌─────────────┐                                     │  │ DEK         │  │ │
│  │ L3: Root    │                                     │  │ (encrypted) │  │ │
│  │  Attacker   │         ✗ OUT OF SCOPE             │  └─────────────┘  │ │
│  │             │                                     │         │         │ │
│  │ • Memory    │    Industry standard exclusion:     │         ▼         │ │
│  │   dump      │    Same as Vault, 1Password,        │  ┌─────────────┐  │ │
│  │ • Root      │    Infisical                        │  │ Master Key  │  │ │
│  │   access    │                                     │  │ (memory)    │  │ │
│  └─────────────┘                                     │  └─────────────┘  │ │
│                                                      └───────────────────┘ │
│                                                                             │
│  ┌─────────────┐         AI-SAFE ACCESS                                    │
│  │ AI Agent    │─────────┐                                                 │
│  │ (via MCP)   │         │                                                 │
│  │             │         ▼                                                 │
│  │ • Prompt    │    ┌──────────────────────────────────────────────┐       │
│  │   injection │    │            AI-SAFE ACCESS                     │       │
│  │ • Context   │    │                                               │       │
│  │   leakage   │    │  ┌────────────┐    ┌────────────────────┐   │       │
│  │ • Training  │    │  │ MCP Server │───▶│ NEVER returns      │   │       │
│  │   data leak │    │  │            │    │ plaintext secrets  │   │       │
│  └─────────────┘    │  └────────────┘    └────────────────────┘   │       │
│                     │                                              │       │
│       ✓ PROTECTED   │  Available to AI:          NOT available:   │       │
│                     │  • secret_list (names)     • secret_get ❌    │       │
│                     │  • secret_exists           • secret_set ❌    │       │
│                     │  • secret_get_masked       • secret_delete ❌ │       │
│                     │  • secret_run (inject)     • export ❌        │       │
│                     │  • secret_list_fields                        │       │
│                     │  • secret_get_field (non-sensitive only)     │       │
│                     │  • secret_run_with_bindings                  │       │
│                     └──────────────────────────────────────────────┘       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Attack Level Summary

| Level | Attacker | Capabilities | Protected? | How |
|-------|----------|--------------|------------|-----|
| **L1** | Network observer | Sniffing, MITM | **Yes** | localhost only, no network exposure |
| **L2** | Malicious app | File read, vault theft | **Yes** | AES-256-GCM + Argon2id encryption |
| **L3** | Root attacker | Memory dump, full access | **No** | Industry standard exclusion |
| **AI** | AI agent via MCP | Prompt injection, leakage | **Yes** | AI-Safe Access (no plaintext) |

## Defense Matrix

```
┌────────────────────────┬──────────────────────────────────────────────────┐
│ THREAT                 │ COUNTERMEASURES                                  │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Network interception   │ ✓ localhost-only MCP (no TCP)                   │
│                        │ ✓ stdio transport for AI communication          │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Vault file theft       │ ✓ AES-256-GCM authenticated encryption          │
│                        │ ✓ Argon2id key derivation (64MB, 3 iter)        │
│                        │ ✓ Per-secret random nonces                      │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Password brute-force   │ ✓ Argon2id memory-hard (64MB per attempt)       │
│                        │ ✓ OWASP 2025 compliant parameters               │
│                        │ ✓ 128-bit salt per vault                        │
├────────────────────────┼──────────────────────────────────────────────────┤
│ AI plaintext exposure  │ ✓ AI-Safe Access: no secret_get                 │
│                        │ ✓ secret_run injects without AI visibility      │
│                        │ ✓ Output sanitization for leaked values         │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Prompt injection       │ ✓ No secrets in AI context = nothing to steal   │
│                        │ ✓ Field sensitivity enforced (sensitive fields blocked) │
│                        │ ✓ Blocked commands (env, printenv, etc.)        │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Audit tampering        │ ✓ HMAC chain integrity verification             │
│                        │ ✓ Tamper detection via prev_hash mismatch       │
│                        │ ✓ secretctl audit verify command                │
├────────────────────────┼──────────────────────────────────────────────────┤
│ Weak file permissions  │ ✓ 0600 for vault.db, vault.salt                 │
│                        │ ✓ 0700 for vault directory                      │
│                        │ ✓ Permission check on startup                   │
└────────────────────────┴──────────────────────────────────────────────────┘
```

## AI-Safe Access Flow

This diagram shows how secrets flow through the system without ever reaching AI context:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AI-SAFE ACCESS DATA FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────┐                                                          │
│   │   AI Agent   │  "Run 'aws s3 ls' with my AWS credentials"              │
│   │  (Claude)    │                                                          │
│   └──────┬───────┘                                                          │
│          │                                                                  │
│          ▼                                                                  │
│   ┌──────────────┐                                                          │
│   │  MCP Server  │  Receives: keys=["aws/*"], command="aws s3 ls"          │
│   │  (secretctl) │  AI sees: key names only                                │
│   └──────┬───────┘                                                          │
│          │                                                                  │
│          ▼                                                                  │
│   ┌──────────────────────────────────────────────────────────┐              │
│   │                    SECURITY BOUNDARY                      │              │
│   │  ┌─────────────────────────────────────────────────────┐ │              │
│   │  │ 1. Decrypt vault with master key                    │ │              │
│   │  │ 2. Retrieve matching secrets                        │ │              │
│   │  │ 3. Set as environment variables in subprocess       │ │              │
│   │  │    AWS_ACCESS_KEY=AKIA... (never sent to AI)        │ │              │
│   │  │    AWS_SECRET_KEY=wJal... (never sent to AI)        │ │              │
│   │  │ 4. Execute command: aws s3 ls                       │ │              │
│   │  │ 5. Capture and sanitize output                      │ │              │
│   │  └─────────────────────────────────────────────────────┘ │              │
│   └──────────────────────────────────────────────────────────┘              │
│          │                                                                  │
│          ▼                                                                  │
│   ┌──────────────┐                                                          │
│   │  MCP Server  │  Returns: {"stdout": "bucket1\nbucket2", "exit_code": 0}│
│   │  (response)  │  AI sees: command output only, sanitized               │
│   └──────┬───────┘                                                          │
│          │                                                                  │
│          ▼                                                                  │
│   ┌──────────────┐                                                          │
│   │   AI Agent   │  "Your S3 buckets are: bucket1, bucket2"                │
│   │  (response)  │  AI used secrets without ever seeing them               │
│   └──────────────┘                                                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Out of Scope (Industry Standard)

The following threats are **explicitly excluded** from secretctl's threat model, consistent with industry leaders like HashiCorp Vault, 1Password, and Infisical:

### Root-Level Compromise
- If an attacker has root access, they can read process memory, install keyloggers, or modify binaries
- No user-space application can defend against a compromised kernel
- **Mitigation**: Secure your system with standard practices (updates, access control, monitoring)

### Memory Dump Attacks
- Go's garbage collector manages memory, making guaranteed secret zeroing difficult
- Secrets may persist in memory until garbage collected
- **Mitigation**: secretctl minimizes secret lifetime in memory; use encrypted swap

### Weak Master Passwords
- Users who choose weak passwords can be brute-forced
- Argon2id slows attacks but cannot prevent a determined attacker with a weak password
- **Mitigation**: Choose a strong master password (12+ characters recommended)

### Physical Device Access
- If an attacker has physical access to an unlocked device, they can access secrets
- **Mitigation**: Use device encryption, auto-lock, and physical security

## Comparison with Industry

| Product | Network Protection | Encryption at Rest | AI Plaintext Protection | Audit Logs |
|---------|-------------------|-------------------|------------------------|------------|
| HashiCorp Vault | mTLS | AES-256 | No (exposes via MCP) | Yes |
| 1Password | TLS | AES-256-GCM | Yes (refuses MCP) | Yes |
| Infisical | TLS | AES-256 | No (exposes via MCP) | Yes |
| **secretctl** | **localhost only** | **AES-256-GCM** | **Yes (AI-Safe Access)** | **Yes** |

secretctl uniquely combines:
- **MCP support** for AI integration
- **AI-Safe Access** preventing plaintext exposure
- **Local-first** architecture with no cloud dependency

## Security Reporting

Found a security issue? Please report responsibly:

- **Email**: secretctl.oss@gmail.com
- **Do NOT** open public GitHub issues for security vulnerabilities
- Allow reasonable time for fixes before public disclosure

See [SECURITY.md](https://github.com/forest6511/secretctl/blob/main/SECURITY.md) for full policy.

## Related Documentation

- [Security Overview](/docs/security/) - Core security principles
- [How It Works](/docs/security/how-it-works) - Detailed architecture
- [Encryption Details](/docs/security/encryption) - Cryptographic specifications
- [MCP Tools Reference](/docs/reference/mcp-tools) - AI integration security
