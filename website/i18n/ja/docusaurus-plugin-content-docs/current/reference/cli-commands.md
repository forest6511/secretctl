---
title: CLI コマンド
description: secretctl CLI コマンドの完全リファレンス。
sidebar_position: 1
---

# CLI コマンドリファレンス

secretctl のすべての CLI コマンドの完全リファレンス。

## グローバルオプション

```bash
secretctl [command] --help    # 任意のコマンドのヘルプを表示
```

---

## init

新しいシークレット Vault を初期化。

```bash
secretctl init
```

`~/.secretctl/vault.db` に新しい暗号化 Vault を作成します。マスターパスワード（最低8文字）の設定を求められます。

**例:**

```bash
$ secretctl init
Enter master password: ********
Confirm master password: ********
Vault initialized successfully.
```

---

## set

標準入力からシークレット値を保存、またはマルチフィールドシークレットを作成。

```bash
secretctl set [key] [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--field name=value` | シークレットにフィールドを追加（繰り返し指定可） |
| `--binding ENV=field` | 環境変数バインディングを追加（繰り返し指定可） |
| `--sensitive name` | フィールドを機密としてマーク（繰り返し指定可） |
| `--notes string` | シークレットにメモを追加 |
| `--tags string` | カンマ区切りのタグ（例: `dev,api`） |
| `--url string` | シークレットに URL 参照を追加 |
| `--expires string` | 有効期限（例: `30d`, `1y`） |

**例:**

```bash
# 基本的な使用方法（stdinからの単一値）
echo "sk-your-api-key" | secretctl set OPENAI_API_KEY

# マルチフィールドシークレット
secretctl set db/prod \
  --field host=db.example.com \
  --field port=5432 \
  --field user=admin \
  --field password=secret123 \
  --sensitive password

# 環境変数バインディング付き
secretctl set db/prod \
  --field host=db.example.com \
  --field password=secret123 \
  --binding PGHOST=host \
  --binding PGPASSWORD=password \
  --sensitive password

# メタデータ付き
echo "mypassword" | secretctl set DB_PASSWORD \
  --notes="本番データベース" \
  --tags="prod,db" \
  --url="https://console.example.com"

# 有効期限付き
echo "temp-token" | secretctl set TEMP_TOKEN --expires="30d"
```

---

## get

シークレット値または特定のフィールドを取得。

```bash
secretctl get [key] [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--field name` | 特定のフィールド値を取得 |
| `--fields` | すべてのフィールド名を一覧（値なし） |
| `--show-metadata` | シークレットとともにメタデータを表示 |

**例:**

```bash
# シークレット値のみを取得（レガシー単一値）
secretctl get API_KEY

# マルチフィールドシークレットから特定フィールドを取得
secretctl get db/prod --field host

# すべてのフィールド名を一覧
secretctl get db/prod --fields

# メタデータ付きでシークレットを取得
secretctl get API_KEY --show-metadata
```

---

## delete

Vault からシークレットを削除。

```bash
secretctl delete [key]
```

**例:**

```bash
secretctl delete OLD_API_KEY
```

---

## list

Vault 内のすべてのシークレットキーを一覧。

```bash
secretctl list [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--tag string` | タグでフィルター |
| `--expiring string` | 指定期間内に期限切れになるシークレットを表示（例: `7d`） |

**例:**

```bash
# すべてのシークレットを一覧
secretctl list

# タグでフィルター
secretctl list --tag=prod

# 期限切れ間近のシークレットを表示
secretctl list --expiring=7d
```

---

## run

シークレットを環境変数として注入してコマンドを実行。

```bash
secretctl run [flags] -- command [args...]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `-k, --key stringArray` | 注入するシークレットキー（glob パターン対応） |
| `-t, --timeout duration` | コマンドタイムアウト（デフォルト: `5m`） |
| `--env string` | 環境エイリアス（例: `dev`, `staging`, `prod`） |
| `--env-prefix string` | 環境変数名のプレフィックス |
| `--no-sanitize` | 出力サニタイズを無効化 |
| `--obfuscate-keys` | エラーメッセージ内のシークレットキー名を難読化 |

**環境変数の命名:**

シークレットキーは環境変数名に変換されます:

- `/` は `_` に置換
- `-` は `_` に置換
- 名前は大文字に変換

| シークレットキー | 環境変数 |
|------------------|----------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |

**例:**

```bash
# 単一シークレット
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" https://api.example.com

# 複数シークレット
secretctl run -k DB_HOST -k DB_USER -k DB_PASS -- psql

# ワイルドカードパターン（単一レベルにマッチ）
secretctl run -k "aws/*" -- aws s3 ls

# タイムアウト付き
secretctl run -k API_KEY --timeout=30s -- ./long-script.sh

# 環境エイリアス付き
secretctl run --env=prod -k "db/*" -- ./deploy.sh

# プレフィックス付き
secretctl run -k API_KEY --env-prefix=APP_ -- ./app
```

**出力サニタイズ:**

デフォルトでは、コマンド出力がシークレット値についてスキャンされます。マッチした箇所は `[REDACTED:key]` に置換されます。

```bash
# DB_PASSWORD に "secret123" が含まれている場合
$ secretctl run -k DB_PASSWORD -- echo "Password is $DB_PASSWORD"
Password is [REDACTED:DB_PASSWORD]
```

---

## export

シークレットを `.env` または JSON 形式でエクスポート。

```bash
secretctl export [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `-k, --key strings` | エクスポートするキー（glob パターン対応） |
| `-f, --format string` | 出力形式: `env`, `json`（デフォルト: `env`） |
| `-o, --output string` | 出力ファイルパス（デフォルト: 標準出力） |
| `--with-metadata` | JSON 出力にメタデータを含める |
| `--force` | 確認なしで既存ファイルを上書き |

**例:**

```bash
# すべてのシークレットを標準出力にエクスポート
secretctl export

# .env ファイルにエクスポート
secretctl export -o .env

# 特定のキーを JSON でエクスポート
secretctl export -k "aws/*" -f json -o config.json

# メタデータ付きでエクスポート
secretctl export -f json --with-metadata -o secrets.json

# 別のコマンドにパイプ
secretctl export -f json | jq '.DB_HOST'
```

---

## import

`.env` または JSON ファイルからシークレットをインポート。

```bash
secretctl import [file] [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--on-conflict string` | 既存キーの処理方法: `skip`, `overwrite`, `error`（デフォルト: `error`） |
| `--dry-run` | 変更なしでインポート内容をプレビュー |

**例:**

```bash
# .env ファイルからインポート
secretctl import .env

# JSON ファイルからインポート
secretctl import config.json

# インポートせずに変更をプレビュー
secretctl import .env --dry-run

# 既存キーをスキップ
secretctl import .env --on-conflict=skip

# 既存キーを上書き
secretctl import .env --on-conflict=overwrite
```

**サポート形式:**

- `.env` ファイル: 標準的な KEY=VALUE 形式
- JSON ファイル: キー・バリューペアのオブジェクト `{"KEY": "value"}`

---

## generate

暗号的に安全なランダムパスワードを生成。

```bash
secretctl generate [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `-l, --length int` | パスワード長（8-256、デフォルト: 24） |
| `-n, --count int` | 生成するパスワード数（1-100、デフォルト: 1） |
| `-c, --copy` | 最初のパスワードをクリップボードにコピー |
| `--exclude string` | 除外する文字 |
| `--no-uppercase` | 大文字を除外 |
| `--no-lowercase` | 小文字を除外 |
| `--no-numbers` | 数字を除外 |
| `--no-symbols` | 記号を除外 |

**例:**

```bash
# デフォルトパスワードを生成（24文字）
secretctl generate

# 記号なしの32文字パスワードを生成
secretctl generate -l 32 --no-symbols

# 5つのパスワードを生成
secretctl generate -n 5

# 生成してクリップボードにコピー
secretctl generate -c

# 曖昧な文字を除外
secretctl generate --exclude "0O1lI"
```

---

## audit

監査ログを管理。

### audit list

監査ログエントリを一覧。

```bash
secretctl audit list [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--limit int` | 表示する最大イベント数（デフォルト: 100） |
| `--since string` | 指定期間以降のイベントを表示（例: `24h`） |

**例:**

```bash
secretctl audit list --limit=50 --since=24h
```

### audit verify

監査ログの HMAC チェーン整合性を検証。

```bash
secretctl audit verify
```

**例:**

```bash
$ secretctl audit verify
Audit log integrity verified. 1234 events checked.
```

### audit export

監査ログを JSON または CSV 形式でエクスポート。

```bash
secretctl audit export [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--format string` | 出力形式: `json`, `csv`（デフォルト: `json`） |
| `-o, --output string` | 出力ファイルパス（デフォルト: 標準出力） |
| `--since string` | 指定期間以降のイベントをエクスポート（例: `30d`） |
| `--until string` | 指定日までのイベントをエクスポート（RFC 3339） |

**例:**

```bash
secretctl audit export --format=csv -o audit.csv --since=30d
```

### audit prune

古い監査ログエントリを削除。

```bash
secretctl audit prune [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--older-than string` | 指定期間より古いログを削除（例: `12m` で12ヶ月） |
| `--dry-run` | 削除せずに削除対象を表示 |
| `-f, --force` | 確認プロンプトをスキップ |

**例:**

```bash
# 削除対象をプレビュー
secretctl audit prune --older-than=12m --dry-run

# 確認付きで削除
secretctl audit prune --older-than=12m

# 確認なしで削除
secretctl audit prune --older-than=12m --force
```

---

## backup

Vault の暗号化バックアップを作成。

```bash
secretctl backup [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `-o, --output string` | 出力ファイルパス（`--stdout` 使用時以外は必須） |
| `--stdout` | 標準出力に出力（パイプ用） |
| `--with-audit` | バックアップに監査ログを含める |
| `--backup-password` | 別のバックアップパスワードを使用（プロンプト） |
| `--key-file string` | 暗号化キーファイル（32バイト） |
| `-f, --force` | 確認なしで既存ファイルを上書き |

**例:**

```bash
# 基本的なバックアップ
secretctl backup -o vault-backup.enc

# 監査ログ付きバックアップ
secretctl backup -o full-backup.enc --with-audit

# 標準出力へバックアップ（gpg などへパイプ）
secretctl backup --stdout | gpg --encrypt > backup.gpg

# 別のバックアップパスワードを使用
secretctl backup -o backup.enc --backup-password

# 自動化用にキーファイルを使用
secretctl backup -o backup.enc --key-file=backup.key

# 既存バックアップを上書き
secretctl backup -o backup.enc --force
```

---

## restore

暗号化バックアップから Vault を復元。

```bash
secretctl restore <backup-file> [flags]
```

**フラグ:**

| フラグ | 説明 |
|--------|------|
| `--dry-run` | 変更なしで復元内容をプレビュー |
| `--verify-only` | バックアップ整合性のみ検証（復元なし） |
| `--on-conflict string` | 既存キーの処理方法: `skip`, `overwrite`, `error`（デフォルト: `error`） |
| `--key-file string` | 復号キーファイル |
| `--with-audit` | 監査ログを復元（既存を上書き） |
| `-f, --force` | 確認プロンプトをスキップ |

**例:**

```bash
# バックアップ整合性を検証
secretctl restore backup.enc --verify-only

# 変更なしで復元をプレビュー
secretctl restore backup.enc --dry-run

# 復元、既存キーをスキップ
secretctl restore backup.enc --on-conflict=skip

# 復元、既存キーを上書き
secretctl restore backup.enc --on-conflict=overwrite

# 監査ログ付きで復元
secretctl restore backup.enc --with-audit

# 復号にキーファイルを使用
secretctl restore backup.enc --key-file=backup.key
```

---

## mcp-server

AI コーディングアシスタント連携用の MCP サーバーを起動。

```bash
secretctl mcp-server
```

**認証:**

起動前に `SECRETCTL_PASSWORD` 環境変数を設定:

```bash
SECRETCTL_PASSWORD=your-password secretctl mcp-server
```

**利用可能な MCP ツール:**

| ツール | 説明 |
|--------|------|
| `secret_list` | メタデータ付きでシークレットキーを一覧（値なし） |
| `secret_exists` | メタデータ付きでシークレットの存在を確認 |
| `secret_get_masked` | マスクされたシークレット値を取得（例: `****WXYZ`） |
| `secret_run` | シークレットを環境変数としてコマンドを実行 |

**ポリシー設定:**

`~/.secretctl/mcp-policy.yaml` を作成して許可コマンドを設定:

```yaml
version: 1
default_action: deny
allowed_commands:
  - aws
  - gcloud
  - kubectl
```

詳細な設定は [MCP 連携ガイド](/docs/guides/mcp/) を参照。
