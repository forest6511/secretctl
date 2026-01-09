---
title: フィールド名リファレンス
description: マルチフィールドシークレットの標準フィールド名とテンプレート。
sidebar_position: 4
---

# フィールド名リファレンス

secretctl はマルチフィールドシークレットを一般的なユースケース向けの定義済みテンプレートでサポートしています。このリファレンスでは、標準フィールド名、機密設定、環境変数バインディングについて説明します。

## テンプレート概要

| テンプレート | ユースケース | フィールド数 | デフォルトバインディング |
|-------------|-------------|-------------|------------------------|
| **Login** | Web サイト認証情報 | 2 | なし |
| **Database** | データベース接続 | 5 | PostgreSQL (`PGHOST` など) |
| **API** | API 認証情報 | 2 | `API_KEY`, `API_SECRET` |
| **SSH** | SSH 認証 | 2 | なし |

---

## Login テンプレート

Web サイトやサービスのログイン認証情報を保存します。

### フィールド

| フィールド | 機密 | 説明 |
|-----------|------|------|
| `username` | いいえ | ユーザー名またはメールアドレス |
| `password` | はい | アカウントパスワード |

### 環境バインディング

デフォルトバインディングなし。必要に応じてカスタムバインディングを追加:

```bash
secretctl set github/login \
  --field username=myuser \
  --field password=secret123 \
  --sensitive password \
  --binding GITHUB_USER=username \
  --binding GITHUB_TOKEN=password
```

### デスクトップアプリ

新しいシークレット作成時に **Login** テンプレートを選択すると、これらのフィールドが自動設定されます。

---

## Database テンプレート

データベース接続認証情報を保存します。PostgreSQL 環境変数向けに事前設定されています。

### フィールド

| フィールド | 機密 | 説明 |
|-----------|------|------|
| `host` | いいえ | データベースサーバーのホスト名 |
| `port` | いいえ | データベースサーバーのポート |
| `username` | いいえ | データベースユーザー名 |
| `password` | はい | データベースパスワード |
| `database` | いいえ | データベース名 |

### 環境バインディング

| 環境変数 | マップ先 |
|---------|---------|
| `PGHOST` | `host` |
| `PGPORT` | `port` |
| `PGUSER` | `username` |
| `PGPASSWORD` | `password` |
| `PGDATABASE` | `database` |

### CLI 例

```bash
# PostgreSQL バインディング付きでデータベースシークレットを作成
secretctl set db/prod \
  --field host=db.example.com \
  --field port=5432 \
  --field username=admin \
  --field password=secret123 \
  --field database=myapp \
  --sensitive password \
  --binding PGHOST=host \
  --binding PGPORT=port \
  --binding PGUSER=username \
  --binding PGPASSWORD=password \
  --binding PGDATABASE=database

# 注入された認証情報で psql を実行
secretctl run -k db/prod -- psql
```

### MySQL/MariaDB

MySQL の場合はカスタムバインディングを使用:

```bash
secretctl set db/mysql \
  --field host=mysql.example.com \
  --field port=3306 \
  --field username=root \
  --field password=secret \
  --field database=mydb \
  --sensitive password \
  --binding MYSQL_HOST=host \
  --binding MYSQL_TCP_PORT=port \
  --binding MYSQL_USER=username \
  --binding MYSQL_PWD=password \
  --binding MYSQL_DATABASE=database
```

---

## API テンプレート

API キーとシークレットを保存します。

### フィールド

| フィールド | 機密 | 説明 |
|-----------|------|------|
| `api_key` | はい | API キーまたはアクセストークン |
| `api_secret` | はい | API シークレットまたは秘密鍵 |

### 環境バインディング

| 環境変数 | マップ先 |
|---------|---------|
| `API_KEY` | `api_key` |
| `API_SECRET` | `api_secret` |

### CLI 例

```bash
# API 認証情報を作成
secretctl set stripe/live \
  --field api_key=sk_live_xxx \
  --field api_secret=whsec_xxx \
  --sensitive api_key \
  --sensitive api_secret \
  --binding STRIPE_API_KEY=api_key \
  --binding STRIPE_WEBHOOK_SECRET=api_secret

# API 認証情報で実行
secretctl run -k stripe/live -- ./process-webhooks.sh
```

### サービス固有のバインディング

異なるサービスは異なる環境変数名を使用します。例:

**OpenAI:**
```bash
secretctl set openai/prod \
  --field api_key=sk-proj-xxx \
  --sensitive api_key \
  --binding OPENAI_API_KEY=api_key
```

**AWS:**
```bash
secretctl set aws/prod \
  --field api_key=AKIAIOSFODNN7EXAMPLE \
  --field api_secret=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --sensitive api_key \
  --sensitive api_secret \
  --binding AWS_ACCESS_KEY_ID=api_key \
  --binding AWS_SECRET_ACCESS_KEY=api_secret
```

---

## SSH テンプレート

SSH 秘密鍵とパスフレーズを保存します。

### フィールド

| フィールド | 機密 | 入力タイプ | 説明 |
|-----------|------|-----------|------|
| `private_key` | はい | `textarea` | SSH 秘密鍵の内容（複数行） |
| `passphrase` | はい | `text` | 鍵のパスフレーズ（オプション） |

:::tip 複数行入力
`private_key` フィールドはデスクトップアプリでテキストエリア入力を使用するため、PEM 形式の SSH 鍵を簡単に貼り付けられます。CLI もこのフィールドで複数行入力をサポートしています。
:::

### 環境バインディング

デフォルトバインディングなし。SSH 鍵は通常、環境変数ではなくファイルに書き込まれます。

### CLI 例

```bash
# SSH 鍵を保存
secretctl set ssh/server1 \
  --field private_key="$(cat ~/.ssh/id_ed25519)" \
  --field passphrase=mypassphrase \
  --sensitive private_key \
  --sensitive passphrase

# 特定のフィールドを取得
secretctl get ssh/server1 --field private_key > /tmp/key
chmod 600 /tmp/key
ssh -i /tmp/key user@server1
rm /tmp/key
```

---

## フィールド属性

各フィールドには以下の属性があります:

| 属性 | 型 | 説明 |
|------|-----|------|
| `value` | string | フィールドのシークレット値 |
| `sensitive` | boolean | 値をマスクすべきかどうか |
| `inputType` | string | UI 入力タイプ: `"text"`（デフォルト）または `"textarea"` |
| `kind` | string | Phase 3 スキーマ検証用に予約（オプション） |
| `aliases` | string[] | フィールドの別名（オプション） |
| `hint` | string | UI に表示されるヘルパーテキスト（オプション） |

### 入力タイプ

`inputType` 属性はデスクトップアプリでフィールドがどのようにレンダリングされるかを制御します:

| 入力タイプ | ユースケース | フィールド例 |
|-----------|-------------|-------------|
| `text` | 単一行の値 | `username`, `password`, `api_key` |
| `textarea` | 複数行の値 | `private_key`, 証明書, 設定ファイル |

テンプレート使用時、`inputType` はフィールドの一般的な内容に基づいて自動設定されます。

---

## フィールド命名規則

### ルール

フィールド名は以下のルールに従う必要があります:

- **文字**: 小文字、数字、アンダースコアのみ
- **形式**: `snake_case`（例: `api_key`, `private_key`）
- **長さ**: 最大64文字
- **予約**: アンダースコアで始めることはできない

### 有効な例

```
username
password
api_key
api_secret
private_key
database_url
connection_string
```

### 無効な例

```
apiKey          # camelCase は不可
API_KEY         # 大文字は不可
api-key         # ハイフンは不可
api key         # スペースは不可
_private        # アンダースコアで始めることは不可
```

---

## カスタムテンプレート

secretctl は4つの組み込みテンプレートを提供していますが、CLI フラグを使用して任意のフィールド構造を作成できます:

```bash
# カスタム OAuth 認証情報
secretctl set oauth/google \
  --field client_id=xxx.apps.googleusercontent.com \
  --field client_secret=GOCSPX-xxx \
  --field refresh_token=1//xxx \
  --sensitive client_secret \
  --sensitive refresh_token \
  --binding GOOGLE_CLIENT_ID=client_id \
  --binding GOOGLE_CLIENT_SECRET=client_secret \
  --binding GOOGLE_REFRESH_TOKEN=refresh_token
```

---

## MCP 連携

### フィールドの発見

AI エージェントは `secret_list_fields` を使用してフィールド構造を発見できます:

```json
// リクエスト
{"key": "db/prod"}

// レスポンス
{
  "key": "db/prod",
  "fields": ["host", "port", "username", "password", "database"],
  "sensitive_fields": ["password"],
  "bindings": {
    "PGHOST": "host",
    "PGPORT": "port",
    "PGUSER": "username",
    "PGPASSWORD": "password",
    "PGDATABASE": "database"
  }
}
```

### 非機密フィールドへのアクセス

AI エージェントは `secret_get_field` 経由で非機密フィールドの値を読み取れます:

```json
// リクエスト
{"key": "db/prod", "field": "host"}

// レスポンス
{"key": "db/prod", "field": "host", "value": "db.example.com"}

// 機密フィールドのリクエスト（ブロック）
{"key": "db/prod", "field": "password"}

// レスポンス
{"error": "フィールド 'password' は機密としてマークされています"}
```

### バインディングで実行

`secret_run_with_bindings` を使用して環境変数付きでコマンドを実行:

```json
// リクエスト
{
  "key": "db/prod",
  "command": ["psql", "-c", "SELECT 1"]
}

// レスポンス
{"exit_code": 0, "stdout": "...", "stderr": ""}
```

---

## ベストプラクティス

### 1. 一貫した命名を使用

シークレット全体で命名規則を採用:

```
service/environment/purpose
├── db/prod/main
├── db/staging/main
├── aws/prod/deploy
└── stripe/prod/webhook
```

### 2. 機密フィールドをマーク

パスワード類似のフィールドは常に機密としてマーク:

```bash
--sensitive password
--sensitive api_key
--sensitive private_key
--sensitive secret
--sensitive token
```

### 3. 自動化にバインディングを使用

シークレット作成時にバインディングを定義して `secret_run` をシームレスに連携:

```bash
secretctl run -k db/prod -- psql  # 事前定義のバインディングで動作
```

### 4. カスタムフィールドを文書化

カスタムフィールド構造にはメモを追加:

```bash
secretctl set custom/service \
  --field token=xxx \
  --field endpoint=https://api.example.com \
  --sensitive token \
  --notes "カスタムサービス: token=認証, endpoint=API URL"
```

---

## 関連項目

- [CLI コマンド](/docs/reference/cli-commands) - 完全な CLI リファレンス
- [MCP ツール](/docs/reference/mcp-tools) - MCP ツールドキュメント
- [デスクトップアプリガイド](/docs/guides/desktop) - デスクトップアプリ概要
