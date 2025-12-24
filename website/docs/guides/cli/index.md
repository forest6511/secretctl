---
title: CLI Guide
description: Master the secretctl command-line interface.
sidebar_position: 1
---

# CLI Guide

Learn to use secretctl from the command line.

## Quick Reference

```bash
# Initialize vault
secretctl init

# Manage secrets
echo "value" | secretctl set KEY
secretctl get KEY
secretctl list
secretctl delete KEY

# Run commands with secrets
secretctl run -k KEY -- your-command

# Export secrets
secretctl export -o .env

# Generate passwords
secretctl generate
```

## Guides

- [Running Commands](/docs/guides/cli/running-commands) - Execute commands with secrets as environment variables
- [Password Generation](/docs/guides/cli/password-generation) - Generate secure random passwords

## Reference

For complete command documentation, see the [CLI Commands Reference](/docs/reference/cli-commands).
