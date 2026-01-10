---
title: AI エージェント連携
description: MCP サーバーを通じて secretctl を Claude Code やその他の AI コーディングアシスタントと連携させる。
sidebar_position: 3
---

# AI エージェント連携

secretctl は Model Context Protocol (MCP) を通じて AI コーディングアシスタントをファーストクラスでサポートします。このガイドでは、Claude Code やその他の MCP 互換ツールとの連携を説明します。

## 概要

### 課題

AI コーディングアシスタントは以下のようなタスクでシークレットへのアクセスが必要です：
- API キーが必要なテストの実行
- クラウドサービスへのデプロイ
- データベースへのアクセス
- 外部サービスとの認証

### 従来のアプローチの問題

AI エージェントに生のシークレットを公開するとリスクが生じます：
- **非決定的な動作** - LLM が意図せずシークレットを露出する可能性
- **プロンプトインジェクション** - 悪意のあるプロンプトがシークレットを抽出する可能性
- **ログ露出** - 会話ログにシークレットが表示される可能性
- **失効の困難さ** - 露出した認証情報の無効化が困難

### secretctl のソリューション：AI安全設計

secretctl は「露出なしアクセス」原則に従います：

| 機能 | 従来 | secretctl MCP |
|------|------|---------------|
| 生のシークレットアクセス | あり | **なし** |
| コマンド実行 | 手動 | **自動化** |
| 出力サニタイズ | なし | **あり** |
| 監査ログ | なし | **あり** |
| ポリシー制御 | なし | **あり** |

AI エージェントはシークレットを**見ずに** **使用**できます。

## Claude Code セットアップ

### 1. MCP サーバーの設定

Claude Code 設定に secretctl を追加：

```json
// ~/.claude.json
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

### 2. MCP ポリシーの作成

許可される操作を設定：

```yaml
# ~/.secretctl/mcp-policy.yaml
version: 1
default_action: deny

# AI がシークレット付きで実行できるコマンド
allowed_commands:
  - npm
  - node
  - python
  - go
  - cargo
  - aws
  - gcloud
  - kubectl
  - curl
  - ./deploy.sh
  - ./test.sh

# 決して許可されないコマンド
denied_commands:
  - rm
  - sudo
  - chmod
  - chown

# 環境エイリアス
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### 3. ファイル権限の設定

```bash
# ポリシーファイルはセキュアな権限が必要
chmod 600 ~/.secretctl/mcp-policy.yaml
```

### 4. セットアップの確認

Claude Code を再起動し、MCP サーバーが接続されていることを確認します。

## 利用可能な MCP ツール

### secret_list

すべてのシークレットキーを一覧表示（値は非公開）：

```
Claude: 「どのシークレットが利用可能？」
→ secret_list を呼び出し
→ 返却: ["API_KEY", "DATABASE_URL", "AWS_ACCESS_KEY"]
```

### secret_exists

特定のシークレットの存在を確認：

```
Claude: 「AWS 認証情報は設定されている？」
→ secret_exists("aws/access_key") を呼び出し
→ 返却: true/false
```

### secret_get_masked

メタデータとマスクされた値を取得：

```
Claude: 「API キーの情報を表示して」
→ secret_get_masked("API_KEY") を呼び出し
→ 返却: {
    key: "API_KEY",
    maskedValue: "****xyz",
    tags: ["api", "prod"],
    notes: "OpenAI API key"
  }
```

### secret_run

シークレットを注入してコマンドを実行：

```
Claude: 「API キーでテストを実行して」
→ secret_run({
    keys: ["API_KEY"],
    command: "npm test"
  }) を呼び出し
→ 環境に API_KEY を設定して実行
→ 出力はサニタイズ（シークレットは編集済み）
```

## 実践シナリオ

### テストの実行

```
あなた: 「インテグレーションテストを実行して」

Claude:
1. secret_list を呼び出し → TEST_API_KEY, TEST_DATABASE_URL を検出
2. secret_run({
     keys: ["TEST_API_KEY", "TEST_DATABASE_URL"],
     command: "npm run test:integration"
   }) を呼び出し
3. サニタイズされたテスト出力を返却
```

### AWS へのデプロイ

```
あなた: 「本番にデプロイして」

Claude:
1. secret_exists("aws/*") を呼び出し → AWS 認証情報の存在を確認
2. secret_run({
     keys: ["aws/*"],
     command: "aws s3 sync ./dist s3://my-bucket"
   }) を呼び出し
3. デプロイ結果を返却（認証情報は表示されない）
```

### データベースクエリ

```
あなた: 「データベースのユーザー数を確認して」

Claude:
1. secret_run({
     keys: ["DATABASE_URL"],
     command: "psql $DATABASE_URL -c 'SELECT COUNT(*) FROM users'"
   }) を呼び出し
2. クエリ結果を返却
```

### API 呼び出し

```
あなた: 「GitHub リポジトリを取得して」

Claude:
1. secret_run({
     keys: ["GITHUB_TOKEN"],
     command: "curl -H 'Authorization: Bearer $GITHUB_TOKEN' https://api.github.com/user/repos"
   }) を呼び出し
2. リポジトリ一覧を返却（トークンは露出しない）
```

## セキュリティモデル

### AI エージェントが**できる**こと

- ✅ 利用可能なシークレットキーの一覧表示
- ✅ シークレットの存在確認
- ✅ マスクされた値の閲覧（末尾4文字）
- ✅ メタデータの読み取り（タグ、メモ、URL）
- ✅ 許可されたコマンドをシークレット付きで実行
- ✅ サニタイズされた出力の受信

### AI エージェントが**できない**こと

- ❌ 平文のシークレット値の読み取り
- ❌ 拒否されたコマンドの実行
- ❌ 出力サニタイズのバイパス
- ❌ ポリシー外のシークレットへのアクセス
- ❌ シークレットの変更や削除

### 出力サニタイズ

すべてのコマンド出力はシークレット値がスキャンされます：

```
# API_KEY = "sk-abc123xyz" の場合
# 元の出力: "Connected with key sk-abc123xyz"
# サニタイズ後: "Connected with key [REDACTED:API_KEY]"
```

### 監査ログ

すべての MCP 操作がログに記録されます：

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "action": "secret.run",
  "source": "mcp",
  "keys": ["API_KEY"],
  "command": "npm test",
  "success": true
}
```

## 環境エイリアス

コンテキストに応じたシークレットマッピングに環境エイリアスを使用：

### 設定

```yaml
# ~/.secretctl/mcp-policy.yaml
env_aliases:
  development:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
  production:
    - pattern: "db/*"
      target: "prod/db/*"
```

### 使用方法

```
あなた: 「ステージングに対してテストを実行して」

Claude:
→ secret_run({
    keys: ["db/*"],
    command: "npm test",
    env: "staging"
  }) を呼び出し
→ staging/db/* シークレットが自動的に使用される
```

## ポリシーのベストプラクティス

### 最小権限の原則

AI が実際に必要なコマンドのみを許可：

```yaml
# 良い例: 特定の許可コマンド
allowed_commands:
  - npm test
  - npm run build
  - ./deploy.sh

# 悪い例: 許容範囲が広すぎる
allowed_commands:
  - "*"
```

### 危険なコマンドのブロック

潜在的に危険な操作は常に拒否：

```yaml
denied_commands:
  - rm
  - sudo
  - chmod
  - chown
  - mv
  - dd
  - mkfs
```

### 環境の分離

異なる環境には別々のシークレットを使用：

```yaml
env_aliases:
  dev:
    - pattern: "*"
      target: "dev/*"
  prod:
    - pattern: "*"
      target: "prod/*"
```

### 定期的な監査レビュー

AI エージェントのアクティビティを監視：

```bash
# MCP アクセスをレビュー
secretctl audit export | jq '.[] | select(.source == "mcp")'

# 失敗を確認
secretctl audit export | jq '.[] | select(.success == false)'
```

## トラブルシューティング

### MCP サーバーが接続しない

1. 設定の secretctl パスを確認
2. SECRETCTL_PASSWORD が設定されているか確認
3. Vault が存在しアクセス可能か確認
4. Claude Code のログでエラーを確認

### 「Command not allowed」エラー

ポリシーの `allowed_commands` にコマンドを追加：

```yaml
allowed_commands:
  - your-command
```

### 「Secret not found」エラー

1. シークレットの存在を確認: `secretctl list`
2. キーのスペルを確認（大文字小文字を区別）
3. パターンがマッチするか確認: `secretctl list | grep pattern`

### 出力がサニタイズされない

サニタイズは完全一致でのみ機能します。シークレットが表示される場合：
1. シークレット値が正しく保存されているか確認
2. 出力形式が保存された値と一致するか確認

### Permission denied

```bash
# ポリシーファイルの権限を確認
ls -la ~/.secretctl/mcp-policy.yaml
# -rw------- (600) であるべき

chmod 600 ~/.secretctl/mcp-policy.yaml
```

## 代替手段との比較

### vs. ハードコードされたシークレット

| 側面 | ハードコード | secretctl MCP |
|------|-------------|---------------|
| セキュリティ | ❌ コードで露出 | ✅ 決して露出しない |
| 監査 | ❌ なし | ✅ 完全な監査証跡 |
| ローテーション | ❌ コード変更が必要 | ✅ Vault の更新のみ |

### vs. 環境変数

| 側面 | 環境変数 | secretctl MCP |
|------|----------|---------------|
| AI の可視性 | ❌ 可視 | ✅ 非表示 |
| 出力の安全性 | ❌ 漏洩の可能性 | ✅ サニタイズ済み |
| ポリシー制御 | ❌ なし | ✅ 完全制御 |

### vs. 他のシークレットマネージャー

| 側面 | 他 | secretctl MCP |
|------|------|---------------|
| AI ネイティブ | ❌ 後付け | ✅ 組み込み |
| ローカルファースト | ❌ クラウド依存 | ✅ オフライン対応 |
| ゼロ露出 | ❌ API がシークレットを返す | ✅ 実行専用モデル |

## 次のステップ

- [開発者ワークフロー](/docs/use-cases/developer-workflows) - ローカル開発と CI/CD
- [MCP ツールリファレンス](/docs/reference/mcp-tools) - 完全なツール仕様
- [設定](/docs/reference/configuration) - ポリシー設定の詳細
