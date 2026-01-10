---
title: インストール
description: macOS、Linux、Windowsにsecretctlをインストール。
sidebar_position: 2
---

# secretctl をインストール

secretctlは依存関係のない単一バイナリとして配布されています。

## macOS

### Homebrew（推奨）

```bash
brew install forest6511/tap/secretctl
```

### 手動ダウンロード

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

### バイナリダウンロード

```bash
# x86_64の場合
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl

# ARM64の場合
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-arm64
chmod +x secretctl-linux-arm64
sudo mv secretctl-linux-arm64 /usr/local/bin/secretctl
```

## Windows

### バイナリダウンロード

1. [GitHub Releases](https://github.com/forest6511/secretctl/releases/latest) から `secretctl-windows-amd64.exe` をダウンロード
2. `secretctl.exe` にリネーム
3. PATHに追加

## インストール確認

```bash
secretctl --help
```

利用可能なコマンド一覧が表示されます。

## デスクトップアプリ

デスクトップアプリはCLIの代替としてGUIを提供します。現在、ソースからビルドする必要があります：

```bash
cd desktop
wails build
```

詳細は[デスクトップアプリガイド](/docs/guides/desktop)を参照してください。

## 次のステップ

- [クイックスタート](./quick-start) - 最初のシークレットを作成
- [コアコンセプト](./concepts) - Vaultと暗号化について学ぶ
