---
title: MCP 連携
description: MCP を使用して secretctl を AI エージェントと連携。
sidebar_position: 1
---

# MCP 連携

secretctl には、Claude Code、Codex CLI、その他の MCP 対応ツールなどの AI コーディングアシスタントと安全に連携するための組み込み MCP（Model Context Protocol）サーバーが含まれています。

## 概要

MCP サーバーにより、AI アシスタントは**実際のシークレット値を見ることなく**シークレットを使用できます。これは [AI安全設計セキュリティモデル](/docs/guides/mcp/security-model) によって実現されています。

## クイックスタート

### 1. ポリシーファイルを作成（必須）

MCP サーバーを起動する前に、AI が実行できるコマンドを制御するポリシーファイルを作成します:

```bash
mkdir -p ~/.secretctl
cat > ~/.secretctl/mcp-policy.yaml << 'EOF'
version: 1
default_action: deny
allowed_commands:
  - aws
  - gcloud
  - kubectl
EOF
```

### 2. AI ツールを設定

Claude Code の設定（`~/.claude.json`）に追加:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

詳細な設定は [Claude Code セットアップ](/docs/guides/mcp/claude-code-setup) を参照してください。

## 機能

- **設計上安全**: AI エージェントは平文のシークレットを見ることがない
- **ポリシーベースのアクセス制御**: AI が実行できるコマンドを定義
- **出力サニタイズ**: コマンド出力から漏洩したシークレットを自動的に編集
- **環境エイリアス**: dev/staging/prod をシームレスに切り替え

## 詳細情報

- [セキュリティモデル（AI安全設計）](/docs/guides/mcp/security-model) - シークレットがどのように保護されるか
- [Claude Code セットアップ](/docs/guides/mcp/claude-code-setup) - 詳細なセットアップガイド
- [利用可能なツール](/docs/guides/mcp/available-tools) - MCP ツールリファレンス
- [環境エイリアス](/docs/guides/mcp/env-aliases) - マルチ環境設定
