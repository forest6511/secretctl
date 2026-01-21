---
title: MCP ツール
description: secretctl MCP サーバーツールの完全 API リファレンス。
sidebar_position: 2
---

# MCP ツールリファレンス

secretctl MCP サーバーが提供するすべての MCP ツールの完全 API リファレンス。

## 概要

secretctl MCP サーバーは、**AI エージェントが平文のシークレットを受け取ることがない**セキュリティファーストの設計を実装しています。これは 1Password の「Access Without Exposure」哲学に沿った「AI安全設計」アーキテクチャです。

**利用可能なツール:**

| ツール | 説明 |
|--------|------|
| `secret_list` | メタデータ付きでシークレットキーを一覧（値なし） |
| `secret_exists` | メタデータ付きでシークレットの存在を確認 |
| `secret_get_masked` | マスクされたシークレット値を取得（例: `****WXYZ`） |
| `secret_run` | シークレットを環境変数としてコマンドを実行 |
| `secret_list_fields` | マルチフィールドシークレットのフィールド名を一覧（値なし） |
| `secret_get_field` | 非機密フィールドの値のみを取得 |
| `secret_run_with_bindings` | 定義済み環境バインディングで実行 |
| `security_score` | Vault のセキュリティ健全性スコアと推奨事項を取得 |

---

## secret_list

すべてのシークレットキーをメタデータ付きで一覧。キー名、タグ、有効期限、メモ/URL の存在フラグを返します。シークレット値は返しません。

### 入力スキーマ

```json
{
  "tag": "string (オプション)",
  "expiring_within": "string (オプション)"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `tag` | string | いいえ | タグでフィルター |
| `expiring_within` | string | いいえ | 有効期限でフィルター（例: `7d`, `30d`） |

### 出力スキーマ

```json
{
  "secrets": [
    {
      "key": "string",
      "field_count": "number",
      "tags": ["string"],
      "expires_at": "string (RFC 3339, オプション)",
      "has_notes": "boolean",
      "has_url": "boolean",
      "created_at": "string (RFC 3339)",
      "updated_at": "string (RFC 3339)"
    }
  ]
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `key` | string | シークレットキー名 |
| `field_count` | number | フィールド数（レガシー単一値シークレットは1） |
| `tags` | array | シークレットに関連付けられたタグ |
| `expires_at` | string | RFC 3339 形式の有効期限（オプション） |
| `has_notes` | boolean | シークレットにメモがあるかどうか |
| `has_url` | boolean | シークレットに URL があるかどうか |
| `created_at` | string | RFC 3339 形式の作成タイムスタンプ |
| `updated_at` | string | RFC 3339 形式の最終更新タイムスタンプ |

### 例

**すべてのシークレットを一覧:**

```json
// 入力
{}

// 出力
{
  "secrets": [
    {
      "key": "AWS_ACCESS_KEY",
      "field_count": 1,
      "tags": ["aws", "prod"],
      "has_notes": false,
      "has_url": true,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    },
    {
      "key": "db/prod",
      "field_count": 5,
      "tags": ["db", "prod"],
      "expires_at": "2025-06-15T00:00:00Z",
      "has_notes": true,
      "has_url": false,
      "created_at": "2025-01-10T08:00:00Z",
      "updated_at": "2025-01-10T08:00:00Z"
    }
  ]
}
```

**タグでフィルター:**

```json
// 入力
{
  "tag": "prod"
}

// 出力
{
  "secrets": [/* "prod" タグを持つシークレット */]
}
```

**有効期限でフィルター:**

```json
// 入力
{
  "expiring_within": "30d"
}

// 出力
{
  "secrets": [/* 30日以内に期限切れになるシークレット */]
}
```

---

## secret_exists

シークレットキーが存在するかを確認し、メタデータを返します。シークレット値は返しません。

### 入力スキーマ

```json
{
  "key": "string"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | 確認するシークレットキー |

### 出力スキーマ

```json
{
  "exists": "boolean",
  "key": "string",
  "tags": ["string"],
  "expires_at": "string (RFC 3339, オプション)",
  "has_notes": "boolean",
  "has_url": "boolean",
  "created_at": "string (RFC 3339, オプション)",
  "updated_at": "string (RFC 3339, オプション)"
}
```

### 例

**存在するシークレットを確認:**

```json
// 入力
{
  "key": "API_KEY"
}

// 出力
{
  "exists": true,
  "key": "API_KEY",
  "tags": ["api", "prod"],
  "has_notes": true,
  "has_url": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**存在しないシークレットを確認:**

```json
// 入力
{
  "key": "NONEXISTENT_KEY"
}

// 出力
{
  "exists": false,
  "key": "NONEXISTENT_KEY",
  "tags": null,
  "has_notes": false,
  "has_url": false
}
```

---

## secret_get_masked

シークレット値のマスクされたバージョンを取得（例: `****WXYZ`）。実際の値を公開せずにシークレットの形式を確認するのに便利です。

### 入力スキーマ

```json
{
  "key": "string"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | 取得するシークレットキー |

### 出力スキーマ

```json
{
  "key": "string",
  "masked_value": "string",
  "value_length": "integer",
  "field_count": "integer",
  "fields": {
    "field_name": {
      "value": "string",
      "sensitive": "boolean",
      "value_length": "integer"
    }
  }
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `key` | string | シークレットキー |
| `masked_value` | string | マスクされたプライマリ値（単一値シークレットのみ；マルチフィールドでは空） |
| `value_length` | integer | プライマリ値の長さ（単一値シークレットのみ；マルチフィールドでは0） |
| `field_count` | integer | このシークレットのフィールド数 |
| `fields` | object | フィールド名からマスク済みフィールド情報へのマップ（マルチフィールドシークレットのみ） |

> **注意**: マルチフィールドシークレットの場合、`masked_value` と `value_length` は空/0になります。
> 個々のフィールド値にアクセスするには `fields` マップを使用してください。

### マスキング動作

マスキングアルゴリズム:
- 9文字以上のシークレットは末尾4文字を表示
- 5-8文字のシークレットは末尾2文字を表示
- 1-4文字のシークレットはアスタリスクのみ表示
- 非機密フィールドは完全に表示（マスクなし）

| シークレット長 | マスク出力 |
|---------------|-----------|
| 1-4文字 | `****`（すべてアスタリスク） |
| 5-8文字 | `******YZ`（末尾2文字を表示） |
| 9文字以上 | `*****WXYZ`（末尾4文字を表示） |

**マルチフィールドシークレット:**
- `fields` マップにはすべてのフィールドのマスク済み/完全な値が含まれます
- 機密フィールドは上記のアルゴリズムに従ってマスクされます
- 非機密フィールドは完全な値を表示します

### 例

**マスク値を取得（単一フィールドシークレット）:**

```json
// 入力
{
  "key": "API_KEY"
}

// 出力（API_KEY = "sk-abc123xyz789" の場合）
{
  "key": "API_KEY",
  "masked_value": "**********z789",
  "value_length": 14,
  "field_count": 1
}
```

**短いシークレット:**

```json
// 入力
{
  "key": "PIN"
}

// 出力（PIN = "1234" の場合）
{
  "key": "PIN",
  "masked_value": "****",
  "value_length": 4,
  "field_count": 1
}
```

**マルチフィールドシークレット（例: データベース認証情報）:**

```json
// 入力
{
  "key": "db/postgres"
}

// 出力（username, password, host を持つマルチフィールドシークレット）
{
  "key": "db/postgres",
  "masked_value": "",
  "value_length": 0,
  "field_count": 3,
  "fields": {
    "username": {
      "value": "dbadmin",
      "sensitive": false,
      "value_length": 7
    },
    "password": {
      "value": "***********5678",
      "sensitive": true,
      "value_length": 16
    },
    "host": {
      "value": "db.example.com",
      "sensitive": false,
      "value_length": 14
    }
  }
}
```

---

## secret_run

指定したシークレットを環境変数として注入してコマンドを実行。出力はシークレット漏洩を防ぐために自動的にサニタイズされます。ポリシー承認が必要です。

### 入力スキーマ

```json
{
  "keys": ["string"],
  "command": "string",
  "args": ["string"],
  "timeout": "string (オプション)",
  "env_prefix": "string (オプション)",
  "env": "string (オプション)"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `keys` | string[] | はい | 注入するシークレットキー（glob パターン対応） |
| `command` | string | はい | 実行するコマンド |
| `args` | string[] | いいえ | コマンド引数 |
| `timeout` | string | いいえ | 実行タイムアウト（例: `30s`, `5m`）。デフォルト: `5m` |
| `env_prefix` | string | いいえ | 環境変数名のプレフィックス |
| `env` | string | いいえ | 環境エイリアス（例: `dev`, `staging`, `prod`） |

### 出力スキーマ

```json
{
  "exit_code": "integer",
  "stdout": "string",
  "stderr": "string",
  "duration_ms": "integer",
  "sanitized": "boolean"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `exit_code` | integer | コマンド終了コード（0 = 成功） |
| `stdout` | string | 標準出力（サニタイズ済み） |
| `stderr` | string | 標準エラー（サニタイズ済み） |
| `duration_ms` | integer | 実行時間（ミリ秒） |
| `sanitized` | boolean | 出力がサニタイズされたかどうか |

### キーパターン構文

`keys` フィールドは glob パターンをサポート:

| パターン | マッチ |
|---------|-------|
| `API_KEY` | `API_KEY` に完全一致 |
| `aws/*` | `aws/` 配下のすべてのキー（単一レベル） |
| `db/*` | `db/` 配下のすべてのキー（単一レベル） |

### 環境変数の命名

シークレットキーは環境変数名に変換されます:

- `/` は `_` に置換
- `-` は `_` に置換
- 名前は大文字に変換

| シークレットキー | 環境変数 |
|-----------------|---------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |

### 出力サニタイズ

すべてのコマンド出力はシークレット値についてスキャンされます。マッチした箇所は `[REDACTED:key]` に置換されます:

```
オリジナル: "Connected to database with password secret123"
サニタイズ後: "Connected to database with password [REDACTED:DB_PASSWORD]"
```

### 例

**単一シークレットで実行:**

```json
// 入力
{
  "keys": ["API_KEY"],
  "command": "curl",
  "args": ["-H", "Authorization: Bearer $API_KEY", "https://api.example.com"]
}

// 出力
{
  "exit_code": 0,
  "stdout": "{\"status\": \"ok\"}",
  "stderr": "",
  "duration_ms": 245,
  "sanitized": false
}
```

**ワイルドカードパターンで実行:**

```json
// 入力
{
  "keys": ["aws/*"],
  "command": "aws",
  "args": ["s3", "ls"]
}

// 出力
{
  "exit_code": 0,
  "stdout": "2025-01-15 10:30:00 my-bucket\n",
  "stderr": "",
  "duration_ms": 1250,
  "sanitized": false
}
```

**環境エイリアスで実行:**

```json
// 入力
{
  "keys": ["db/*"],
  "command": "./deploy.sh",
  "env": "prod"
}

// 出力
{
  "exit_code": 0,
  "stdout": "Deployment complete",
  "stderr": "",
  "duration_ms": 5000,
  "sanitized": false
}
```

**プレフィックス付きで実行:**

```json
// 入力
{
  "keys": ["API_KEY"],
  "command": "./app",
  "env_prefix": "MYAPP_"
}

// 環境変数:
// MYAPP_API_KEY=<value>

// 出力
{
  "exit_code": 0,
  "stdout": "Application started",
  "stderr": "",
  "duration_ms": 100,
  "sanitized": false
}
```

---

## secret_list_fields

マルチフィールドシークレットのすべてのフィールド名とメタデータを一覧。フィールド名、機密フラグ、ヒント、エイリアスを返します。フィールド値は返しません。

### 入力スキーマ

```json
{
  "key": "string"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | フィールドを一覧するシークレットキー |

### 出力スキーマ

```json
{
  "key": "string",
  "fields": [
    {
      "name": "string",
      "sensitive": "boolean",
      "hint": "string (オプション)",
      "kind": "string (オプション)",
      "aliases": ["string"]
    }
  ]
}
```

### 例

**データベースシークレットのフィールドを一覧:**

```json
// 入力
{
  "key": "database/production"
}

// 出力
{
  "key": "database/production",
  "fields": [
    {
      "name": "host",
      "sensitive": false,
      "hint": "データベースホスト名"
    },
    {
      "name": "port",
      "sensitive": false,
      "hint": "データベースポート"
    },
    {
      "name": "username",
      "sensitive": false,
      "hint": "データベースユーザー名"
    },
    {
      "name": "password",
      "sensitive": true,
      "hint": "データベースパスワード"
    }
  ]
}
```

---

## secret_get_field

マルチフィールドシークレットから特定のフィールド値を取得。非機密フィールドのみ取得可能（AI安全設計ポリシー）。機密フィールドは拒否されます。

### 入力スキーマ

```json
{
  "key": "string",
  "field": "string"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | シークレットキー |
| `field` | string | はい | 取得するフィールド名 |

### 出力スキーマ

```json
{
  "key": "string",
  "field": "string",
  "value": "string",
  "sensitive": "boolean"
}
```

### 例

**非機密フィールドを取得:**

```json
// 入力
{
  "key": "database/production",
  "field": "host"
}

// 出力
{
  "key": "database/production",
  "field": "host",
  "value": "db.example.com",
  "sensitive": false
}
```

**機密フィールドの取得を試行（拒否）:**

```json
// 入力
{
  "key": "database/production",
  "field": "password"
}

// エラーレスポンス
{
  "error": "フィールド 'password' は機密としてマークされており、MCP 経由で取得できません（AI安全設計ポリシー）"
}
```

---

## secret_run_with_bindings

シークレットの定義済みバインディングに基づいて環境変数を注入してコマンドを実行。各バインディングは環境変数名をフィールドにマップします。ポリシー承認が必要です。

### 入力スキーマ

```json
{
  "key": "string",
  "command": "string",
  "args": ["string"],
  "timeout": "string (オプション)"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `key` | string | はい | バインディングが定義されたシークレットキー |
| `command` | string | はい | 実行するコマンド |
| `args` | string[] | いいえ | コマンド引数 |
| `timeout` | string | いいえ | 実行タイムアウト（例: `30s`, `5m`）。デフォルト: `5m` |

### 出力スキーマ

```json
{
  "exit_code": "integer",
  "stdout": "string",
  "stderr": "string",
  "sanitized": "boolean"
}
```

### バインディングの仕組み

バインディング付きでシークレットを作成すると、各バインディングが環境変数名をフィールドにマップします:

```bash
secretctl set database/production \
  --field host=db.example.com \
  --field port=5432 \
  --field password \
  --binding PGHOST=host \
  --binding PGPORT=port \
  --binding PGPASSWORD=password
```

このシークレットで `secret_run_with_bindings` を呼び出すと、以下の環境変数が設定されます:
- `PGHOST=db.example.com`
- `PGPORT=5432`
- `PGPASSWORD=<password value>`

### 例

**PostgreSQL コマンドを実行:**

```json
// 入力
{
  "key": "database/production",
  "command": "psql",
  "args": ["-c", "SELECT 1"]
}

// 出力
{
  "exit_code": 0,
  "stdout": " ?column? \n----------\n        1\n(1 row)\n",
  "stderr": "",
  "sanitized": true
}
```

---

## security_score

パスワード強度、重複検出、有効期限状態を含む Vault のセキュリティ健全性スコアを取得します。0-100のスコアと問題の詳細、推奨事項を返します。

### 入力スキーマ

```json
{
  "include_keys": "boolean (オプション)"
}
```

| フィールド | 型 | 必須 | 説明 |
|-----------|-----|------|------|
| `include_keys` | boolean | いいえ | 問題詳細にシークレットキーを含めるか（デフォルト: false） |

### 出力スキーマ

```json
{
  "overall_score": "integer (0-100)",
  "components": {
    "strength": "integer (0-25)",
    "uniqueness": "integer (0-25)",
    "expiration": "integer (0-25)",
    "coverage": "integer (0-25)"
  },
  "issues_count": {
    "duplicates": "integer",
    "weak": "integer",
    "expiring": "integer",
    "expired": "integer"
  },
  "top_issues": [
    {
      "type": "string",
      "severity": "string",
      "count": "integer",
      "description": "string",
      "secret_keys": ["string"]
    }
  ],
  "suggestions": ["string"],
  "limited": "boolean"
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `overall_score` | integer | 総合セキュリティスコア（0-100） |
| `components` | object | カテゴリ別のスコア内訳（各0-25） |
| `components.strength` | integer | パスワード強度スコア |
| `components.uniqueness` | integer | パスワードユニーク性スコア |
| `components.expiration` | integer | 有効期限コンプライアンススコア |
| `components.coverage` | integer | フィールドカバレッジスコア（Phase 3、現在は常に25） |
| `issues_count` | object | 問題タイプ別のカウント |
| `top_issues` | array | 問題と詳細（Free版: weak/duplicateは各3件まで） |
| `suggestions` | array | 実行可能な推奨事項 |
| `limited` | boolean | Free版の制限により結果が制限された場合true |

### 問題タイプ

| タイプ | 重大度 | 説明 |
|--------|--------|------|
| `weak` | warning | パスワード強度が不十分 |
| `duplicate` | warning | 複数のシークレットが同じパスワードを共有 |
| `expiring` | warning | シークレットが警告期間内に期限切れ |
| `expired` | critical | シークレットが既に期限切れ |

### 例

**セキュリティスコアを取得:**

```json
// 入力
{}

// 出力
{
  "overall_score": 85,
  "components": {
    "strength": 20,
    "uniqueness": 25,
    "expiration": 15,
    "coverage": 25
  },
  "issues_count": {
    "duplicates": 0,
    "weak": 2,
    "expiring": 1,
    "expired": 0
  },
  "top_issues": [
    {
      "type": "weak",
      "severity": "warning",
      "description": "Password has insufficient strength"
    },
    {
      "type": "expiring",
      "severity": "warning",
      "description": "Secret expires in 5 days"
    }
  ],
  "suggestions": [
    "Update weak passwords with stronger alternatives (14+ characters)",
    "Plan to renew expiring credentials before they expire"
  ],
  "limited": false
}
```

**シークレットキー付きでセキュリティスコアを取得:**

```json
// 入力
{
  "include_keys": true
}

// 出力
{
  "overall_score": 85,
  "components": {
    "strength": 20,
    "uniqueness": 25,
    "expiration": 15,
    "coverage": 25
  },
  "issues_count": {
    "duplicates": 0,
    "weak": 2,
    "expiring": 1,
    "expired": 0
  },
  "top_issues": [
    {
      "type": "weak",
      "severity": "warning",
      "description": "Password has insufficient strength",
      "secret_keys": ["github-token", "api/legacy"]
    },
    {
      "type": "expiring",
      "severity": "warning",
      "description": "Secret expires in 5 days",
      "secret_keys": ["aws/temp-token"]
    }
  ],
  "suggestions": [
    "Update weak passwords with stronger alternatives (14+ characters)",
    "Plan to renew expiring credentials before they expire"
  ],
  "limited": false
}
```

### Free版 vs Team版

| 機能 | Free | Team |
|------|------|------|
| セキュリティスコア | ✅ | ✅ |
| 問題カウント | ✅ | ✅ |
| 弱いパスワードの問題 | 最大3件 | 無制限 |
| 重複パスワードの問題 | 最大3件 | 無制限 |
| 期限切れ/期限間近の問題 | ✅ | ✅ |
| 問題内のシークレットキー | ✅ | ✅ |
| チーム全体ダッシュボード | ❌ | ✅ |

`limited: true` の場合、Free版の制限により一部の弱いパスワードまたは重複パスワードの問題が省略されています。完全な可視性のためにTeam版にアップグレードしてください。

---

## ポリシー設定

`secret_run` と `secret_run_with_bindings` ツールにはポリシー承認が必要です。`~/.secretctl/mcp-policy.yaml` を作成してください:

```yaml
version: 1
default_action: deny

# 常にブロックされるコマンド（セキュリティ）
# - env, printenv, set, export, cat /proc/*/environ

# ユーザー定義の拒否コマンド
denied_commands: []

# 許可されたコマンド（default_action が deny の場合に必要）
allowed_commands:
  - aws
  - gcloud
  - kubectl
  - curl
  - ./deploy.sh

# キー変換用の環境エイリアス
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### ポリシー評価順序

1. **デフォルト拒否コマンド**（常にブロック）: `env`, `printenv`, `set`, `export`
2. **ユーザー定義の `denied_commands`**: 明示的にブロックされたコマンド
3. **ユーザー定義の `allowed_commands`**: 明示的に許可されたコマンド
4. **`default_action`**: フォールバックアクション（`allow` または `deny`）

### セキュリティ要件

ポリシーファイルは以下の要件を満たす必要があります:
- ファイルパーミッション: `0600`（オーナーのみ読み書き）
- シンボリックリンク不可（直接ファイルのみ）
- 現在のユーザーが所有

---

## セキュリティ設計

### AI安全設計アーキテクチャ

secretctl MCP サーバーは「AI安全設計」セキュリティモデルに従っています:

| ツール | 平文アクセス | 目的 |
|--------|-------------|------|
| `secret_list` | なし | キーとメタデータのみ一覧 |
| `secret_exists` | なし | 存在とメタデータを確認 |
| `secret_get_masked` | なし | 公開せずに形式を確認 |
| `secret_run` | なし* | 環境変数経由で注入 |
| `secret_list_fields` | なし | フィールド名とメタデータのみ一覧 |
| `secret_get_field` | なし** | 非機密フィールドのみ |
| `secret_run_with_bindings` | なし* | 定義済みバインディング経由で注入 |
| `security_score` | なし | セキュリティ指標と推奨事項を取得 |

\* シークレットは子プロセスに環境変数として注入されます。AI エージェントは平文の値を見ることはありません。

\** `sensitive: false` とマークされたフィールドのみ取得可能。機密フィールドはエラーで拒否されます。

### なぜ `secret_get` がないのか？

平文の値を返す `secret_get` ツールは意図的に**実装されていません**。この設計上の選択は以下に準拠しています:

- **1Password**: MCP 経由で生の認証情報を公開することを明示的に拒否
- **HashiCorp Vault**: 「生のシークレットは決して公開しない」ポリシー
- **業界のベストプラクティス**: シークレットの露出面を最小化

### 出力サニタイズ

`secret_run` ツールは偶発的なシークレット漏洩を防ぐためにコマンド出力を自動的にサニタイズします:

- シークレット値の完全一致を検出
- `[REDACTED:key]` プレースホルダーに置換
- stdout と stderr の両方に適用

**制限事項:**
- Base64 や hex エンコードされたシークレットは検出されません
- 部分一致は検出されません
- 難読化または変換された値は検出されません

---

## エラー処理

### 一般的なエラー

| エラー | 説明 |
|--------|------|
| `secret not found` | 要求されたシークレットキーが存在しない |
| `policy not found` | MCP ポリシーファイルが存在しない |
| `command not allowed` | コマンドがポリシーでブロックされた |
| `timeout exceeded` | コマンド実行がタイムアウトを超過 |

### エラーレスポンス形式

```json
{
  "error": {
    "code": -32000,
    "message": "secret not found: API_KEY"
  }
}
```

---

## 連携例

Claude Code (`~/.claude.json`) で設定:

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

詳細なセットアップ手順は [MCP 連携ガイド](/docs/guides/mcp/) を参照してください。
