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
# Apple Silicon (M1/M2/M3)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl

# Intel Mac
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-amd64
chmod +x secretctl-darwin-amd64
sudo mv secretctl-darwin-amd64 /usr/local/bin/secretctl
```

## Linux

### Download Binary

```bash
# For x86_64
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl

# For ARM64
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-arm64
chmod +x secretctl-linux-arm64
sudo mv secretctl-linux-arm64 /usr/local/bin/secretctl
```

## Windows

### Download Binary

1. Download `secretctl-windows-amd64.exe` from [GitHub Releases](https://github.com/forest6511/secretctl/releases/latest)
2. Rename to `secretctl.exe`
3. Add to your PATH

## Verify Installation

```bash
secretctl --help
```

You should see the list of available commands.

## Desktop App

The desktop app provides a GUI alternative to the CLI. Currently, you need to build it from source:

```bash
cd desktop
wails build
```

See [Desktop App Guide](/docs/guides/desktop) for details.

## Next Steps

- [Quick Start](./quick-start) - Create your first secret
- [Core Concepts](./concepts) - Learn about vaults and encryption
