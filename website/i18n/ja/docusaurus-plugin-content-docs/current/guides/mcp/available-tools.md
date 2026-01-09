---
title: 利用可能なツール
description: secretctl が提供する MCP ツール。
sidebar_position: 4
---

# 利用可能なツール

secretctl は AI エージェントがシークレットを安全に扱うための7つの MCP ツールを提供します。

## 概要

| ツール | 説明 |
|--------|------|
| `secret_list` | メタデータ付きでシークレットキーを一覧（値なし） |
| `secret_exists` | メタデータ付きでシークレットの存在を確認 |
| `secret_get_masked` | マスクされたシークレット値を取得（例: `****WXYZ`） |
| `secret_run` | シークレットを環境変数としてコマンドを実行 |
| `secret_list_fields` | マルチフィールドシークレットのフィールド名を一覧（値なし） |
| `secret_get_field` | 非機密フィールドの値のみを取得 |
| `secret_run_with_bindings` | 定義済み環境バインディングで実行 |

## secret_list

すべてのシークレットキーをメタデータ付きで一覧。シークレット値は**返しません**。

**入力スキーマ:**

```json
{
  "tag": "オプションのタグフィルター",
  "expiring_within": "オプションの有効期限フィルター（例: '7d', '30d'）"
}
```

**レスポンス例:**

```json
{
  "secrets": [
    {
      "key": "aws/access_key",
      "tags": ["aws", "prod"],
      "has_url": true,
      "has_notes": false,
      "expires_at": "2025-12-31T00:00:00Z",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-06-15T10:30:00Z"
    }
  ]
}
```

## secret_exists

シークレットキーが存在するかを確認し、メタデータを返します。

**入力スキーマ:**

```json
{
  "key": "aws/access_key"
}
```

**レスポンス例:**

```json
{
  "exists": true,
  "key": "aws/access_key",
  "tags": ["aws", "prod"],
  "has_url": true,
  "has_notes": false,
  "expires_at": "2025-12-31T00:00:00Z",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-06-15T10:30:00Z"
}
```

## secret_get_masked

シークレット値のマスクされたバージョンを取得。検証に便利です。

**入力スキーマ:**

```json
{
  "key": "aws/access_key"
}
```

**レスポンス例:**

```json
{
  "key": "aws/access_key",
  "masked_value": "****WXYZ",
  "value_length": 20
}
```

## secret_run

シークレットを環境変数として注入してコマンドを実行。

**入力スキーマ:**

```json
{
  "command": "aws",
  "args": ["s3", "ls"],
  "keys": ["aws/access_key", "aws/secret_key"],
  "timeout": "30s",
  "env_prefix": "AWS_",
  "env": "prod"
}
```

**パラメータ:**

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `command` | string | はい | 実行するコマンド |
| `args` | string[] | いいえ | コマンド引数 |
| `keys` | string[] | はい | シークレットキーパターン（glob 対応） |
| `timeout` | string | いいえ | 実行タイムアウト（デフォルト: 5m、最大: 1h） |
| `env_prefix` | string | いいえ | 環境変数名のプレフィックス |
| `env` | string | いいえ | 環境エイリアス（例: "dev", "staging", "prod"） |

**機能:**

- シークレットは環境変数として注入
- 漏洩したシークレットは `[REDACTED:key]` で出力が自動的にサニタイズ
- ポリシー承認が必要
- 最大5つの同時実行

**レスポンス例:**

```json
{
  "exit_code": 0,
  "stdout": "2024-01-15 mybucket\n2024-02-20 myotherbucket",
  "stderr": "",
  "sanitized": true
}
```

### 環境変数の命名

シークレットキーは以下のように環境変数名に変換されます:

1. スラッシュ（`/`）はアンダースコア（`_`）に置換
2. ハイフン（`-`）はアンダースコア（`_`）に置換
3. 結果は大文字に変換
4. `env_prefix` が指定されている場合、先頭に追加

**例:**

| シークレットキー | env_prefix | 環境変数 |
|-----------------|------------|---------|
| `aws/access_key` | (なし) | `AWS_ACCESS_KEY` |
| `aws/access_key` | `MY_` | `MY_AWS_ACCESS_KEY` |
| `db-password` | (なし) | `DB_PASSWORD` |
| `api/prod/key` | `APP_` | `APP_API_PROD_KEY` |

## secret_list_fields

マルチフィールドシークレットのすべてのフィールド名とメタデータを一覧。フィールド値は**返しません**。

**入力スキーマ:**

```json
{
  "key": "database/production"
}
```

**レスポンス例:**

```json
{
  "key": "database/production",
  "fields": [
    {
      "name": "host",
      "sensitive": false,
      "hint": "データベースホスト名",
      "kind": "string"
    },
    {
      "name": "password",
      "sensitive": true,
      "hint": "データベースパスワード"
    }
  ]
}
```

## secret_get_field

マルチフィールドシークレットから特定のフィールド値を取得。**非機密フィールドのみ取得可能**（AI安全設計ポリシー）。機密フィールドは拒否されます。

**入力スキーマ:**

```json
{
  "key": "database/production",
  "field": "host"
}
```

**レスポンス例:**

```json
{
  "key": "database/production",
  "field": "host",
  "value": "db.example.com",
  "sensitive": false
}
```

**機密フィールドのエラー:**

```json
{
  "error": "フィールド 'password' は機密としてマークされており、MCP 経由で取得できません（AI安全設計ポリシー）"
}
```

## secret_run_with_bindings

シークレットの定義済みバインディングに基づいて環境変数を注入してコマンドを実行。各バインディングは環境変数名をフィールドにマップします。

**入力スキーマ:**

```json
{
  "key": "database/production",
  "command": "psql",
  "args": ["-c", "SELECT 1"],
  "timeout": "30s"
}
```

**パラメータ:**

| パラメータ | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | バインディング付きシークレットキー |
| `command` | string | はい | 実行するコマンド |
| `args` | string[] | いいえ | コマンド引数 |
| `timeout` | string | いいえ | 実行タイムアウト（デフォルト: 5m） |

**バインディングの仕組み:**

バインディング付きでシークレットを作成:

```bash
secretctl set database/production \
  --field host=db.example.com \
  --field port=5432 \
  --field password \
  --binding PGHOST=host \
  --binding PGPORT=port \
  --binding PGPASSWORD=password
```

`secret_run_with_bindings` を呼び出すと以下が注入されます:
- `PGHOST=db.example.com`
- `PGPORT=5432`
- `PGPASSWORD=<password value>`

**レスポンス例:**

```json
{
  "exit_code": 0,
  "stdout": " ?column? \n----------\n        1\n(1 row)\n",
  "stderr": "",
  "sanitized": true
}
```

## 技術詳細

### プロトコル

- stdio 上の JSON-RPC 2.0
- MCP 仕様 2024-11-05 対応

### 並行性

- 最大5つの `secret_run` 同時実行
- 追加リクエストはキューイング

### パフォーマンス目標

| 操作 | p50 | p99 |
|------|-----|-----|
| secret_list | < 50ms | < 200ms |
| secret_exists | < 10ms | < 50ms |
| secret_get_masked | < 10ms | < 50ms |
| secret_run（起動） | < 100ms | < 500ms |
| MCP 初期化 | < 500ms | < 2s |
