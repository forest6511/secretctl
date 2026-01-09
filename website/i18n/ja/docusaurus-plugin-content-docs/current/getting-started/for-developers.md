---
title: 開発者向け
description: AI支援開発ワークフローでsecretctlを使い始める。
sidebar_position: 2
---

# はじめに: 開発者向け

このガイドは、Claude Codeなどのコーディングアシスタントとsecretctlを連携させたい、CI/CDでシークレット注入を自動化したい、またはMCPサーバーをプログラムで使用したい開発者向けです。

## 学べること

- 5分でsecretctlをClaude Codeと設定
- AI安全なシークレットアクセスのためにMCPサーバーを使用
- 開発ワークフローでシークレット注入を自動化

## 前提条件

- macOS、Linux、または Windows
- ターミナルアクセス
- Claude Code または同様のAIコーディングアシスタント（オプション）

## ステップ 1: secretctlをインストール

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl

# macOS (Intel)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-amd64
chmod +x secretctl-darwin-amd64
sudo mv secretctl-darwin-amd64 /usr/local/bin/secretctl

# Linux (x86_64)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl

# Windows - GitHub Releasesからダウンロード
# https://github.com/forest6511/secretctl/releases/latest/download/secretctl-windows-amd64.exe
```

インストール確認:

```bash
secretctl --version
```

## ステップ 2: Vaultを初期化

```bash
secretctl init
```

マスターパスワードの作成を求められます。強力なパスワードを選んでください - これがすべてのシークレットを保護します。

:::tip パスワード要件
- 最低8文字（必須）
- 強力なセキュリティには12文字以上を推奨
- 大文字、小文字、数字、記号の組み合わせを推奨
:::

## ステップ 3: APIキーを追加

```bash
# OpenAI APIキー
echo "sk-proj-..." | secretctl set OPENAI_API_KEY

# AWS認証情報
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# データベース認証情報
echo "postgres://user:pass@localhost:5432/db" | secretctl set db/connection_string

# メタデータ付き
echo "ghp_xxxx" | secretctl set github/token \
  --notes "CI用パーソナルアクセストークン" \
  --tags "ci,github" \
  --expires "2025-12-31"
```

## ステップ 4: Claude Codeを設定（MCP連携）

Claude Code設定に追加（`~/.config/claude-code/settings.json` またはVS Code設定）:

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

:::warning セキュリティノート
パスワードは起動時に一度読み取られ、すぐに環境から消去されます。設定ファイルにパスワードを保存することを避けるため、ラッパースクリプトやシェルの安全な環境変数処理の使用を検討してください。
:::

## ステップ 5: Claude Codeで使用

設定後、Claude Codeは以下が可能です:

| 機能 | 例 |
|------|-----|
| シークレット一覧 | 「保存されているAPIキーは何？」 |
| キーの存在確認 | 「OpenAI APIキーはある？」 |
| シークレットでコマンド実行 | 「認証情報を使ってAWSにデプロイ」 |
| マスク値の取得 | 「GitHubトークン（マスク済）を表示」 |

**Claude Codeが（設計上）できないこと**:
- 平文のシークレット値を読む
- シークレットを変更または削除
- Vaultをエクスポート

これが**AI安全設計**セキュリティモデルです - AIエージェントはシークレットを見ることなく使用できます。

## ステップ 6: シークレットでコマンドを実行

`secretctl run`でシークレットを環境変数として注入:

```bash
# 認証情報でAWS CLIを実行
secretctl run -k "aws/*" -- aws s3 ls

# 特定のキーで実行
secretctl run -k OPENAI_API_KEY -k ANTHROPIC_API_KEY -- python my_script.py

# 環境エイリアスを使用
secretctl run --env dev -k "db/*" -- ./migrate.sh
```

## 開発ワークフローの例

### CI/CD連携

```yaml
# GitHub Actionsの例
- name: Deploy
  env:
    SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
  run: |
    secretctl run -k "aws/*" -- ./deploy.sh
```

### ローカル開発

```bash
# すべてのAPIキーで開発サーバーを起動
secretctl run -k "api/*" -- npm run dev

# または必要なフレームワーク用に.envにエクスポート
secretctl export --format env > .env
```

### Vaultのバックアップ

```bash
# 暗号化バックアップを作成
secretctl backup -o ~/backup/secrets-$(date +%Y%m%d).enc

# 必要時に復元
secretctl restore ~/backup/secrets-20241224.enc --dry-run
```

## 次のステップ

- [MCPセキュリティモデル](/docs/guides/mcp/security-model) - AI安全設計がシークレットを守る仕組みを理解
- [利用可能なMCPツール](/docs/guides/mcp/available-tools) - 完全なMCPツールリファレンス
- [環境エイリアス](/docs/guides/mcp/env-aliases) - dev/staging/prod環境を管理
- [CLIリファレンス](/docs/reference/cli-commands) - 完全なCLIコマンドドキュメント

## トラブルシューティング

### Claude Codeがsecretctlを認識しない

1. バイナリパスを確認: `which secretctl`
2. 設定で絶対パスを使用: `/usr/local/bin/secretctl`
3. 設定変更後にClaude Codeを再起動

### "vault not initialized" エラー

先に `secretctl init` を実行するか、`SECRETCTL_VAULT_DIR` 環境変数を確認してください。

### MCPサーバー接続の問題

ログを確認:
```bash
secretctl mcp-server 2>&1 | head -20
```
