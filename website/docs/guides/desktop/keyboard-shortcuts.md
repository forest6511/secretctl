---
title: Keyboard Shortcuts
description: Master keyboard shortcuts for efficient secret management.
sidebar_position: 3
---

# Keyboard Shortcuts

The secretctl desktop app supports keyboard shortcuts for power users who want to work efficiently without reaching for the mouse.

## Prerequisites

- [Desktop app installed](/docs/guides/desktop/)
- Vault unlocked

## Quick Reference

### Essential Shortcuts

| Shortcut | Action | Context |
|----------|--------|---------|
| `Cmd/Ctrl + K` | Command Palette | Anywhere |
| `Cmd/Ctrl + N` | New secret | Anywhere |
| `Cmd/Ctrl + F` | Focus search | Anywhere |
| `Cmd/Ctrl + ,` | Settings | Anywhere |
| `Cmd/Ctrl + C` | Copy value | Secret selected |
| `Cmd/Ctrl + S` | Save changes | Editing |
| `Cmd/Ctrl + L` | Lock vault | Anywhere |
| `Cmd/Ctrl + /` | Show shortcuts | Anywhere |
| `Escape` | Cancel | Editing/Creating |

:::note
On macOS, use `Cmd`. On Windows/Linux, use `Ctrl`.
:::

## Command Palette

**Shortcut:** `Cmd/Ctrl + K`

The Command Palette provides quick access to all common actions:

1. Press `Cmd/Ctrl + K` to open
2. Type to filter commands
3. Press `Enter` to execute, or click a command
4. Press `Escape` to close

### Available Commands

| Command | Shortcut | Description |
|---------|----------|-------------|
| New Secret | `Cmd/Ctrl + N` | Create a new secret |
| Search Secrets | `Cmd/Ctrl + F` | Focus the search bar |
| Settings | `Cmd/Ctrl + ,` | Open settings page |
| Lock Vault | `Cmd/Ctrl + L` | Lock the vault |
| Keyboard Shortcuts | `Cmd/Ctrl + /` | Show shortcut help |

The Command Palette is searchable, so you can type partial command names to filter.

## Navigation Shortcuts

### Focus Search

**Shortcut:** `Cmd/Ctrl + F`

Quickly jump to the search bar to filter secrets:

1. Press `Cmd/Ctrl + F`
2. Start typing to filter
3. Press `Escape` to clear focus

The search field highlights and accepts input immediately.

### Lock Vault

**Shortcut:** `Cmd/Ctrl + L`

Immediately lock the vault for security:

1. Press `Cmd/Ctrl + L`
2. Vault locks instantly
3. Returns to unlock screen

Use this when stepping away from your computer.

## Secret Management Shortcuts

### Create New Secret

**Shortcut:** `Cmd/Ctrl + N`

Start creating a new secret:

1. Press `Cmd/Ctrl + N`
2. Form opens with Key field focused
3. Fill in details and save

Works from any screen in the app.

### Copy Secret Value

**Shortcut:** `Cmd/Ctrl + C`

Copy the selected secret's value to clipboard:

**Requirements:**
- A secret must be selected
- Not currently editing text
- No text selection in the app

**Behavior:**
1. Press `Cmd/Ctrl + C`
2. Value copies to clipboard
3. Toast confirms: "Copied! Auto-clears in 30s"

:::tip
If you have text selected in the app, `Cmd/Ctrl + C` copies the selected text instead. To copy the secret value, deselect any text first.
:::

### Save Changes

**Shortcut:** `Cmd/Ctrl + S`

Save when creating or editing a secret:

1. Make your changes
2. Press `Cmd/Ctrl + S`
3. Changes are saved

Only active when in edit or create mode.

### Cancel Editing

**Shortcut:** `Escape`

Discard changes and exit edit mode:

1. Press `Escape`
2. Changes are discarded
3. Returns to view mode

Also closes dialogs and modals.

## Workflow Examples

### Quick Secret Lookup

Find and copy a secret in seconds:

1. `Cmd/Ctrl + F` - Focus search
2. Type secret name
3. `↓` or click to select
4. `Cmd/Ctrl + C` - Copy value

### Create Secret Workflow

Efficiently create a new secret:

1. `Cmd/Ctrl + N` - Open new form
2. Type key name
3. `Tab` - Move to value
4. Type value
5. `Cmd/Ctrl + S` - Save

### Security Lock Pattern

Before leaving your desk:

1. `Cmd/Ctrl + L` - Lock vault
2. Walk away safely

The vault requires your password to unlock again.

## Platform Differences

### macOS

- Uses `Cmd` (⌘) key
- Native keyboard behavior
- Touch Bar support (if available)

### Windows

- Uses `Ctrl` key
- Standard Windows shortcuts
- Alt key alternatives not yet supported

### Linux

- Uses `Ctrl` key
- Follows GTK conventions
- Works with most desktop environments

## Accessibility

### Keyboard-Only Navigation

The app is designed to be fully usable with keyboard only:

1. `Tab` moves between focusable elements
2. `Enter` activates buttons
3. `Space` toggles checkboxes
4. `Escape` closes dialogs

### Screen Reader Support

The app includes accessibility labels for:
- All buttons and icons
- Form fields
- Status messages
- Navigation elements

## Customization

Currently, keyboard shortcuts cannot be customized. The default shortcuts are designed to be familiar to users of similar applications.

**Future plans:**
- Custom shortcut mapping
- Additional shortcuts for power users
- Vim-style navigation (optional)

## Troubleshooting

### Shortcut Not Working

**Check these common issues:**

1. **Focus**: Is the app in focus? Click the window first.
2. **Context**: Some shortcuts only work in specific contexts (e.g., save requires edit mode).
3. **Selection**: Text selection may override some shortcuts.
4. **System conflicts**: Other apps may capture global shortcuts.

### System Shortcut Conflicts

Some system shortcuts may conflict:

| System | Conflicting Shortcuts |
|--------|----------------------|
| macOS Spotlight | `Cmd + Space` (not used) |
| Windows Search | `Ctrl + F` (app captures when focused) |
| Linux Launchers | Varies by DE |

The app only captures shortcuts when it has focus.

### Text vs Value Copy

`Cmd/Ctrl + C` behavior:

- **With text selected**: Copies selected text (standard behavior)
- **Without selection**: Copies secret value (app shortcut)

To ensure you copy the secret value:
1. Click somewhere that deselects text
2. Ensure a secret is selected
3. Press `Cmd/Ctrl + C`

## Complete Shortcut Table

| Shortcut | Action | Works When |
|----------|--------|------------|
| `Cmd/Ctrl + K` | Open Command Palette | Always |
| `Cmd/Ctrl + N` | New secret | Always |
| `Cmd/Ctrl + F` | Focus search | Always |
| `Cmd/Ctrl + ,` | Open settings | Always |
| `Cmd/Ctrl + C` | Copy secret value | Secret selected, no text selection |
| `Cmd/Ctrl + S` | Save | Editing or creating |
| `Cmd/Ctrl + L` | Lock vault | Always |
| `Cmd/Ctrl + /` | Show shortcuts help | Always |
| `Escape` | Cancel/Close | Editing, creating, or dialog open |
| `Tab` | Next field | Form navigation |
| `Shift + Tab` | Previous field | Form navigation |
| `Enter` | Activate button | Button focused |

## Next Steps

- [Managing Secrets](/docs/guides/desktop/managing-secrets) - Complete secret operations
- [Audit Logs](/docs/guides/desktop/audit-logs) - View activity history
- [Desktop App Overview](/docs/guides/desktop/) - Full feature guide
