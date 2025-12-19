---
title: Encryption Details
description: Cryptographic implementation and specifications.
sidebar_position: 3
---

# Encryption Details

This guide provides detailed information about the cryptographic algorithms, parameters, and implementation used in secretctl.

## Cryptographic Specifications

### Summary

| Component | Algorithm | Parameters |
|-----------|-----------|------------|
| Symmetric encryption | AES-256-GCM | 256-bit key, 96-bit nonce |
| Key derivation | Argon2id | 64MB memory, 3 iterations, 4 threads |
| Salt | Random | 128-bit (16 bytes) |
| Nonce | Random | 96-bit (12 bytes) per encryption |
| HMAC (audit logs) | HMAC-SHA256 | 256-bit key |

### Standards Compliance

| Specification | Reference |
|---------------|-----------|
| AES-GCM | NIST SP 800-38D |
| Argon2 | RFC 9106 |
| OWASP | Password Storage Cheat Sheet |
| Key derivation | HKDF (RFC 5869) |

## AES-256-GCM

### Why AES-256-GCM?

| Property | Benefit |
|----------|---------|
| **Authenticated encryption** | Detects tampering automatically |
| **NIST recommended** | Government and industry standard |
| **Hardware acceleration** | Fast on modern CPUs (AES-NI) |
| **Well-studied** | Decades of cryptanalysis |

### Implementation

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
)

func encrypt(plaintext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    // Generate random nonce
    nonce := make([]byte, gcm.NonceSize()) // 12 bytes
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    // Encrypt and authenticate
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, errors.New("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

### Blob Format

All encrypted data uses a consistent format:

```
┌──────────────┬─────────────────────┬────────────────┐
│ Nonce (12B)  │ Ciphertext          │ GCM Tag (16B)  │
└──────────────┴─────────────────────┴────────────────┘
```

**Applies to:**
- `encrypted_dek` (vault_keys table)
- `encrypted_key` (secrets table)
- `encrypted_value` (secrets table)
- `encrypted_metadata` (secrets table)

### Benefits of This Format

| Benefit | Description |
|---------|-------------|
| Self-contained | Each blob can be decrypted independently |
| No separate nonce column | Simpler database schema |
| Industry standard | Same as libsodium sealed boxes |

## Argon2id Key Derivation

### Why Argon2id?

| Property | Benefit |
|----------|---------|
| **Memory-hard** | Expensive for GPUs/ASICs |
| **Time-hard** | Multiple iterations required |
| **Parallelism** | Utilizes multiple cores |
| **RFC standard** | RFC 9106 (2021) |

Argon2id combines Argon2i (side-channel resistant) and Argon2d (GPU resistant).

### Parameters

```go
import "golang.org/x/crypto/argon2"

const (
    memory      = 64 * 1024  // 64 MB
    iterations  = 3
    parallelism = 4
    keyLength   = 32         // 256 bits
    saltLength  = 16         // 128 bits
)

func deriveKey(password string, salt []byte) []byte {
    return argon2.IDKey(
        []byte(password),
        salt,
        iterations,
        memory,
        parallelism,
        keyLength,
    )
}
```

### OWASP Compliance

These parameters follow [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html):

| Parameter | Value | OWASP Recommendation |
|-----------|-------|---------------------|
| Memory | 64 MB | 64 MB minimum |
| Time | 3 | 3 iterations |
| Threads | 4 | 4 (modern multi-core) |

### Performance by Environment

| Environment | Memory | Unlock Time | Status |
|-------------|--------|-------------|--------|
| Modern PC/Mac | 8GB+ | 0.5-1 sec | Optimal |
| Low-spec laptop | 4GB | 1-2 sec | Good |
| Raspberry Pi 4 | 2-8GB | 1-3 sec | Acceptable |
| CI environment | 2-4GB | 1-2 sec | Good |
| Docker (limited) | Varies | Varies | 64MB+ required |

:::warning Docker Memory
Containers with less than 64MB memory will fail during key derivation. Ensure your container has sufficient memory allocated.
:::

## Nonce Management

### Uniqueness Guarantee

AES-GCM requires unique nonces. Reusing a nonce with the same key compromises security (Forbidden Attack).

### Strategy: Random Nonce

```go
func generateNonce() ([]byte, error) {
    nonce := make([]byte, 12) // 96 bits
    if _, err := rand.Read(nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }
    return nonce, nil
}
```

### Collision Probability

| Nonce Length | Collision After | Probability |
|--------------|-----------------|-------------|
| 96-bit | 2^32 encryptions | 2^-33 ≈ 1 in 8.6 billion |

For personal use, reaching 4 billion encryptions is practically impossible.

### Randomness Source

```go
import "crypto/rand"
```

Go's `crypto/rand` uses:
- **Linux**: `/dev/urandom` (CSPRNG)
- **macOS**: `getentropy()` (kernel entropy)
- **Windows**: `CryptGenRandom()` (CryptoAPI)

All are cryptographically secure pseudo-random number generators.

## Key Hierarchy Details

### Three-Tier Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Key Hierarchy                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Tier 1: Master Password (user input)                               │
│          │                                                          │
│          │ Never stored, used only for derivation                   │
│          │                                                          │
│          ▼                                                          │
│  Tier 2: Master Key (derived)                                       │
│          │                                                          │
│          │ = Argon2id(password, salt)                               │
│          │ Lives in memory only during session                      │
│          │                                                          │
│          ▼                                                          │
│  Tier 3: Data Encryption Key (DEK)                                  │
│          │                                                          │
│          │ Stored as: AES-GCM(DEK, MasterKey)                       │
│          │ Used to encrypt all secrets                              │
│          │                                                          │
│          ▼                                                          │
│  Secrets: AES-GCM(secret, DEK)                                      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Why This Design?

| Feature | Benefit |
|---------|---------|
| **Password rotation** | Change password without re-encrypting all secrets |
| **Defense in depth** | Compromise of one layer doesn't expose everything |
| **Session isolation** | Master key cleared after lock |

### Password Rotation Flow

```
1. Unlock vault (derive old master key)
2. Decrypt DEK with old master key
3. Generate new salt
4. Derive new master key from new password
5. Re-encrypt DEK with new master key
6. Store new encrypted DEK and salt
7. Secrets remain unchanged (still encrypted with same DEK)
```

## Database Schema

### Tables

```sql
-- Encrypted DEK storage
CREATE TABLE vault_keys (
    id INTEGER PRIMARY KEY,
    encrypted_dek BLOB NOT NULL,  -- nonce || AES-GCM ciphertext
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Secrets storage
CREATE TABLE secrets (
    id INTEGER PRIMARY KEY,
    key_hash TEXT UNIQUE NOT NULL,     -- SHA-256(key) for lookup
    encrypted_key BLOB NOT NULL,       -- nonce || encrypted key name
    encrypted_value BLOB NOT NULL,     -- nonce || encrypted value
    encrypted_metadata BLOB,           -- nonce || encrypted JSON (optional)
    tags TEXT,                         -- comma-separated, plaintext
    expires_at TIMESTAMP,              -- plaintext for queries
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit log with HMAC chain
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY,
    action TEXT NOT NULL,              -- get, set, delete, list
    key_hash TEXT,                     -- SHA-256 of accessed key
    source TEXT NOT NULL,              -- cli, mcp, ui
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    prev_hash TEXT NOT NULL,           -- previous record hash
    record_hash TEXT NOT NULL          -- HMAC of this record
);
```

### Why key_hash?

Storing `SHA-256(key)` instead of the plaintext key name:
- Allows lookups without decryption
- Protects key names at rest
- One-way (can't derive key name from hash)

## Metadata Encryption

### Structure

```go
type SecretMetadata struct {
    Version int    `json:"v"`              // Schema version (1)
    Notes   string `json:"notes,omitempty"` // Max 10KB
    URL     string `json:"url,omitempty"`   // Max 2048 chars
}
```

### Storage Rules

| Condition | encrypted_metadata |
|-----------|-------------------|
| notes="" AND url="" | NULL (no encryption) |
| notes OR url has value | nonce + AES-GCM(JSON) |

### MCP Access Restrictions

Option D+ extends to metadata:

| Data | MCP Access |
|------|------------|
| `key` (name) | Yes (via secret_list) |
| `tags` | Yes (plaintext, for search) |
| `expires_at` | Yes (plaintext, for queries) |
| `has_notes` | Yes (boolean flag only) |
| `has_url` | Yes (boolean flag only) |
| `notes` content | **No** |
| `url` content | **No** |
| `value` | **No** |

**Rule**: Encrypted columns = MCP access prohibited.

## Audit Log Integrity

### HMAC Chain

```go
// Derive audit key from master key
auditKey := hkdf.Expand(sha256.New, masterKey, []byte("audit-log-v1"), 32)

// Compute record HMAC
func computeRecordHMAC(record AuditRecord, prevHash string, key []byte) string {
    data := fmt.Sprintf("%d|%s|%s|%s|%s|%s",
        record.ID,
        record.Action,
        record.KeyHash,
        record.Source,
        record.Timestamp.Format(time.RFC3339Nano),
        prevHash,
    )
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(data))
    return hex.EncodeToString(mac.Sum(nil))
}
```

### Verification

```go
func verifyChain(records []AuditRecord, key []byte) error {
    for i, r := range records {
        // Verify HMAC
        var prevHash string
        if i > 0 {
            prevHash = records[i-1].RecordHash
        }
        expected := computeRecordHMAC(r, prevHash, key)
        if r.RecordHash != expected {
            return fmt.Errorf("record %d: HMAC mismatch", r.ID)
        }

        // Verify chain
        if i > 0 && r.PrevHash != records[i-1].RecordHash {
            return fmt.Errorf("record %d: chain broken", r.ID)
        }
    }
    return nil
}
```

## Libraries Used

### Go Standard Library

```go
import (
    "crypto/aes"           // AES encryption
    "crypto/cipher"        // GCM mode
    "crypto/hmac"          // HMAC
    "crypto/rand"          // Secure random
    "crypto/sha256"        // SHA-256 hashing
)
```

### golang.org/x/crypto

```go
import (
    "golang.org/x/crypto/argon2"  // Key derivation
    "golang.org/x/crypto/hkdf"    // Key expansion
)
```

### Why These Libraries?

| Library | Reason |
|---------|--------|
| Go standard library | Battle-tested, audited, maintained |
| golang.org/x/crypto | Official Go cryptography extensions |

No third-party cryptography libraries are used, minimizing supply chain risk.

## Security Considerations

### What's Protected

| Asset | Protection |
|-------|------------|
| Secret values | AES-256-GCM encryption |
| Key names | AES-256-GCM encryption |
| Metadata | AES-256-GCM encryption |
| Master password | Never stored |
| Audit integrity | HMAC chain |

### What's NOT Protected

| Asset | Reason |
|-------|--------|
| Tags | Stored plaintext for search |
| Expiration dates | Stored plaintext for queries |
| Record counts | Observable from DB |
| Access patterns | Protected by audit log |

### Backup Considerations

```bash
# Safe to backup (encrypted)
~/.secretctl/vault.db
~/.secretctl/vault.salt
~/.secretctl/vault.meta

# Required for restore
# - All three files above
# - Master password (not stored)
```

:::warning Lost Password
If you forget your master password, your secrets cannot be recovered. This is by design - there are no backdoors.
:::

## Next Steps

- [Security Overview](/docs/security/) - High-level security architecture
- [How It Works](/docs/security/how-it-works) - Security data flow
- [MCP Tools Reference](/docs/reference/mcp-tools) - AI integration security
