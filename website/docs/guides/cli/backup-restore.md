---
title: Backup and Restore
description: How to backup and restore your secretctl vault securely.
sidebar_position: 4
---

# Backup and Restore

secretctl provides encrypted backup and restore functionality to protect your secrets against data loss and enable vault migration.

## Overview

- **Encrypted backups** — AES-256-GCM encryption with fresh salt for each backup
- **Integrity verification** — HMAC-SHA256 detects any tampering
- **Atomic restore** — All-or-nothing restoration prevents partial states
- **Key file support** — Enable automated backups without password prompts

## Creating Backups

### Basic Backup

Create an encrypted backup with your master password:

```bash
secretctl backup -o vault-backup.enc
```

### Include Audit Logs

For a complete backup including audit history:

```bash
secretctl backup -o full-backup.enc --with-audit
```

### Use a Separate Backup Password

Use a different password for the backup (prompted):

```bash
secretctl backup -o backup.enc --backup-password
```

### Use Key File (for Automation)

For automated backup scripts, use a key file instead of password prompts:

```bash
# Generate a key file (once)
head -c 32 /dev/urandom > backup.key
chmod 600 backup.key

# Backup using key file
secretctl backup -o backup.enc --key-file=backup.key
```

:::warning
Keep your key file as secure as your master password. Anyone with the key file can decrypt your backups.
:::

### Pipe to External Tools

Backup to stdout for piping to encryption tools:

```bash
# Encrypt with GPG
secretctl backup --stdout | gpg --encrypt -r you@email.com > backup.gpg

# Compress and encrypt
secretctl backup --stdout | gzip | gpg --encrypt > backup.gz.gpg
```

## Restoring Backups

### Verify Backup First

Always verify backup integrity before restoring:

```bash
secretctl restore backup.enc --verify-only
```

Output shows backup metadata:
```
Backup verification successful!
  Version: 1
  Created: 2025-01-15 10:30:00
  Secrets: 42
  Includes Audit: true
```

### Preview Restore (Dry Run)

See what would be restored without making changes:

```bash
secretctl restore backup.enc --dry-run
```

### Restore to Empty Vault

For a fresh restore (no existing vault):

```bash
secretctl restore backup.enc
```

### Handle Conflicts

When restoring to an existing vault:

```bash
# Skip existing keys (add new ones only)
secretctl restore backup.enc --on-conflict=skip

# Overwrite existing keys
secretctl restore backup.enc --on-conflict=overwrite

# Error on conflicts (default)
secretctl restore backup.enc --on-conflict=error
```

### Restore with Audit Logs

Restore audit logs from backup (overwrites existing):

```bash
secretctl restore backup.enc --with-audit
```

### Use Key File

For backups created with a key file:

```bash
secretctl restore backup.enc --key-file=backup.key
```

## Backup Format

secretctl backups use a secure, versioned format:

| Component | Description |
|-----------|-------------|
| Magic number | 8-byte identifier (`SCTL_BKP`) |
| Header | JSON metadata (version, timestamps, encryption mode) |
| Encrypted payload | AES-256-GCM encrypted vault data |
| HMAC | HMAC-SHA256 integrity check |

### Key Derivation

For password-based backups:

1. Fresh 32-byte salt generated per backup
2. Argon2id KDF (64MB memory, 3 iterations, 4 threads)
3. HKDF-SHA256 derives separate encryption and MAC keys

## Best Practices

### Regular Backups

Schedule automated backups using cron:

```bash
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/secretctl backup \
  -o /backup/vault-$(date +\%Y\%m\%d).enc \
  --key-file=/etc/secretctl/backup.key
```

### Backup Rotation

Keep multiple backup generations:

```bash
#!/bin/bash
BACKUP_DIR=/backup/secretctl
MAX_BACKUPS=7

# Create new backup
secretctl backup -o "$BACKUP_DIR/vault-$(date +%Y%m%d-%H%M%S).enc" \
  --key-file=/etc/secretctl/backup.key

# Remove old backups
ls -t "$BACKUP_DIR"/vault-*.enc | tail -n +$((MAX_BACKUPS + 1)) | xargs rm -f
```

### Offsite Storage

Store backups in multiple locations:

```bash
# Backup to cloud storage
secretctl backup --stdout | aws s3 cp - s3://my-backups/vault-backup.enc

# Backup to remote server
secretctl backup --stdout | ssh backup-server "cat > /backup/vault.enc"
```

### Test Restores

Periodically verify backups can be restored:

```bash
# Restore to temporary location
SECRETCTL_VAULT_DIR=/tmp/test-restore secretctl restore backup.enc

# Verify secrets
SECRETCTL_VAULT_DIR=/tmp/test-restore secretctl list

# Clean up
rm -rf /tmp/test-restore
```

## Troubleshooting

### "Backup integrity check failed"

The backup file may be corrupted or tampered with. Restore from a different backup.

### "Invalid password or corrupted data"

Either the password is incorrect, or the backup is damaged. Try:

1. Verify you're using the correct password
2. If using key file, ensure it's the original file
3. Check if the backup file was truncated during transfer

### "Vault is locked by another process"

Another secretctl process is accessing the vault. Wait for it to complete or check for stale lock files.

## Security Considerations

- **Backup passwords** are never stored; you must remember them
- **Key files** should be stored separately from backups
- **Audit logs** may contain sensitive information about access patterns
- **Backup files** should have restricted permissions (0600)
