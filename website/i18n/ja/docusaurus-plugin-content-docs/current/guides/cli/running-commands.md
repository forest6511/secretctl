---
title: シークレット付きコマンド実行
description: コマンド実行時にシークレットを環境変数として注入する方法。
sidebar_position: 3
---

# シークレット付きコマンド実行

`run` コマンドは、シークレットを環境変数として注入して任意のコマンドを実行します。これにより、シークレットがシェル履歴やコマンドライン引数に残りません。

## 前提条件

- [secretctl をインストール](/docs/getting-started/installation)
- [シークレットを Vault に保存](/docs/guides/cli/managing-secrets)

## 基本的な使い方

```bash
secretctl run -k <key> -- <command> [args...]
```

`--` は secretctl のフラグと実行したいコマンドを分離します。

### 単一シークレット

```bash
# API_KEY を注入して curl を実行
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" https://api.example.com
```

### 複数シークレット

```bash
# 複数のシークレットを注入
secretctl run -k DB_HOST -k DB_USER -k DB_PASS -- psql -h $DB_HOST -U $DB_USER
```

## ワイルドカードパターン

glob パターンを使用して複数のシークレットを一度に注入できます。

### 単一レベルワイルドカード

`*` は単一レベルのシークレットにマッチします:

```bash
# aws/* は aws/access_key, aws/secret_key にマッチ
secretctl run -k "aws/*" -- aws s3 ls

# db/* は db/host, db/password にマッチ（db/prod/host にはマッチしない）
secretctl run -k "db/*" -- ./connect.sh
```

:::info
シェル展開を防ぐため、ワイルドカードパターンは常にクォートしてください: `-k "aws/*"` であって `-k aws/*` ではありません
:::

### パターン例

| パターン | マッチ | マッチしない |
|---------|-------|-------------|
| `aws/*` | `aws/access_key`, `aws/secret_key` | `aws/prod/key` |
| `db/*` | `db/host`, `db/password` | `db/prod/host` |
| `API_KEY` | `API_KEY`（完全一致） | `API_KEY_DEV` |

## 環境変数の命名

シークレットキーは有効な環境変数名に変換されます:

- `/` は `_` に置換
- `-` は `_` に置換
- 名前は大文字に変換

| シークレットキー | 環境変数 |
|-----------------|---------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |
| `API_KEY` | `API_KEY` |

## 出力サニタイズ

デフォルトで、secretctl はコマンド出力をスキャンしてシークレット値を編集します:

```bash
$ secretctl run -k DB_PASSWORD -- echo "Password is $DB_PASSWORD"
Password is [REDACTED:DB_PASSWORD]
```

これにより、ログやターミナル出力への偶発的なシークレット漏洩を防ぎます。

### サニタイズを無効化

デバッグや生の出力が必要な場合:

```bash
secretctl run -k API_KEY --no-sanitize -- ./script.sh
```

:::caution
`--no-sanitize` は慎重に使用してください。シークレットがターミナル出力、ログ、または画面録画にキャプチャされる可能性があります。
:::

## コマンドタイムアウト

長時間実行されるコマンドを防ぐためにタイムアウトを設定:

```bash
# 30秒後にタイムアウト
secretctl run -k API_KEY --timeout=30s -- ./slow-script.sh

# 5分後にタイムアウト（デフォルト）
secretctl run -k API_KEY --timeout=5m -- ./deploy.sh
```

### タイムアウト形式

| 形式 | 時間 |
|------|------|
| `30s` | 30秒 |
| `5m` | 5分 |
| `1h` | 1時間 |

## 環境エイリアス

環境エイリアスを使用して、スクリプトを変更せずにシークレットを異なる環境にマップします。

### 設定

まず、`~/.secretctl/mcp-policy.yaml` でエイリアスを設定:

```yaml
version: 1
default_action: allow

env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### 使い方

```bash
# dev/db/host, dev/db/password を使用
secretctl run --env=dev -k "db/*" -- ./app

# prod/db/host, prod/db/password を使用
secretctl run --env=prod -k "db/*" -- ./app
```

これにより、同じコマンドが変更なしで異なる環境で動作します。

## 環境変数プレフィックス

注入されるすべての環境変数にプレフィックスを追加:

```bash
# 変数は APP_API_KEY, APP_DB_PASSWORD になる
secretctl run -k API_KEY -k DB_PASSWORD --env-prefix=APP_ -- ./app
```

アプリケーションが特定のプレフィックスを期待する場合に便利です。

## 実践的な例

### AWS CLI

```bash
# AWS 認証情報を保存
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# AWS コマンドを実行
secretctl run -k "aws/*" -- aws s3 ls
secretctl run -k "aws/*" -- aws ec2 describe-instances
```

### Docker

```bash
# Docker ビルドにシークレットを渡す
secretctl run -k GITHUB_TOKEN -- docker build \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t myapp .

# シークレット付きでコンテナを実行
secretctl run -k "db/*" -- docker run \
  -e DB_HOST=$DB_HOST \
  -e DB_PASSWORD=$DB_PASSWORD \
  myapp
```

### Node.js / npm

```bash
# シークレット付きで npm スクリプトを実行
secretctl run -k API_KEY -k DATABASE_URL -- npm start

# テスト認証情報でテストを実行
secretctl run -k "test/*" -- npm test
```

### データベース接続

```bash
# PostgreSQL
secretctl run -k DB_HOST -k DB_USER -k DB_PASSWORD -- \
  psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST/mydb"

# MySQL
secretctl run -k MYSQL_HOST -k MYSQL_USER -k MYSQL_PASSWORD -- \
  mysql -h $MYSQL_HOST -u $MYSQL_USER -p$MYSQL_PASSWORD
```

### CI/CD スクリプト

```bash
# デプロイスクリプト
secretctl run -k DEPLOY_TOKEN -k AWS_ACCESS_KEY -k AWS_SECRET_KEY -- ./deploy.sh

# 長時間のデプロイ用にタイムアウトを設定
secretctl run -k "deploy/*" --timeout=30m -- ./full-deploy.sh
```

## セキュリティ上の考慮事項

### シェル履歴内のシークレット

`run` コマンドはシークレットをシェル履歴に残しません:

```bash
# 良い例: シークレットが履歴に残らない
secretctl run -k API_KEY -- curl -H "Authorization: Bearer $API_KEY" ...

# 悪い例: シークレットが履歴に表示される
curl -H "Authorization: Bearer sk-abc123" ...
```

### プロセス環境

シークレットは子プロセスでのみ利用可能で、親シェルでは見えません:

```bash
# このコマンドの後、$API_KEY はシェルで設定されていない
secretctl run -k API_KEY -- ./script.sh
echo $API_KEY  # 空
```

### ブロックされるコマンド

セキュリティのため、特定のコマンドは常にブロックされます:

- `env` - すべての環境変数を公開する
- `printenv` - すべての環境変数を公開する
- `set` - 変数を公開する可能性がある
- `export` - シェルに漏洩する可能性がある

## トラブルシューティング

### "secret not found" エラー

シークレットが存在することを確認:

```bash
secretctl list
secretctl get MY_KEY
```

### "command not found" エラー

コマンドが PATH にあることを確認するか、フルパスを使用:

```bash
# フルパスを使用
secretctl run -k API_KEY -- /usr/local/bin/myapp

# または PATH が正しいことを確認
secretctl run -k API_KEY -- bash -c 'which myapp && myapp'
```

### ワイルドカードがマッチしない

`*` は1レベルのみにマッチすることを覚えておいてください:

```bash
# これは aws/key にマッチし、aws/prod/key にはマッチしない
secretctl run -k "aws/*" -- ./script.sh

# どのシークレットが存在するか確認
secretctl list | grep aws
```

### タイムアウトの問題

長時間実行されるコマンドにはタイムアウトを増やす:

```bash
# デフォルトは5分、必要に応じて増やす
secretctl run -k API_KEY --timeout=30m -- ./long-running-task.sh
```

## 次のステップ

- [シークレットのエクスポート](/docs/guides/cli/exporting-secrets) - .env または JSON ファイルにエクスポート
- [MCP 連携](/docs/guides/mcp/) - AI コーディングアシスタントでシークレットを使用
- [CLI コマンドリファレンス](/docs/reference/cli-commands) - 完全なコマンドリファレンス
