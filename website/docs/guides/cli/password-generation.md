---
title: Password Generation
description: Generate secure passwords and passphrases using cryptographically secure random generation.
sidebar_position: 5
---

# Password Generation

The `generate` command creates cryptographically secure random passwords using Go's `crypto/rand` package.

## Prerequisites

- [secretctl installed](/docs/getting-started/installation)

## Basic Usage

```bash
secretctl generate [flags]
```

### Default Password

Generate a 24-character password with all character types:

```bash
secretctl generate
```

**Output:**
```
x9K#mP2!nQ7@wR4$tY6&jL
```

The default password includes:
- Lowercase letters (a-z)
- Uppercase letters (A-Z)
- Numbers (0-9)
- Symbols (!@#$%^&*()_+-=[]{}|;:,.<>?)

## Command Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--length` | `-l` | Password length (8-256) | 24 |
| `--count` | `-n` | Number of passwords (1-100) | 1 |
| `--no-symbols` | | Exclude symbols | false |
| `--no-numbers` | | Exclude numbers | false |
| `--no-uppercase` | | Exclude uppercase | false |
| `--no-lowercase` | | Exclude lowercase | false |
| `--exclude` | | Characters to exclude | "" |
| `--copy` | `-c` | Copy to clipboard | false |

## Customizing Password Length

### Longer Passwords

```bash
# 32-character password for high-security use
secretctl generate -l 32

# 64-character password for encryption keys
secretctl generate -l 64
```

### Shorter Passwords

```bash
# Minimum 8-character password
secretctl generate -l 8
```

:::caution
Passwords shorter than 16 characters may be vulnerable to brute-force attacks. Use longer passwords for sensitive accounts.
:::

## Generating Multiple Passwords

Generate several passwords at once:

```bash
# Generate 5 passwords
secretctl generate -n 5
```

**Output:**
```
kL9#mN2!pQ7@rS4$
tU6&vW8*xY0^zA2%
bC4(dE6)fG8+hI0-
jK2=lM4[nO6]pQ8{
rS0|tU2;vW4:xY6,
```

Useful for:
- Creating passwords for team onboarding
- Generating test credentials
- Password rotation across services

## Character Set Customization

### Alphanumeric Only

For systems that don't accept symbols:

```bash
secretctl generate --no-symbols
```

### Letters Only

For codes or identifiers:

```bash
secretctl generate --no-symbols --no-numbers
```

### Numbers Only

For PINs or numeric codes:

```bash
secretctl generate --no-symbols --no-uppercase --no-lowercase -l 6
```

**Output:**
```
847291
```

### Uppercase Only

For system codes or license keys:

```bash
secretctl generate --no-symbols --no-numbers --no-lowercase
```

## Excluding Ambiguous Characters

Remove characters that look similar to avoid confusion:

```bash
# Exclude 0/O, 1/l/I which are easy to confuse
secretctl generate --exclude "0O1lI"
```

This is useful for:
- Passwords that will be read aloud or typed manually
- QR codes and printed materials
- Reducing user error in manual entry

### Common Exclusion Sets

```bash
# Exclude ambiguous characters
secretctl generate --exclude "0O1lI"

# Exclude characters problematic in shells
secretctl generate --exclude '$`"'\''\\!'

# Exclude XML/HTML special characters
secretctl generate --exclude "<>&'\""
```

## Clipboard Integration

Copy the generated password directly to clipboard:

```bash
secretctl generate -c
```

**Output:**
```
kL9#mN2!pQ7@rS4$tU6&vW8*
WARNING: Password copied to clipboard is accessible by all processes
         Clipboard will not be automatically cleared. Overwrite manually when done.
Password copied to clipboard
```

### Platform Support

| Platform | Clipboard Tool |
|----------|---------------|
| macOS | `pbcopy` (built-in) |
| Linux | `xclip` or `xsel` |
| Windows | `clip` (built-in) |

:::caution
The clipboard is accessible by all running processes. Clear it after use by copying other content.
:::

### Linux Setup

Install clipboard support on Linux:

```bash
# Debian/Ubuntu
sudo apt install xclip

# Fedora
sudo dnf install xclip

# Arch
sudo pacman -S xclip
```

## Practical Examples

### Database Password

Strong 32-character password for database:

```bash
secretctl generate -l 32 | secretctl set DB_PASSWORD
```

### API Key Format

Generate alphanumeric strings suitable for API keys:

```bash
secretctl generate -l 40 --no-symbols
```

### WiFi Password

Human-readable password for sharing:

```bash
secretctl generate -l 16 --exclude "0O1lI"
```

### SSH Key Passphrase

Strong passphrase for SSH key protection:

```bash
secretctl generate -l 24 -c
```

### Team Onboarding

Generate temporary passwords for new team members:

```bash
# Generate 10 passwords for onboarding
secretctl generate -n 10 -l 16 --no-symbols
```

### AWS Secret Key Format

Simulate AWS-style secret access keys:

```bash
secretctl generate -l 40 --no-symbols
```

### Backup Encryption Key

Generate high-entropy key for backup encryption:

```bash
secretctl generate -l 64 --no-symbols
```

## Password Strength

### Entropy Calculation

Password strength is measured in bits of entropy:

| Configuration | Charset Size | 24-char Entropy |
|--------------|--------------|-----------------|
| All characters | 94 | ~157 bits |
| No symbols | 62 | ~143 bits |
| Letters only | 52 | ~137 bits |
| Alphanumeric lowercase | 36 | ~124 bits |

Higher entropy = stronger password.

### Recommended Lengths

| Use Case | Minimum Length | Recommended |
|----------|----------------|-------------|
| Personal accounts | 12 | 16+ |
| Database passwords | 24 | 32+ |
| API keys/tokens | 32 | 40+ |
| Encryption keys | 32 | 64+ |
| Master passwords | 16 | 24+ |

## Combining with Other Commands

### Generate and Store

```bash
# Generate and store a new password
secretctl generate | secretctl set SERVICE_PASSWORD \
  --notes="Auto-generated on $(date)" \
  --expires="90d"
```

### Generate for Export

```bash
# Generate multiple passwords and export
secretctl generate -n 5 | while read pw; do
  echo "Temp password: $pw"
done
```

### Rotate Password

```bash
# Generate new password and update existing secret
secretctl generate -l 32 | secretctl set DB_PASSWORD \
  --notes="Rotated on $(date +%Y-%m-%d)"
```

## Security Considerations

### Cryptographic Security

secretctl uses Go's `crypto/rand` package which:
- Uses the operating system's cryptographic random number generator
- Is suitable for security-sensitive applications
- Never uses pseudorandom generators like `math/rand`

### Clipboard Security

When using `--copy`:
- The password is accessible to all running processes
- The clipboard is not automatically cleared
- Always overwrite the clipboard after use

### Avoid Logging

Be careful not to log generated passwords:

```bash
# Bad: Password appears in shell history
echo $(secretctl generate)

# Good: Pipe directly or use clipboard
secretctl generate | secretctl set MY_SECRET
secretctl generate -c
```

### Shell History

The `generate` command itself is safe for shell history as it doesn't contain the password in the command line.

## Troubleshooting

### "clipboard tool not found" Error

On Linux, install a clipboard tool:

```bash
# Install xclip
sudo apt install xclip

# Or install xsel
sudo apt install xsel
```

### "password length must be at least 8" Error

Minimum password length is 8 characters:

```bash
# This fails
secretctl generate -l 4

# Use minimum of 8
secretctl generate -l 8
```

### "character set is empty" Error

Ensure at least one character type is enabled:

```bash
# This fails - all character types excluded
secretctl generate --no-symbols --no-numbers --no-uppercase --no-lowercase

# Include at least one type
secretctl generate --no-symbols --no-numbers
```

### Clipboard Not Working

Verify clipboard command is available:

```bash
# macOS
which pbcopy

# Linux
which xclip || which xsel

# Windows (PowerShell)
Get-Command clip
```

## Next Steps

- [Managing Secrets](/docs/guides/cli/managing-secrets) - Store generated passwords
- [Running Commands](/docs/guides/cli/running-commands) - Use secrets in commands
- [CLI Commands Reference](/docs/reference/cli-commands) - Complete command reference
