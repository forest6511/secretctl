---
title: Troubleshooting
description: Common issues and solutions for secretctl.
sidebar_position: 2
---

# Troubleshooting

Common issues and their solutions.

## Installation Issues

### "command not found: secretctl"

The binary is not in your PATH.

**Solution:**

```bash
# Check where you installed it
ls -la /usr/local/bin/secretctl

# Or add the installation directory to PATH
export PATH=$PATH:/path/to/secretctl
```

### Permission denied when running secretctl

The binary doesn't have execute permissions.

**Solution:**

```bash
chmod +x /usr/local/bin/secretctl
```

## Vault Issues

### "vault not initialized"

You're trying to use secretctl before creating a vault.

**Solution:**

```bash
secretctl init
```

### "failed to unlock vault: invalid password"

The password you entered is incorrect.

**Solution:**

- Double-check your password
- Passwords are case-sensitive
- If you've forgotten your password, you'll need to restore from a backup

### "vault already exists"

You're trying to initialize a vault that already exists.

**Solution:**

```bash
# If you want to start fresh (WARNING: destroys existing secrets)
rm -rf ~/.secretctl
secretctl init

# Or use a different vault location
secretctl init --vault-dir=/path/to/new/vault
```

## MCP Server Issues

### Claude Code doesn't see secretctl

**Check 1:** Verify the binary path

```bash
which secretctl
# Should output: /usr/local/bin/secretctl
```

**Check 2:** Use absolute path in Claude Code config

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/usr/local/bin/secretctl",
      "args": ["mcp-server"]
    }
  }
}
```

**Check 3:** Restart Claude Code after config changes

### "no password provided"

The MCP server needs a password to unlock the vault.

**Solution:**

Set the `SECRETCTL_PASSWORD` environment variable in your Claude Code config:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

### MCP server connection timeout

The server is taking too long to start.

**Solution:**

1. Test the MCP server manually:
   ```bash
   SECRETCTL_PASSWORD=your-password secretctl mcp-server 2>&1 | head -5
   ```

2. Check for vault issues:
   ```bash
   secretctl list
   ```

## Desktop App Issues

### App won't start

**macOS:** You may need to allow the app in System Preferences > Security & Privacy.

**Windows:** You may need to allow the app through Windows Defender.

**Linux:** Check if the binary has execute permissions:
```bash
chmod +x secretctl-desktop
```

### "Failed to connect to backend"

The Go backend isn't communicating with the frontend.

**Solution:**

1. Close the app completely
2. Delete any stale lock files:
   ```bash
   rm -f ~/.secretctl/*.lock
   ```
3. Restart the app

### Session timeout too aggressive

The app locks after 15 minutes of inactivity by default.

**Solution:**

This is a security feature and cannot be disabled. Keep the app active or unlock when needed.

## Secret Management Issues

### "key already exists"

You're trying to create a secret with a key that already exists.

**Solution:**

```bash
# Update the existing secret
echo "new-value" | secretctl set existing-key

# Or delete first, then create
secretctl delete existing-key
echo "new-value" | secretctl set existing-key
```

### Secrets not found in `secretctl run`

The key pattern might not match any secrets.

**Solution:**

```bash
# List all secrets to see available keys
secretctl list

# Check your pattern matches
secretctl list | grep "aws"

# Use correct pattern
secretctl run -k "aws/*" -- your-command
```

### Output sanitization removing valid output

The sanitizer detected secret-like patterns in your command output.

**Solution:**

This is a security feature. If you need to see the output without sanitization:

```bash
# Use the CLI directly (not through MCP)
secretctl get MY_SECRET

# Or export to a file
secretctl export --format=env > .env
```

## Backup & Restore Issues

### "invalid backup file"

The backup file is corrupted or not a valid secretctl backup.

**Solution:**

1. Verify the file exists and has content:
   ```bash
   ls -la backup.enc
   ```

2. Try with `--verify-only` first:
   ```bash
   secretctl restore backup.enc --verify-only
   ```

### "password mismatch" during restore

The password used for restore doesn't match the backup password.

**Solution:**

Use the same password that was used when creating the backup.

## Performance Issues

### Slow vault operations

SQLite database might be fragmented.

**Solution:**

1. Lock the vault
2. Make a backup
3. Restore from backup (creates fresh database)

```bash
secretctl lock
secretctl backup -o backup.enc
rm -rf ~/.secretctl
secretctl restore backup.enc
```

## Getting More Help

If your issue isn't listed here:

1. Check [GitHub Issues](https://github.com/forest6511/secretctl/issues) for similar problems
2. Read the [FAQ](/docs/help/faq) for common questions
3. Open a new issue with:
   - secretctl version (`secretctl --version`)
   - Operating system and version
   - Steps to reproduce
   - Error messages (with secrets redacted)
