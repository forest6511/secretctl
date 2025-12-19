---
title: Desktop App
description: Use the secretctl desktop application for visual secret management.
sidebar_position: 1
---

# Desktop App Guide

The secretctl desktop app provides a graphical interface for managing your secrets. Built with native performance, it offers the same security as the CLI with an intuitive visual experience.

## Overview

The desktop app is designed for:

- **Visual users** who prefer GUI over command line
- **Quick access** to view and copy secrets
- **Team members** who need occasional secret access
- **Audit review** with visual filtering and export

## Features

### Secret Management

- Create, view, edit, and delete secrets
- Rich metadata support (notes, tags, URLs)
- Search and filter secrets instantly
- Secure clipboard copy with auto-clear

### Security

- Same encryption as CLI (AES-256-GCM + Argon2id)
- Auto-lock after inactivity
- Visual password masking
- Tamper-evident audit logging

### Audit Logs

- Visual audit log viewer
- Filter by action, source, date range
- Chain integrity verification
- Export to CSV/JSON

## Installation

### macOS

Download the latest `.dmg` from [GitHub Releases](https://github.com/forest6511/secretctl/releases):

```bash
# Or install via Homebrew (coming soon)
brew install --cask secretctl
```

### Windows

Download the latest `.exe` installer from [GitHub Releases](https://github.com/forest6511/secretctl/releases).

### Linux

Download the appropriate package for your distribution:

- `.deb` for Debian/Ubuntu
- `.rpm` for Fedora/RHEL
- `.AppImage` for universal compatibility

## Getting Started

### First Launch

1. **Launch the app** - Open secretctl from your applications
2. **Create vault** - Set a master password (minimum 8 characters)
3. **Start adding secrets** - Click "Add Secret" to store your first secret

### Returning Users

1. **Launch the app**
2. **Enter master password** - Unlock your existing vault
3. **Access secrets** - Browse, search, or modify your secrets

## Interface Overview

### Sidebar

The left sidebar displays:
- **Secret list** - All stored secrets
- **Search bar** - Filter secrets by name
- **Action buttons** - Refresh, lock, and navigate

### Detail Panel

The right panel shows:
- **Secret value** - With show/hide toggle
- **Metadata** - URL, tags, notes
- **Timestamps** - Created and updated dates
- **Actions** - Copy, edit, delete

### Header Bar

Quick access to:
- **Audit Log** - View activity history
- **Refresh** - Reload secret list
- **Lock** - Secure the vault

## Keyboard Shortcuts

Power users can navigate quickly with shortcuts:

| Shortcut | Action |
|----------|--------|
| `Cmd/Ctrl + N` | New secret |
| `Cmd/Ctrl + F` | Focus search |
| `Cmd/Ctrl + C` | Copy selected value |
| `Cmd/Ctrl + S` | Save changes |
| `Cmd/Ctrl + L` | Lock vault |
| `Escape` | Cancel editing |

See [Keyboard Shortcuts](/docs/guides/desktop/keyboard-shortcuts) for the complete list.

## Shared Vault

The desktop app shares the same vault as the CLI:

- Secrets created in CLI appear in the app
- Secrets created in the app are available via CLI
- Both use the same encryption and storage

**Vault location**: `~/.secretctl/`

## Comparison with CLI

| Feature | Desktop App | CLI |
|---------|-------------|-----|
| Visual interface | Yes | No |
| Keyboard shortcuts | Yes | N/A |
| Batch operations | No | Yes |
| Scripting/automation | No | Yes |
| Password generation | No | Yes |
| Secret injection | No | Yes (`run`) |
| Audit log viewer | Yes | Yes |

Use the desktop app for visual access and the CLI for automation.

## Security Considerations

### Auto-Lock

The vault automatically locks after 15 minutes of inactivity. Any mouse or keyboard activity resets the timer.

### Clipboard Security

When copying a secret:
- Value is copied to system clipboard
- Auto-clears after 30 seconds
- Toast notification confirms the action

### Screen Privacy

- Secret values are hidden by default
- Click the eye icon to reveal temporarily
- Values are masked when switching away

## Next Steps

- [Managing Secrets](/docs/guides/desktop/managing-secrets) - Create, edit, delete secrets
- [Keyboard Shortcuts](/docs/guides/desktop/keyboard-shortcuts) - Master the shortcuts
- [Audit Logs](/docs/guides/desktop/audit-logs) - View and export activity logs
