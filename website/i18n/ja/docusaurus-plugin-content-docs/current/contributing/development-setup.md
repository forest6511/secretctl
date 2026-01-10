---
title: 開発セットアップ
description: secretctl への貢献のための開発環境をセットアップする。
sidebar_position: 2
---

# 開発セットアップ

このガイドでは、secretctl への貢献のための開発環境をセットアップする方法を説明します。

## 前提条件

### 必須

- **Go 1.24+** - [ダウンロード](https://go.dev/dl/)
- **Git** - バージョン管理用
- **Make**（オプション）- ビルド自動化用

### デスクトップアプリ開発用

- **Node.js 18+** - フロントエンド開発用
- **Wails v2** - [インストール](https://wails.io/docs/gettingstarted/installation)

## クイックスタート

### 1. リポジトリのクローン

```bash
git clone https://github.com/forest6511/secretctl.git
cd secretctl
```

### 2. 依存関係のインストール

```bash
# Go 依存関係
go mod download

# ビルドの確認
go build ./...
```

### 3. テストの実行

```bash
# すべてのテスト
go test ./...

# race detector 付き
go test -race ./...

# カバレッジ付き
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 4. リンターの実行

```bash
# 必要に応じて golangci-lint をインストール
# https://golangci-lint.run/usage/install/

golangci-lint run ./...
```

## プロジェクト構造

```
secretctl/
├── cmd/                    # CLI コマンド（Cobra）
│   └── secretctl/
├── internal/               # 内部パッケージ
│   ├── cli/               # CLI ユーティリティ
│   ├── config/            # 設定処理
│   └── mcp/               # MCP サーバー実装
├── pkg/                    # 公開パッケージ
│   ├── audit/             # 監査ログ
│   ├── backup/            # バックアップとリストア
│   ├── crypto/            # 暗号操作
│   ├── secret/            # シークレット型
│   └── vault/             # Vault 操作
├── desktop/               # Wails デスクトップアプリ
│   ├── frontend/          # React + TypeScript
│   └── *.go               # Go バックエンド
└── website/               # ドキュメントサイト
```

## ビルド

### CLI バイナリ

```bash
# 開発ビルド
go build -o bin/secretctl ./cmd/secretctl

# ローカルで実行
./bin/secretctl --help
```

### デスクトップアプリ

```bash
cd desktop

# 開発モード（ホットリロード）
wails dev

# 本番ビルド
wails build
```

## 開発ワークフロー

### 1. 機能ブランチの作成

```bash
git checkout -b feature/your-feature-name
```

### 2. 変更の実施

- 明確で読みやすいコードを書く
- 既存のパターンに従う
- 新機能にはテストを追加
- 必要に応じてドキュメントを更新

### 3. 変更のテスト

```bash
# すべてのテストを実行
go test ./...

# 特定のパッケージテストを実行
go test ./pkg/vault/...

# 詳細出力で実行
go test -v ./...
```

### 4. リントとフォーマット

```bash
# コードをフォーマット
gofmt -w .
goimports -w .

# リンターを実行
golangci-lint run ./...
```

### 5. コミットとプッシュ

[Conventional Commits](https://www.conventionalcommits.org/) を使用：

```bash
git add .
git commit -m "feat: add password strength indicator"
git push origin feature/your-feature-name
```

## テストのヒント

### テーブル駆動テスト

secretctl はテーブル駆動テストを広く使用しています：

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {"valid", "SecurePass123!", false},
        {"too short", "short", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### インテグレーションテスト

実際の Vault が必要なテストの場合：

```go
func TestVaultOperations(t *testing.T) {
    // 一時ディレクトリを作成
    tmpDir := t.TempDir()

    // Vault を初期化
    v := vault.New(tmpDir)
    err := v.Create("test-password-123!")
    require.NoError(t, err)

    // テストロジック...
}
```

## セキュリティに関する考慮事項

セキュリティ関連のコードを貢献する際：

- **シークレットをログに出力しない** - ログにパスワード、キー、機密データを含めない
- **crypto/rand を使用** - セキュリティ目的で `math/rand` を使用しない
- **機密データを消去** - パスワードとキーには `crypto.SecureWipe()` を使用
- **エラーを処理** - 暗号操作のエラーを無視しない

## ヘルプを得る

- パターンと規約については既存のコードを読む
- コンテキストについては [issues](https://github.com/forest6511/secretctl/issues) を確認
- 質問は [discussion](https://github.com/forest6511/secretctl/discussions) を開く

## 次のステップ

- [コントリビュートガイドライン](/docs/contributing) - 完全なコントリビュートガイド
- [アーキテクチャ概要](/docs/architecture) - システム設計
- [セキュリティ設計](/docs/security/encryption) - 暗号の詳細
