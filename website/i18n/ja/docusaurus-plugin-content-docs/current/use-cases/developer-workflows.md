---
title: 開発者ワークフロー
description: ローカル開発、CI/CD、チームコラボレーションで secretctl を日常の開発ワークフローに活用する。
sidebar_position: 2
---

# 開発者ワークフロー

このガイドでは、ローカル開発から本番デプロイまで、secretctl がシークレット管理を効率化する一般的な開発シナリオを説明します。

## ローカル開発

### 散在する .env ファイルの置き換え

**問題:** プロジェクト全体に散在する `.env` ファイル、誤ってコミットしやすい、同期を保つのが難しい。

**解決策:** シークレットを secretctl Vault に一元化。

```bash
# 複数の .env ファイルを管理する代わりに
# シークレットを Vault に一度保存
echo "sk-abc123" | secretctl set OPENAI_API_KEY
echo "postgres://user:pass@localhost/db" | secretctl set DATABASE_URL

# シークレットを注入してアプリを実行
secretctl run -k OPENAI_API_KEY -k DATABASE_URL -- npm start
```

### プロジェクト固有のシークレット

階層的なキーを使用してプロジェクトごとにシークレットを整理：

```bash
# プロジェクト A のシークレット
echo "key-a" | secretctl set projectA/api_key
echo "db-a" | secretctl set projectA/database_url

# プロジェクト B のシークレット
echo "key-b" | secretctl set projectB/api_key
echo "db-b" | secretctl set projectB/database_url

# プロジェクト固有のシークレットで実行
secretctl run -k "projectA/*" -- ./run-project-a.sh
secretctl run -k "projectB/*" -- ./run-project-b.sh
```

### 複数環境

環境エイリアスを使用してシームレスに環境を切り替え：

```yaml
# ~/.secretctl/mcp-policy.yaml
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

```bash
# 同じコマンドで異なる環境
secretctl run --env=dev -k "db/*" -- ./app
secretctl run --env=prod -k "db/*" -- ./app
```

## CI/CD 連携

### GitHub Actions

GitHub Actions ワークフローにシークレットを注入：

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install secretctl
        run: |
          curl -fsSL https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64 -o secretctl
          chmod +x secretctl
          sudo mv secretctl /usr/local/bin/

      - name: Initialize vault
        env:
          SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
        run: |
          # セキュアストレージから Vault を復元または初期化
          secretctl init

      - name: Deploy with secrets
        env:
          SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
        run: |
          secretctl run -k DEPLOY_TOKEN -k AWS_ACCESS_KEY -- ./deploy.sh
```

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  script:
    - curl -fsSL https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64 -o secretctl
    - chmod +x secretctl && mv secretctl /usr/local/bin/
    - secretctl run -k "deploy/*" -- ./deploy.sh
  variables:
    SECRETCTL_PASSWORD: $SECRETCTL_PASSWORD
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    environment {
        SECRETCTL_PASSWORD = credentials('secretctl-password')
    }
    stages {
        stage('Deploy') {
            steps {
                sh '''
                    secretctl run -k DEPLOY_TOKEN -- ./deploy.sh
                '''
            }
        }
    }
}
```

## Docker ワークフロー

### Docker Compose での開発

Docker Compose 用に `.env` を生成：

```bash
# Docker Compose 用にシークレットをエクスポート
secretctl export -k "docker/*" -o .env

# または docker-compose を直接実行
secretctl run -k "docker/*" -- docker-compose up
```

```yaml
# docker-compose.yml
services:
  app:
    build: .
    env_file:
      - .env
```

### Docker ビルド引数

ビルド引数としてシークレットを安全に渡す：

```bash
# Dockerfile にシークレットを埋め込まない
# 代わりにビルド時に渡す
secretctl run -k GITHUB_TOKEN -- docker build \
  --build-arg GITHUB_TOKEN=$GITHUB_TOKEN \
  -t myapp .
```

### Docker Run

ランタイムでシークレットを注入：

```bash
secretctl run -k "app/*" -- docker run \
  -e API_KEY=$API_KEY \
  -e DATABASE_URL=$DATABASE_URL \
  myapp
```

## データベース接続

### PostgreSQL

```bash
# データベース認証情報を保存
echo "prod-db.example.com" | secretctl set db/host
echo "myuser" | secretctl set db/user
echo "secret123" | secretctl set db/password

# psql で接続
secretctl run -k "db/*" -- psql "postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST/mydb"
```

### MySQL

```bash
secretctl run -k "mysql/*" -- mysql \
  -h $MYSQL_HOST \
  -u $MYSQL_USER \
  -p$MYSQL_PASSWORD \
  mydb
```

### データベースマイグレーション

```bash
# データベース認証情報でマイグレーションを実行
secretctl run -k DATABASE_URL -- npx prisma migrate deploy
secretctl run -k DATABASE_URL -- rails db:migrate
secretctl run -k "db/*" -- flyway migrate
```

## クラウドプロバイダー CLI

### AWS CLI

```bash
# AWS 認証情報を保存
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# AWS コマンドを実行
secretctl run -k "aws/*" -- aws s3 ls
secretctl run -k "aws/*" -- aws ec2 describe-instances
```

### Google Cloud

```bash
# サービスアカウントキーパスまたは認証情報を保存
echo "/path/to/service-account.json" | secretctl set gcp/credentials_file

# gcloud コマンドを実行
secretctl run -k "gcp/*" -- gcloud compute instances list
```

### Azure CLI

```bash
# Azure 認証情報を保存
echo "your-client-id" | secretctl set azure/client_id
echo "your-client-secret" | secretctl set azure/client_secret

secretctl run -k "azure/*" -- az vm list
```

## API 開発

### curl での API テスト

```bash
# API キーを保存
echo "sk-live-xxx" | secretctl set stripe/api_key
echo "Bearer xxx" | secretctl set github/token

# 認証済みリクエストを実行
secretctl run -k "stripe/*" -- curl -u $STRIPE_API_KEY: https://api.stripe.com/v1/charges

secretctl run -k "github/*" -- curl -H "Authorization: $GITHUB_TOKEN" https://api.github.com/user
```

### Postman/Insomnia

API テストツール用にシークレットをエクスポート：

```bash
# インポート用に JSON でエクスポート
secretctl export -k "api/*" --format=json -o api-secrets.json
```

## Kubernetes ワークフロー

### Kubernetes Secrets の作成

```bash
# エクスポートして k8s secret を作成
secretctl export -k "app/*" --format=json | \
  kubectl create secret generic app-secrets --from-env-file=/dev/stdin

# または個別のキーを使用
secretctl run -k "k8s/*" -- kubectl create secret generic db-creds \
  --from-literal=username=$DB_USER \
  --from-literal=password=$DB_PASSWORD
```

### Helm デプロイメント

```bash
# Helm にシークレットを渡す
secretctl run -k "helm/*" -- helm upgrade myapp ./chart \
  --set database.password=$DB_PASSWORD \
  --set api.key=$API_KEY
```

## Terraform ワークフロー

### 環境変数

```bash
# Terraform は TF_VAR_* 環境変数を読み取る
secretctl run -k "terraform/*" -- terraform apply

# シークレットは以下のように保存:
# terraform/TF_VAR_db_password -> TF_VAR_DB_PASSWORD
```

### バックエンド設定

```bash
# リモート state バックエンドを設定
secretctl run -k AWS_ACCESS_KEY -k AWS_SECRET_KEY -- terraform init
```

## ベストプラクティス

### 命名規則

シークレットを一貫して整理：

```
# サービス別
aws/access_key
aws/secret_key
stripe/api_key
stripe/webhook_secret

# 環境別
dev/database_url
staging/database_url
prod/database_url

# プロジェクト別
projectA/api_key
projectB/api_key
```

### ローテーションワークフロー

```bash
# 新しいパスワードを生成
NEW_PASS=$(secretctl generate -l 32)

# シークレットを更新
echo "$NEW_PASS" | secretctl set db/password \
  --notes="Rotated on $(date +%Y-%m-%d)"

# サービスを再起動
secretctl run -k "db/*" -- ./restart-services.sh
```

### チームオンボーディング

```bash
# 新しいチームメンバー用に必要なシークレットを文書化
secretctl list > required-secrets.txt

# 新メンバーは自分の Vault を作成してシークレットを追加
secretctl init
echo "my-api-key" | secretctl set API_KEY
```

### 監査証跡

```bash
# アクセスパターンをレビュー
secretctl audit export --format=json -o audit-$(date +%Y%m%d).json

# 異常なアクセスをチェック
secretctl audit export | jq '.[] | select(.source == "mcp")'
```

## トラブルシューティング

### シークレットが見つからない

```bash
# シークレットの存在を確認
secretctl list | grep API_KEY

# 正確なキー名を確認（大文字小文字を区別）
secretctl get API_KEY
```

### 環境変数が設定されない

```bash
# デバッグ: 注入された変数を出力
secretctl run -k "api/*" -- env | grep API

# ワイルドカードパターンがマッチするか確認
secretctl list | grep "api/"
```

### CI/CD パスワードの問題

```bash
# SECRETCTL_PASSWORD が設定されていることを確認
if [ -z "$SECRETCTL_PASSWORD" ]; then
  echo "Error: SECRETCTL_PASSWORD not set"
  exit 1
fi
```

## 次のステップ

- [AI エージェント連携](/docs/use-cases/ai-agent-integration) - MCP と Claude Code
- [コマンド実行](/docs/guides/cli/running-commands) - 詳細な run コマンドガイド
- [シークレットのエクスポート](/docs/guides/cli/exporting-secrets) - エクスポート形式とオプション
