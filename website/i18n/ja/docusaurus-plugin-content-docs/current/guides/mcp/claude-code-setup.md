---
title: Claude Code セットアップ
description: secretctl を Claude Code と設定。
sidebar_position: 3
---

# Claude Code セットアップ

このガイドでは、secretctl MCP サーバーを Claude Code やその他の MCP 対応 AI ツールと設定する方法を説明します。

## 前提条件

1. secretctl がインストールされ初期化済み（`secretctl init`）
2. 少なくとも1つのシークレットが保存済み（`secretctl secret add`）
3. ポリシーファイルが作成済み（下記参照）

## ステップ 1: ポリシーファイルを作成

`~/.secretctl/mcp-policy.yaml` を作成:

```bash
mkdir -p ~/.secretctl
cat > ~/.secretctl/mcp-policy.yaml << 'EOF'
version: 1
default_action: deny
allowed_commands:
  # クラウド CLI
  - aws
  - gcloud
  - az

  # コンテナツール
  - kubectl
  - docker
  - helm

  # データベースツール
  - psql
  - mysql
  - mongosh

denied_commands:
  - env
  - printenv
  - set
  - export
EOF
```

## ステップ 2: Claude Code を設定

`~/.claude.json` に追加:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp", "serve"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

:::caution
`/path/to/secretctl` を実際の secretctl バイナリのパスに置き換えてください。
`which secretctl` で見つけることができます。
:::

### Homebrew インストールの場合

Homebrew でインストールした場合:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/opt/homebrew/bin/secretctl",
      "args": ["mcp", "serve"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

## ステップ 3: Codex CLI を設定

OpenAI Codex CLI の場合、`~/.codex/config.yaml` に追加:

```yaml
mcpServers:
  secretctl:
    command: /path/to/secretctl
    args:
      - mcp
      - serve
    env:
      SECRETCTL_PASSWORD: your-master-password
```

## ステップ 4: 連携をテスト

1. Claude Code または MCP クライアントを起動
2. Claude にシークレットの一覧を依頼:
   ```
   "保存されているシークレットを一覧表示して"
   ```
3. Claude はシークレットキー（値ではなく）で応答するはずです

## 環境変数

| 変数 | 説明 | デフォルト |
|------|------|----------|
| `SECRETCTL_PASSWORD` | Vault 用のマスターパスワード | （必須） |
| `SECRETCTL_VAULT_DIR` | カスタム Vault ディレクトリ | `~/.secretctl` |

## トラブルシューティング

### サーバーが起動しない

**エラー: "no password provided"**

```bash
# パスワードが正しく設定されているか確認
export SECRETCTL_PASSWORD=your-master-password
secretctl mcp serve
```

**エラー: "failed to unlock vault"**

- パスワードが正しいか確認
- Vault が存在するか確認（`secretctl list` が動作するか）

### コマンドが拒否される

**エラー: "command not allowed by policy"**

1. `~/.secretctl/mcp-policy.yaml` が存在するか確認
2. コマンドを `allowed_commands` に追加
3. Claude Code を再起動して MCP サーバーを再読み込み

### 接続の問題

Claude Code が接続できない場合:

1. secretctl のパスが正しいか確認
2. secretctl が実行可能か確認
3. 手動でテスト: `secretctl mcp serve`

## セキュリティ推奨事項

1. **シークレットをコミットしない** - パスワード付きの Claude 設定を git に追加しない
2. **環境変数を使用** - パスワード用にシェルラッパーの使用を検討
3. **権限を制限** - 必要なコマンドのみを許可
4. **定期的にレビュー** - `secretctl audit list` で異常なアクティビティを確認

## ワークフロー例

設定後、Claude に以下を依頼できます:

```
"aws/* 認証情報を使って 'aws s3 ls' を実行して"

"k8s/prod/* シークレットを使って Kubernetes にデプロイして"

"db/prod/password でデータベースに接続して"
```

Claude は `secret_run` を使用して、実際の値を見ることなくシークレットを環境変数として注入してこれらのコマンドを実行します。
