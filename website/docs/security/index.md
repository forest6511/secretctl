---
title: Security Overview
description: Security architecture and design principles of secretctl.
sidebar_position: 1
---

# Security Overview

secretctl is designed with security as the foundational principle. This guide explains the security architecture, threat model, and the measures taken to protect your secrets.

## Core Security Principles

### 1. Standard Libraries Only

secretctl uses only Go standard library and `golang.org/x/crypto` for cryptographic operations:

- **No custom cryptography** - All encryption uses battle-tested implementations
- **Minimal dependencies** - Reduces supply chain attack surface
- **Auditable codebase** - Easier security reviews

### 2. Option D+ for AI Integration

The cornerstone of secretctl's AI security model:

> **AI agents NEVER receive plaintext secrets.**

| Access Method | Plaintext Access | Use Case |
|--------------|------------------|----------|
| CLI | Yes | Human operators |
| Desktop App | Yes | Human operators |
| MCP Server | **No** | AI agents |

This aligns with [1Password's "Access Without Exposure"](https://1password.com/blog/security-principles-guiding-1passwords-approach-to-ai) philosophy.

### 3. Local-First Architecture

Your secrets never leave your machine:

- **No cloud dependency** - Works completely offline
- **No account required** - No external authentication
- **No network exposure** - MCP uses stdio transport (Phase 0-2)
- **Full data ownership** - You control your vault files

### 4. Encryption at Rest

All secrets are encrypted using industry-standard algorithms:

- **AES-256-GCM** - Authenticated encryption for secrets
- **Argon2id** - Memory-hard key derivation function
- **OWASP parameters** - Following 2025 best practices

## Security Features Summary

### Encryption

| Feature | Implementation |
|---------|----------------|
| Symmetric encryption | AES-256-GCM |
| Key derivation | Argon2id (64MB, 3 iterations, 4 threads) |
| Nonce | 96-bit random per encryption |
| Salt | 128-bit per vault |

### Access Control

| Feature | Description |
|---------|-------------|
| Master password | Required to decrypt vault |
| File permissions | 0600 for sensitive files |
| MCP policy | Configurable allowed/denied keys |
| Output sanitization | Automatic secret redaction in command output |

### Audit & Compliance

| Feature | Description |
|---------|-------------|
| Audit logging | All operations recorded |
| HMAC chain | Tamper-evident log integrity |
| Export options | JSON, CSV for compliance |
| Verification | `secretctl audit verify` command |

## Threat Model

### What secretctl Protects

- **Confidentiality** of encrypted secrets at rest
- **Integrity** of vault data and audit logs
- **Access control** for MCP operations (no plaintext to AI)
- **Audit trail** of all secret access

### Out of Scope (Industry Standard Exclusions)

Like Vault, Infisical, and 1Password, secretctl excludes:

- **Root-level compromise** - If attacker has root, game over
- **Memory dump attacks** - Go runtime limitations (see below)
- **Weak master passwords** - User responsibility
- **Unlimited storage access** - Physical access assumed trusted

:::info Memory Protection Limitations
Go's garbage collector manages memory, making guaranteed zeroing difficult. This is an industry-standard exclusion, shared by Vault and Infisical. secretctl minimizes the time secrets exist in memory.
:::

### Attack Levels

| Level | Attacker | Capabilities | Protected? |
|-------|----------|--------------|------------|
| L1 | Network observer | Network interception | Yes (localhost only) |
| L2 | Malicious app | File system read | Yes (encryption) |
| L3 | Root attacker | Full system access | No (excluded) |

## Comparison with Alternatives

| Product | MCP Support | AI Plaintext Access | Local-First | OSS |
|---------|-------------|---------------------|-------------|-----|
| HashiCorp Vault | Yes (experimental) | Yes (`read_secret`) | No (server required) | BSL |
| Infisical | Yes | Yes (`get-secret`) | No (server required) | Yes |
| 1Password | No (refuses MCP) | No (policy) | No (subscription) | No |
| **secretctl** | **Yes** | **No (Option D+)** | **Yes** | **Yes** |

**Unique Position**: secretctl is the only solution with MCP support + no plaintext to AI + fully local + open source.

## 1Password's Security Principles

secretctl follows the same principles that guide 1Password's approach to AI:

| Principle | 1Password | secretctl |
|-----------|-----------|-----------|
| Secrets stay secret | Zero-knowledge encryption | AES-256-GCM + Argon2id |
| Deterministic authorization | LLMs don't make auth decisions | Policy engine controls access |
| No raw credentials to LLMs | Don't include secrets in prompts | Option D+ prohibits plaintext |
| Auditability | Record access and actions | Full audit logging |
| Transparency | Disclose what AI sees | Only masked values returned |
| Least privilege | Minimum necessary access | Policy-based key restrictions |
| Security built-in | Not bolted on | CLI/MCP/UI share consistent design |

## Why AI Agents Shouldn't See Secrets

1Password identified [four key reasons](https://1password.com/blog/where-mcp-fits-and-where-it-doesnt) to avoid giving secrets to AI:

| Risk | Description | secretctl Mitigation |
|------|-------------|---------------------|
| **Non-determinism** | AI behavior is unpredictable | Policy engine for deterministic control |
| **Prompt injection** | Malicious prompts could extract secrets | Secrets never reach AI context |
| **Irrevocability** | Can't un-see a secret in LLM context | Nothing to revoke if never exposed |
| **Cache/sharing** | AI might store or share downstream | No secret value exists to share |

## Quick Links

- [How It Works](/docs/security/how-it-works) - Detailed security architecture
- [Encryption Details](/docs/security/encryption) - Cryptographic implementation
- [MCP Tools Reference](/docs/reference/mcp-tools) - AI integration security
- [Audit Logs](/docs/guides/desktop/audit-logs) - Activity monitoring

## Security Reporting

Found a security issue? Please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email security concerns to the maintainer
3. Allow reasonable time for a fix before disclosure

We take security seriously and appreciate responsible disclosure.
