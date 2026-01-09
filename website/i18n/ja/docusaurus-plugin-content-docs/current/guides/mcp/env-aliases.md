---
title: 環境エイリアス
description: シークレットキーを異なる環境にマップ。
sidebar_position: 5
---

# 環境エイリアス

環境エイリアスにより、シークレットキーのパターンを変更せずに異なるシークレットプレフィックス（dev/staging/prod）間をシームレスに切り替えできます。これは異なる環境で作業する必要がある AI アシスタントに特に便利です。

## 概要

シークレットリクエストで環境固有のパスをハードコーディングする代わりに:

```json
// エイリアスなし - フルパスを指定する必要がある
{ "keys": ["prod/db/host", "prod/db/password"] }
```

エイリアスを使用してキーを動的にマップできます:

```json
// エイリアスあり - パターンと環境を指定
{ "keys": ["db/*"], "env": "prod" }
```

## 設定

ポリシーファイル（`~/.secretctl/mcp-policy.yaml`）でエイリアスを定義:

```yaml
version: 1
default_action: deny
allowed_commands:
  - kubectl
  - aws

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

## 仕組み

1. ポリシーファイルの `env_aliases` でエイリアスを定義
2. `secret_run` で `env` パラメータを使用してエイリアスを選択
3. シークレット検索前にキーパターンが変換される

**例:**

上記のポリシーで:

```json
{
  "command": "kubectl",
  "args": ["apply", "-f", "deployment.yaml"],
  "keys": ["db/*"],
  "env": "prod"
}
```

キーパターン `db/*` は `prod/db/*` に変換されるので、以下のようなシークレット:

- `prod/db/host`
- `prod/db/password`
- `prod/db/username`

...が環境変数として注入されます。

## パターンマッチング

| パターン | キー | 結果 |
|---------|------|------|
| `db/*` | `db/host` | サフィックス `host` にマッチ |
| `api/*` | `api/v1/key` | サフィックス `v1/key` にマッチ |
| `special_key` | `special_key` | 完全一致 |

## CLI での使用

`--env` フラグは CLI の `run` コマンドでも使用できます:

```bash
# dev 環境のシークレットを使用
secretctl run --env=dev -k "db/*" -- ./app

# prod 環境のシークレットを使用
secretctl run --env=prod -k "api/*" -- kubectl apply -f deployment.yaml
```

## ユースケース

### マルチ環境デプロイメント

```yaml
env_aliases:
  dev:
    - pattern: "k8s/*"
      target: "dev/k8s/*"
  staging:
    - pattern: "k8s/*"
      target: "staging/k8s/*"
  prod:
    - pattern: "k8s/*"
      target: "prod/k8s/*"
```

AI は任意の環境にデプロイできます:

```
"k8s/* シークレットを使って staging にアプリをデプロイして"
```

### データベース接続

```yaml
env_aliases:
  local:
    - pattern: "postgres/*"
      target: "local/postgres/*"
  cloud:
    - pattern: "postgres/*"
      target: "cloud/postgres/*"
```

### API キー管理

```yaml
env_aliases:
  test:
    - pattern: "stripe/*"
      target: "test/stripe/*"
  live:
    - pattern: "stripe/*"
      target: "live/stripe/*"
```

## ベストプラクティス

1. **一貫した命名を使用** - 環境間でパターン名を一貫させる
2. **エイリアスを文書化** - 各エイリアスの説明コメントを追加
3. **本番アクセスを制限** - 本番用に別のポリシーファイルを検討
4. **まず dev でテスト** - 本番前に dev でコマンドが動作することを確認
