---
title: Managing Secrets
description: Create, view, edit, and delete secrets using the desktop app.
sidebar_position: 2
---

# Managing Secrets

This guide covers how to manage secrets using the secretctl desktop app, including creating, viewing, editing, and deleting secrets.

## Prerequisites

- [Desktop app installed](/docs/guides/desktop/)
- Vault unlocked with master password

## Viewing Secrets

### Secret List

The sidebar displays all your secrets:

1. **Browse** - Scroll through the secret list
2. **Click** - Select a secret to view details
3. **Tags** - See tags displayed under each secret name

### Search Secrets

Use the search bar to filter secrets:

1. Click the search bar or press `Cmd/Ctrl + F`
2. Type part of the secret name
3. Results filter in real-time

The search matches against secret keys (names).

### Secret Details

When you select a secret, the detail panel shows:

- **Key** - The secret's unique identifier
- **Value** - The secret content (hidden by default)
- **URL** - Associated URL (if set)
- **Tags** - Categorization labels
- **Notes** - Additional information
- **Created** - When the secret was created
- **Updated** - When last modified

### Show/Hide Value

Secret values are masked for security:

1. Click the **eye icon** next to the value field
2. The value becomes visible
3. Click again to hide

:::tip
Values are automatically hidden when you select a different secret.
:::

## Creating Secrets

### Add a New Secret

1. Click the **"Add Secret"** button at the bottom of the sidebar
2. Or press `Cmd/Ctrl + N`

### Fill in the Form

| Field | Required | Description |
|-------|----------|-------------|
| Key | Yes | Unique identifier (e.g., `aws/api-key`) |
| Value | Yes | The secret content |
| URL | No | Related URL (e.g., dashboard link) |
| Tags | No | Comma-separated labels |
| Notes | No | Additional context |

### Key Naming

Use hierarchical keys with forward slashes:

```
aws/access-key
aws/secret-key
db/production/password
api/stripe/secret-key
```

This creates a logical structure and enables wildcard patterns in CLI.

### Save the Secret

1. Fill in required fields (Key and Value)
2. Add optional metadata
3. Click **"Create"** or press `Cmd/Ctrl + S`

A success toast confirms creation.

## Editing Secrets

### Start Editing

1. Select the secret you want to edit
2. Click the **edit icon** (pencil) in the detail panel header
3. Or select and press `Cmd/Ctrl + E`

### Edit Form

When editing:
- **Key** is read-only (cannot be changed)
- **Value** can be updated
- **Metadata** (URL, tags, notes) can be modified

### Save Changes

1. Make your changes
2. Click **"Save"** or press `Cmd/Ctrl + S`
3. Success toast confirms the update

### Cancel Editing

To discard changes:
- Click **"Cancel"**
- Or press `Escape`

## Copying Secrets

### Copy to Clipboard

1. Select a secret
2. Click the **copy icon** next to the value
3. Or press `Cmd/Ctrl + C`

### Security Features

When copying:
- Toast shows "Copied! Auto-clears in 30s"
- Clipboard is automatically cleared after 30 seconds
- No need to manually clear the clipboard

:::caution
The clipboard is accessible to all running applications. The 30-second auto-clear helps limit exposure, but be aware of clipboard managers that may persist data.
:::

## Deleting Secrets

### Delete a Secret

1. Select the secret to delete
2. Click the **trash icon** in the detail panel header

### Confirmation Dialog

A confirmation dialog appears:
- Shows the secret key being deleted
- Warns that deletion cannot be undone
- Requires explicit confirmation

### Confirm Deletion

- Click **"Delete"** to confirm
- Click **"Cancel"** to abort

:::warning
Deletion is permanent. The secret cannot be recovered after deletion.
:::

## Working with Metadata

### Tags

Tags help categorize and organize secrets:

**Adding tags:**
1. Edit the secret
2. Enter tags separated by commas: `prod, aws, api`
3. Save changes

**Tag display:**
- Tags appear as badges under the secret key
- First 3 tags shown in list, all tags in detail view

**Use cases:**
- Environment: `prod`, `staging`, `dev`
- Service: `aws`, `stripe`, `github`
- Type: `api`, `database`, `ssh`

### URL Field

Link secrets to their management consoles:

**Examples:**
- `https://console.aws.amazon.com/iam`
- `https://dashboard.stripe.com/apikeys`
- `https://github.com/settings/tokens`

Clicking the URL opens it in your default browser.

### Notes

Add context for future reference:

**Good notes include:**
- What the secret is for
- Who should be contacted for issues
- When it should be rotated
- Any restrictions or usage notes

**Example:**
```
Production API key for payment processing.
Contact: devops@example.com
Rotation: Every 90 days
Scope: Read/write transactions
```

## Refreshing the List

### Manual Refresh

Click the **refresh icon** in the header to reload the secret list.

**When to refresh:**
- After external changes (CLI or another app instance)
- If the list seems out of date
- After network reconnection

### Automatic Updates

The list automatically updates after:
- Creating a new secret
- Editing a secret
- Deleting a secret

## Best Practices

### Naming Conventions

Use consistent, descriptive key names:

```
# Good: Clear hierarchy
aws/production/access-key
db/postgres/main/password
api/stripe/live/secret-key

# Avoid: Ambiguous names
key1
password
token
```

### Metadata Usage

Take advantage of metadata:

- **Tags** for filtering and organization
- **URLs** for quick access to management consoles
- **Notes** for documentation and context

### Security Habits

Maintain good security practices:

1. **Lock when away** - Press `Cmd/Ctrl + L` when leaving
2. **Don't screenshot** - Avoid capturing visible values
3. **Verify before delete** - Double-check the confirmation dialog
4. **Update regularly** - Rotate secrets periodically

## Troubleshooting

### "Key is required" Error

The key field cannot be empty:
- Enter a unique identifier for the secret
- Use alphanumeric characters, `-`, `_`, `/`, `.`

### "Value is required" Error

The value field cannot be empty:
- Enter the secret content
- Whitespace-only values are not allowed

### Secret Not Appearing

If a newly created secret doesn't appear:
1. Click the refresh button
2. Clear the search filter
3. Check if vault is still unlocked

### Edit Button Disabled

The edit button is disabled when:
- No secret is selected
- You're already in edit mode
- You're in create mode

### Copy Not Working

If copying fails:
- Check system clipboard permissions
- Ensure value is not empty
- Try clicking the copy button directly

## Next Steps

- [Keyboard Shortcuts](/docs/guides/desktop/keyboard-shortcuts) - Speed up your workflow
- [Audit Logs](/docs/guides/desktop/audit-logs) - View activity history
- [CLI Guide](/docs/guides/cli/) - Command-line operations
