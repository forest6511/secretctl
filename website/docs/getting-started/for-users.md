---
title: For General Users
description: Get started with secretctl Desktop App for secure password management.
sidebar_position: 3
---

# Getting Started: For General Users

This guide is for users who want a simple, secure way to manage passwords, API keys, and other secrets using the Desktop App or basic CLI commands.

## What You'll Learn

- Install and set up the Desktop App
- Create your secure vault
- Add, view, and manage your secrets
- Keep your credentials organized and safe

## Why secretctl?

- **Completely Local**: Your secrets never leave your computer
- **No Account Required**: No sign-up, no cloud sync, no subscription
- **Strong Encryption**: Military-grade AES-256 encryption
- **Simple Interface**: Easy-to-use desktop application

## Option 1: Desktop App (Recommended)

### Step 1: Download

1. Go to [GitHub Releases](https://github.com/forest6511/secretctl/releases)
2. Download the file for your system:
   - **macOS**: `secretctl-desktop-macos.zip`
   - **Windows**: `secretctl-desktop-windows.exe`
   - **Linux**: `secretctl-desktop-linux`

### Step 2: Install

**macOS**:
1. Unzip the downloaded file
2. Drag the app to your Applications folder
3. Right-click and select "Open" (first time only, to bypass Gatekeeper)

**Windows**:
1. Run the installer
2. Follow the installation wizard
3. Launch from Start Menu

**Linux**:
1. Make executable: `chmod +x secretctl-desktop-linux`
2. Run: `./secretctl-desktop-linux`

### Step 3: Create Your Vault

1. Open the secretctl app
2. Click "Create New Vault"
3. Enter a strong master password
4. Confirm your password
5. Click "Create"

:::tip Choosing a Master Password
- Use at least 12 characters
- Mix letters, numbers, and symbols
- Consider using a passphrase like "correct-horse-battery-staple"
- **Write it down** and store it safely - there's no recovery if you forget it!
:::

### Step 4: Add Your First Secret

1. Click the "+" button or "Add Secret"
2. Enter a name (e.g., "Gmail Password" or "OPENAI_API_KEY")
3. Enter the secret value
4. Optionally add:
   - **Notes**: Additional information
   - **URL**: Related website
   - **Tags**: Categories like "email", "work", "api"
5. Click "Save"

### Step 5: View and Use Secrets

- **View**: Click on any secret to see details
- **Copy**: Click the copy icon to copy to clipboard (auto-clears after 30 seconds)
- **Search**: Use the search bar to find secrets quickly
- **Filter**: Filter by tags or date

### Step 6: Stay Secure

- The app auto-locks after 15 minutes of inactivity
- Lock manually: Click the lock icon or use `Cmd+L` (Mac) / `Ctrl+L` (Windows)
- Always lock before leaving your computer

## Option 2: Command Line (CLI)

If you prefer using the terminal:

### Install

**macOS (Apple Silicon)**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl
```

**macOS (Intel)**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-amd64
chmod +x secretctl-darwin-amd64
sudo mv secretctl-darwin-amd64 /usr/local/bin/secretctl
```

**Linux**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl
```

**Windows**: Download `secretctl-windows-amd64.exe` from [GitHub Releases](https://github.com/forest6511/secretctl/releases).

### Basic Commands

```bash
# Create your vault
secretctl init

# Add a secret
echo "my-password-123" | secretctl set "Gmail Password"

# View a secret
secretctl get "Gmail Password"

# List all secrets
secretctl list

# Delete a secret
secretctl delete "Gmail Password"
```

## Organizing Your Secrets

### Use Key Prefixes

Organize secrets by category using prefixes:

```
email/gmail
email/outlook
social/twitter
social/facebook
work/github
work/aws_key
banking/chase
```

### Use Tags

Add tags when creating secrets:
- Desktop: Enter tags in the Tags field
- CLI: `echo "value" | secretctl set KEY --tags "work,api,important"`

### Add Notes and URLs

Keep track of where each secret is used:
- Desktop: Fill in the Notes and URL fields
- CLI: `echo "value" | secretctl set KEY --notes "Main account" --url "https://example.com"`

## Backup Your Secrets

### Create a Backup

**Desktop**: Menu → File → Backup Vault

**CLI**:
```bash
secretctl backup -o ~/Desktop/my-secrets-backup.enc
```

The backup is encrypted with your master password.

### Restore from Backup

**Desktop**: Menu → File → Restore from Backup

**CLI**:
```bash
secretctl restore ~/Desktop/my-secrets-backup.enc
```

## Keyboard Shortcuts (Desktop App)

| Action | macOS | Windows/Linux |
|--------|-------|---------------|
| Lock vault | `Cmd+L` | `Ctrl+L` |
| New secret | `Cmd+N` | `Ctrl+N` |
| Search | `Cmd+F` | `Ctrl+F` |
| Copy value | `Cmd+C` | `Ctrl+C` |
| Quit | `Cmd+Q` | `Ctrl+Q` |

## Frequently Asked Questions

### What if I forget my master password?

Unfortunately, there's no recovery option. The master password is the only way to decrypt your secrets. We recommend:
- Writing it down and storing it in a safe place
- Using a memorable passphrase

### Are my secrets synced to the cloud?

No. secretctl is completely local. Your secrets are stored only on your computer in an encrypted file.

### Can I use it on multiple computers?

Yes, but you'll need to manually transfer your vault:
1. Create a backup on Computer A
2. Copy the backup file to Computer B
3. Restore the backup on Computer B

### Is it safe?

Yes. secretctl uses:
- AES-256-GCM encryption (same as banks and governments)
- Argon2id key derivation (protects against password cracking)
- Local-only storage (no network exposure)

## Next Steps

- [Desktop App Guide](/docs/guides/desktop/) - Complete desktop app documentation
- [CLI Guide](/docs/guides/cli/) - Learn more CLI commands
- [Security Overview](/docs/security/) - Understand how your data is protected
- [FAQ](/docs/help/faq) - More frequently asked questions
