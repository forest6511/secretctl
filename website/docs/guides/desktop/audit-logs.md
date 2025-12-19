---
title: Audit Logs
description: View, filter, and export audit logs for security compliance.
sidebar_position: 4
---

# Audit Logs

The audit log viewer provides a visual interface for reviewing all vault activity. Every action is recorded with tamper-evident cryptographic chaining.

## Prerequisites

- [Desktop app installed](/docs/guides/desktop/)
- Vault unlocked

## Accessing Audit Logs

### Open Audit Log

1. Click the **clipboard icon** in the header bar
2. Or navigate from the secrets page

The audit log page displays activity history with filtering options.

## Understanding the Interface

### Header Section

The header displays:

- **Back button** - Return to secrets page
- **Verify Chain** - Check audit log integrity
- **Export buttons** - Download as CSV or JSON
- **Chain status** - Current verification result
- **Statistics** - Total, success, and failure counts

### Filter Bar

Available filters:

| Filter | Options | Purpose |
|--------|---------|---------|
| Action | All, Get, Set, Delete, List, Unlock, Lock, Init | Filter by operation type |
| Source | All, CLI, MCP, Desktop | Filter by access method |
| Key | Text input | Search by secret key |
| Date Range | Start and end dates | Filter by time period |

### Log Table

Each row shows:

| Column | Description |
|--------|-------------|
| Timestamp | When the action occurred |
| Action | What operation was performed |
| Source | How the vault was accessed |
| Key | Which secret was affected (if applicable) |
| Status | Success or failure indicator |

## Filtering Logs

### By Action Type

Filter to see specific operations:

1. Click the **Action** dropdown
2. Select an action type
3. Click **Apply**

**Action types:**

| Action | Description |
|--------|-------------|
| `secret.get` | Secret value was retrieved |
| `secret.set` | Secret was created or updated |
| `secret.delete` | Secret was deleted |
| `secret.list` | Secret list was viewed |
| `auth.unlock` | Vault was unlocked |
| `auth.lock` | Vault was locked |
| `vault.init` | Vault was initialized |

### By Source

See how the vault was accessed:

| Source | Description |
|--------|-------------|
| CLI | Command-line interface |
| MCP | AI tool integration |
| Desktop | Desktop application |

This helps distinguish between different access methods.

### By Secret Key

Search for activity on a specific secret:

1. Enter the key name (or partial match)
2. Click **Apply**

Useful for investigating who accessed a particular secret.

### By Date Range

Filter to a specific time period:

1. Set **Start date** (beginning of range)
2. Set **End date** (end of range)
3. Click **Apply**

Helpful for compliance audits or incident investigation.

### Clear Filters

Remove all filters to see full history:

1. Click **Clear**
2. All filters reset to default

## Chain Verification

### What is Chain Verification?

Each audit log entry is cryptographically linked to the previous entry using HMAC-SHA256. This creates a tamper-evident chain:

- Modifying any entry breaks the chain
- Deleting entries breaks the chain
- Order changes break the chain

### Verify the Chain

1. Click **Verify Chain** button
2. Wait for verification to complete
3. Check the status indicator

### Verification Results

| Status | Icon | Meaning |
|--------|------|---------|
| Verified | Green checkmark | Chain is intact, no tampering detected |
| Invalid | Red X | Chain is broken, possible tampering |
| Checking | Gray spinner | Verification in progress |

:::warning
If chain verification fails, your audit logs may have been tampered with. This is a serious security concern that should be investigated immediately.
:::

## Viewing Details

### Open Detail Modal

Click any log entry to see full details:

- **Timestamp** - Full date and time
- **Action** - Operation performed
- **Source** - Access method
- **Key** - Secret key (if applicable)
- **Status** - Success or failure
- **Error** - Error message (if failed)

### Close Detail Modal

- Click outside the modal
- Click the X button
- Press `Escape`

## Exporting Logs

### Export to CSV

For spreadsheet analysis:

1. Apply any desired filters
2. Click **CSV** button
3. File downloads automatically

**CSV format:**
```csv
Timestamp,Action,Source,Key,Status,Error
2025-01-15T10:30:00Z,secret.get,cli,API_KEY,Success,
2025-01-15T10:25:00Z,secret.set,desktop,DB_PASSWORD,Success,
```

### Export to JSON

For programmatic processing:

1. Apply any desired filters
2. Click **JSON** button
3. File downloads automatically

**JSON format:**
```json
[
  {
    "timestamp": "2025-01-15T10:30:00Z",
    "action": "secret.get",
    "source": "cli",
    "key": "API_KEY",
    "success": true,
    "error": ""
  }
]
```

### Export Tips

- Filters apply to exports (export only what you need)
- Large exports may take a moment
- Files are named `audit-logs.csv` or `audit-logs.json`

## Pagination

### Navigate Pages

For large audit histories:

- **Prev** - Go to previous page
- **Next** - Go to next page
- Page indicator shows current position

Each page displays 20 entries.

### Page Information

The footer shows:
- Current range (e.g., "Showing 1-20 of 150")
- Page number (e.g., "Page 1 of 8")

## Use Cases

### Security Incident Investigation

When investigating a potential breach:

1. Filter by the **date range** of the incident
2. Look for unusual **source** patterns
3. Check for failed **auth.unlock** attempts
4. Export evidence as JSON

### Compliance Audit

For regulatory compliance:

1. Set **date range** to audit period
2. Export complete log as CSV
3. Provide to compliance team
4. Document chain verification status

### Access Review

Review who accessed what:

1. Filter by specific **key**
2. Review all access by **source**
3. Identify patterns or anomalies
4. Document findings

### Troubleshooting Failures

When operations fail:

1. Filter by **status** (failures)
2. Check the **error** message in detail view
3. Identify the **source** and **action**
4. Address the root cause

## Understanding Actions

### Authentication Actions

| Action | Triggered By |
|--------|-------------|
| `vault.init` | First-time vault creation |
| `auth.unlock` | Entering master password |
| `auth.lock` | Manual lock or auto-lock |

### Secret Actions

| Action | Triggered By |
|--------|-------------|
| `secret.list` | Viewing secret list |
| `secret.get` | Viewing secret value |
| `secret.set` | Creating or updating |
| `secret.delete` | Removing a secret |

### Source Identification

| Source | When Used |
|--------|-----------|
| `cli` | secretctl command-line tool |
| `mcp` | AI coding assistant integration |
| `ui` | Desktop application |

## Statistics

### Understanding Stats

The header shows aggregated statistics:

- **Total**: All recorded actions
- **Success**: Successfully completed actions
- **Failure**: Failed actions

### Interpreting Numbers

**High failure count** may indicate:
- Incorrect password attempts
- Permission issues
- System errors

**Unexpected sources** may indicate:
- Unauthorized access attempts
- Misconfigured integrations

## Best Practices

### Regular Reviews

Schedule periodic audit reviews:
- Weekly: Quick scan for anomalies
- Monthly: Full export and analysis
- Quarterly: Compliance documentation

### Verify Chain Regularly

Run chain verification:
- After any suspicious activity
- Before compliance audits
- After system maintenance

### Export for Backup

Keep offline copies:
- Export monthly archives
- Store in secure location
- Include verification results

### Monitor Sources

Track access patterns:
- Unexpected MCP access may indicate AI tool misconfiguration
- CLI access during off-hours may need investigation
- Desktop access from unusual patterns worth reviewing

## Troubleshooting

### No Logs Displayed

**Possible causes:**
- Filters too restrictive - click **Clear**
- New vault with no history
- Page at end of results

### Chain Verification Fails

**Immediate actions:**
1. Do not trust the audit log
2. Investigate potential tampering
3. Check for corruption
4. Contact security team

### Export Not Working

**Try these fixes:**
1. Check browser download settings
2. Reduce filter scope
3. Try different export format
4. Check available disk space

### Slow Performance

**For large logs:**
1. Use more specific filters
2. Narrow date range
3. Wait for pagination to load
4. Consider exporting and analyzing offline

## Next Steps

- [Managing Secrets](/docs/guides/desktop/managing-secrets) - Secret operations guide
- [Desktop App Overview](/docs/guides/desktop/) - Full feature guide
- [Configuration Reference](/docs/reference/configuration) - Audit log retention settings
