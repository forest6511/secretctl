---
title: 設定
description: secretctl の設定オプション、環境変数、ファイル構成。
sidebar_position: 3
---

# 設定リファレンス

secretctl の設定オプション、環境変数、ファイル構成の完全リファレンス。

## 環境変数

### Vault 設定

| 変数 | 説明 | デフォルト |
|------|------|----------|
| `SECRETCTL_VAULT_DIR` | Vault ファイルを格納するディレクトリ | `~/.secretctl` |
| `SECRETCTL_PASSWORD` | Vault 操作用のマスターパスワード | (なし) |

**使用例:**

```bash
# カスタム Vault ディレクトリを使用
export SECRETCTL_VAULT_DIR=/path/to/custom/vault
secretctl list

# 環境変数でパスワードを指定（非対話的）
SECRETCTL_PASSWORD=mypassword secretctl get API_KEY

# MCP サーバーは認証に SECRETCTL_PASSWORD を使用
SECRETCTL_PASSWORD=mypassword secretctl mcp-server
```

### セキュリティに関する注意

- `SECRETCTL_PASSWORD` は読み取り後に環境から自動的に消去されます
- シェルプロファイルや永続的な環境変数に `SECRETCTL_PASSWORD` を設定しないでください
- MCP サーバーでは、MCP クライアント設定でプロセスレベルの環境変数を使用してください

---

## ファイル構成

### Vault ディレクトリ

デフォルトの Vault ディレクトリは `~/.secretctl` です。構成は以下の通りです:

```
~/.secretctl/
├── vault.salt       # 暗号化ソルト (16バイト)
├── vault.meta       # Vault メタデータ (暗号化)
├── vault.db         # SQLite データベース (暗号化)
├── vault.lock       # 同時アクセス用ロックファイル
├── audit/           # 監査ログディレクトリ
│   └── *.jsonl      # JSON Lines 監査ログファイル
└── mcp-policy.yaml  # MCP サーバーポリシー (オプション)
```

### ファイルパーミッション

すべてのファイルは安全なパーミッションで作成されます:

| ファイル/ディレクトリ | パーミッション | 説明 |
|---------------------|---------------|------|
| `~/.secretctl/` | `0700` | オーナーのみアクセス可能 |
| `vault.salt` | `0600` | ソルトファイル (オーナーのみ読み書き) |
| `vault.meta` | `0600` | メタデータファイル (オーナーのみ読み書き) |
| `vault.db` | `0600` | データベースファイル (オーナーのみ読み書き) |
| `mcp-policy.yaml` | `0600` | ポリシーファイル (MCP サーバーに必要) |
| `audit/` | `0700` | 監査ログディレクトリ |

**重要:** MCP ポリシーファイルは `0600` パーミッションで、現在のユーザーが所有している必要があります。セキュリティ上の理由から、シンボリックリンクは許可されていません。

---

## MCP ポリシー設定

MCP サーバーを設定するには `~/.secretctl/mcp-policy.yaml` を作成します:

```yaml
version: 1
default_action: deny

# 常にブロックされるコマンド（ハードコード）
# - env, printenv, set, export, cat /proc/*/environ

# ユーザー定義の拒否コマンド（最初にチェック）
denied_commands:
  - rm
  - mv
  - sudo

# 許可されたコマンド（次にチェック）
allowed_commands:
  - aws
  - gcloud
  - kubectl
  - curl
  - wget
  - ./deploy.sh

# キー変換用の環境エイリアス
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
    - pattern: "api/*"
      target: "staging/api/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
    - pattern: "api/*"
      target: "prod/api/*"
```

### ポリシーフィールド

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `version` | integer | はい | ポリシーバージョン (必ず `1`) |
| `default_action` | string | いいえ | デフォルトアクション: `allow` または `deny` (デフォルト: `deny`) |
| `denied_commands` | string[] | いいえ | 常にブロックするコマンド |
| `allowed_commands` | string[] | いいえ | 許可するコマンド |
| `env_aliases` | map | いいえ | 環境エイリアスマッピング |

### ポリシー評価順序

1. **ハードコードされた拒否**: `env`, `printenv`, `set`, `export`, `cat /proc/*/environ`
2. **ユーザーの `denied_commands`**: 明示的にブロックされたコマンド
3. **ユーザーの `allowed_commands`**: 明示的に許可されたコマンド
4. **`default_action`**: フォールバック (デフォルト: `deny`)

### 環境エイリアス

環境エイリアスにより、環境ごとに異なるシークレットキーマッピングが可能になります:

```yaml
env_aliases:
  dev:
    - pattern: "db/*"      # db/host, db/password などにマッチ
      target: "dev/db/*"   # dev/db/host, dev/db/password に変換
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

**使用方法:**

```bash
# CLI: --env フラグを使用
secretctl run --env=prod -k "db/*" -- ./deploy.sh

# MCP: env パラメータを使用
{
  "keys": ["db/*"],
  "command": "./deploy.sh",
  "env": "prod"
}
```

---

## 検証制限

### マスターパスワード

| 制約 | 値 |
|------|-----|
| 最小長 | 8文字 |
| 最大長 | 128文字 |

### シークレットキー

| 制約 | 値 |
|------|-----|
| 最小長 | 1文字 |
| 最大長 | 256文字 |
| 許可される文字 | 英数字、`-`、`_`、`/`、`.` |

### シークレット値

| 制約 | 値 |
|------|-----|
| 最大サイズ | 1 MB (1,048,576 バイト) |

### メタデータ

| 制約 | 値 |
|------|-----|
| 最大メモサイズ | 10 KB (10,240 バイト) |
| 最大 URL 長 | 2,048 文字 |
| 最大タグ数 | 10 個 |
| 最大タグ長 | 64 文字 |

---

## ロック解除クールダウン

ブルートフォース攻撃から保護するため、secretctl はロック解除失敗後に段階的なクールダウンを実装しています:

| 失敗回数 | クールダウン時間 |
|---------|-----------------|
| 5 | 30秒 |
| 10 | 5分 |
| 20 | 30分 |

クールダウンカウンターはロック解除成功後にリセットされます。

---

## ディスク容量要件

secretctl は利用可能なディスク容量を監視します:

| しきい値 | アクション |
|---------|----------|
| 空き容量 < 10 MB | Vault 操作がブロック |
| 使用率 > 90% | 警告を表示 |
| 空き容量 < 1 MB | 監査ログがブロック |

---

## 暗号化パラメータ

### 鍵導出 (Argon2id)

| パラメータ | 値 |
|-----------|-----|
| アルゴリズム | Argon2id |
| メモリ | 64 MB |
| イテレーション | 3 |
| 並列度 | 4 スレッド |
| ソルト長 | 16 バイト (128ビット) |
| 出力長 | 32 バイト (256ビット) |

これらのパラメータは高セキュリティアプリケーション向けの OWASP 推奨に従っています。

### 暗号化 (AES-256-GCM)

| パラメータ | 値 |
|-----------|-----|
| アルゴリズム | AES-256-GCM |
| 鍵長 | 256 ビット |
| ノンス長 | 12 バイト (96ビット) |
| タグ長 | 16 バイト (128ビット) |

### HMAC (監査チェーン)

| パラメータ | 値 |
|-----------|-----|
| アルゴリズム | HMAC-SHA256 |
| 鍵導出 | HKDF-SHA256 |
| チェーン検証 | 順次検証 |

---

## シェル補完

CLI 体験を向上させるためにシェル補完をインストール:

### Bash

```bash
secretctl completion bash > /etc/bash_completion.d/secretctl
# またはユーザーレベルのインストール
secretctl completion bash > ~/.local/share/bash-completion/completions/secretctl
```

### Zsh

```bash
secretctl completion zsh > "${fpath[1]}/_secretctl"
# またはカスタムディレクトリを指定
secretctl completion zsh > ~/.zsh/completions/_secretctl
```

### Fish

```bash
secretctl completion fish > ~/.config/fish/completions/secretctl.fish
```

### PowerShell

```powershell
secretctl completion powershell | Out-String | Invoke-Expression
# またはプロファイルに保存
secretctl completion powershell >> $PROFILE
```

---

## Claude Code 連携

`~/.claude.json` で secretctl MCP サーバーを設定:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

**セキュリティ上の考慮事項:**

- `SECRETCTL_PASSWORD` は安全に保管してください（キーチェーンやシークレットマネージャーの使用を検討）
- MCP サーバーは AI エージェントにメタデータとマスクされた値のみを公開します
- `mcp-policy.yaml` を設定して実行できるコマンドを制限してください

詳細なセットアップ手順は [MCP 連携ガイド](/docs/guides/mcp/) を参照してください。
