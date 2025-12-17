---
title: Installation
description: Install secretctl on macOS, Linux, or Windows.
sidebar_position: 2
---

# Install secretctl

secretctl is distributed as a single binary with no dependencies.

## macOS

### Homebrew (Recommended)

```bash
brew install forest6511/tap/secretctl
```

### Manual Download

```bash
# Download the latest release
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64.tar.gz

# Extract and install
tar -xzf secretctl-darwin-arm64.tar.gz
sudo mv secretctl /usr/local/bin/
```

## Linux

### Download Binary

```bash
# For x86_64
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64.tar.gz
tar -xzf secretctl-linux-amd64.tar.gz
sudo mv secretctl /usr/local/bin/

# For ARM64
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-arm64.tar.gz
tar -xzf secretctl-linux-arm64.tar.gz
sudo mv secretctl /usr/local/bin/
```

## Windows

### Download Binary

1. Download from [GitHub Releases](https://github.com/forest6511/secretctl/releases)
2. Extract `secretctl-windows-amd64.zip`
3. Add to your PATH

## Verify Installation

```bash
secretctl version
```

Expected output:

```
secretctl version 1.0.0
```

## Desktop App

The desktop app is available for macOS, Linux, and Windows:

1. Download from [GitHub Releases](https://github.com/forest6511/secretctl/releases)
2. Install the appropriate package for your platform
3. Launch the app

## Next Steps

- [Quick Start](./quick-start) - Create your first secret
- [Core Concepts](./concepts) - Learn about vaults and encryption
