---
title: .env ファイルから
description: .env ファイルから secretctl にシークレットをインポートする。
sidebar_position: 2
---

# .env ファイルからの移行

このガイドでは、既存の `.env` ファイルを secretctl の暗号化 Vault に移行する方法を説明します。

## なぜ移行するのか？

`.env` ファイルにはいくつかのセキュリティ上の問題があります：

- **平文で保存** - ファイルアクセス権を持つ誰でもシークレットを読める
- **git にコミットされがち** - 誤って認証情報を漏洩しやすい
- **アクセス制御なし** - 誰がどのシークレットを見るか制限できない
- **監査証跡なし** - シークレットがいつアクセスまたは変更されたか記録されない

secretctl はこれらの問題を解決します：

- **AES-256-GCM 暗号化** - 保存時に暗号化されたシークレット
- **マスターパスワード保護** - 認証されたユーザーのみが復号可能
- **監査ログ** - シークレットのアクセスと変更の完全な証跡
- **AI安全設計** - Claude Code やその他の AI ツールで安全に使用

## クイックマイグレーション

### ステップ 1: Vault の初期化

まだの場合：

```bash
secretctl init
```

### ステップ 2: .env ファイルのインポート

```bash
# インポートされる内容をプレビュー
secretctl import .env --dry-run

# すべてのシークレットをインポート
secretctl import .env
```

### ステップ 3: インポートの確認

```bash
secretctl list
```

### ステップ 4: 古い .env ファイルの保護

確認後、古い `.env` ファイルを削除または保護：

```bash
# オプション 1: ファイルを削除
rm .env

# オプション 2: ローカル開発で必要な場合は .gitignore に追加
echo ".env" >> .gitignore
```

## コンフリクトの処理

既にシークレットがある Vault にインポートする場合：

```bash
# 既存のキーをスキップ（Vault の値を保持）
secretctl import .env --on-conflict=skip

# 既存のキーを上書き（.env の値を使用）
secretctl import .env --on-conflict=overwrite

# コンフリクトで停止（デフォルト動作）
secretctl import .env --on-conflict=error
```

## JSON からのインポート

secretctl は JSON 形式もサポートしています：

```bash
# キーと値のペアを含む JSON ファイル
# {"API_KEY": "sk-xxx", "DB_PASSWORD": "secret"}
secretctl import config.json
```

## 複数の環境ファイル

複数の環境を持つプロジェクトの場合：

```bash
# 環境プレフィックス付きでインポート
secretctl import .env.development --dry-run
secretctl import .env.production --dry-run

# または、キープレフィックスを使用して別々の「環境」にインポート
# インポート前にファイルを編集してキーをリネーム
```

## チームでの作業

移行後、チームメンバーは：

1. プロジェクトをクローン
2. 自分の Vault を初期化: `secretctl init`
3. 共有シークレットをインポート（セキュアなチャネルから、git からではなく）
4. `secretctl run` を使用してコマンドにシークレットを注入

## 次のステップ

- [Vault のバックアップ](/docs/guides/cli/backup-restore) - 暗号化されたシークレットを保護
- [AI ツールとの使用](/docs/getting-started/for-developers) - Claude Code 連携のセットアップ
- [必要に応じてエクスポート](/docs/reference/cli-commands#export) - レガシーツール用に `.env` ファイルを生成
