---
title: 一般ユーザー向け
description: secretctlデスクトップアプリで安全なパスワード管理を始める。
sidebar_position: 3
---

# はじめに: 一般ユーザー向け

このガイドは、デスクトップアプリまたは基本的なCLIコマンドを使用して、パスワード、APIキー、その他のシークレットをシンプルかつ安全に管理したいユーザー向けです。

## 学べること

- デスクトップアプリのインストールと設定
- 安全なVaultの作成
- シークレットの追加、表示、管理
- 資格情報を整理して安全に保つ

## なぜ secretctl？

- **完全にローカル**: シークレットはあなたのコンピューターから出ることはありません
- **アカウント不要**: サインアップなし、クラウド同期なし、サブスクリプションなし
- **強力な暗号化**: 軍事レベルのAES-256暗号化
- **シンプルなインターフェース**: 使いやすいデスクトップアプリケーション

## オプション 1: デスクトップアプリ（推奨）

### ソースからビルド

デスクトップアプリは現在ソースからのビルドが必要です。ビルド済みバイナリは将来のリリースで提供予定です。

**要件**:
- [Go 1.24+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2](https://wails.io/docs/gettingstarted/installation)

**ビルド手順**:

```bash
# リポジトリをクローン
git clone https://github.com/forest6511/secretctl.git
cd secretctl/desktop

# アプリをビルド
wails build
```

コンパイルされたアプリは `desktop/build/bin/` にあります。

### ステップ 3: Vaultを作成

1. secretctlアプリを開く
2. 「Create New Vault」をクリック
3. 強力なマスターパスワードを入力
4. パスワードを確認
5. 「Create」をクリック

:::tip マスターパスワードの選び方
- 最低8文字必須、12文字以上を推奨
- 文字、数字、記号を組み合わせる
- 「correct-horse-battery-staple」のようなパスフレーズの使用を検討
- **書き留めて**安全な場所に保管 - 忘れると復元できません！
:::

### ステップ 4: 最初のシークレットを追加

1. 「+」ボタンまたは「Add Secret」をクリック
2. 名前を入力（例: 「Gmail Password」または「OPENAI_API_KEY」）
3. シークレットの値を入力
4. オプションで追加:
   - **Notes**: 追加情報
   - **URL**: 関連ウェブサイト
   - **Tags**: 「email」「work」「api」などのカテゴリ
5. 「Save」をクリック

### ステップ 5: シークレットを表示・使用

- **表示**: シークレットをクリックして詳細を見る
- **コピー**: コピーアイコンをクリックでクリップボードにコピー（30秒後に自動消去）
- **検索**: 検索バーでシークレットをすばやく見つける
- **フィルター**: タグや日付でフィルター

### ステップ 6: セキュリティを維持

- アプリは15分間操作がないと自動ロック
- 手動ロック: ロックアイコンをクリックまたは `Cmd+L`（Mac）/ `Ctrl+L`（Windows）
- コンピューターから離れる前に必ずロック

## オプション 2: コマンドライン（CLI）

ターミナルを好む場合:

### インストール

**macOS (Apple Silicon)**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl
```

**macOS (Intel)**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-amd64
chmod +x secretctl-darwin-amd64
sudo mv secretctl-darwin-amd64 /usr/local/bin/secretctl
```

**Linux**:
```bash
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl
```

**Windows**: [GitHub Releases](https://github.com/forest6511/secretctl/releases) から `secretctl-windows-amd64.exe` をダウンロード。

### 基本コマンド

```bash
# Vaultを作成
secretctl init

# シークレットを追加
echo "my-password-123" | secretctl set "Gmail Password"

# シークレットを表示
secretctl get "Gmail Password"

# すべてのシークレットを一覧
secretctl list

# シークレットを削除
secretctl delete "Gmail Password"
```

## シークレットの整理

### キープレフィックスを使用

プレフィックスでカテゴリごとにシークレットを整理:

```
email/gmail
email/outlook
social/twitter
social/facebook
work/github
work/aws_key
banking/chase
```

### タグを使用

シークレット作成時にタグを追加:
- デスクトップ: タグフィールドに入力
- CLI: `echo "value" | secretctl set KEY --tags "work,api,important"`

### メモとURLを追加

各シークレットの使用場所を追跡:
- デスクトップ: メモとURLフィールドに入力
- CLI: `echo "value" | secretctl set KEY --notes "メインアカウント" --url "https://example.com"`

## シークレットのバックアップ

### バックアップを作成

**デスクトップ**: メニュー → File → Backup Vault

**CLI**:
```bash
secretctl backup -o ~/Desktop/my-secrets-backup.enc
```

バックアップはマスターパスワードで暗号化されます。

### バックアップから復元

**デスクトップ**: メニュー → File → Restore from Backup

**CLI**:
```bash
secretctl restore ~/Desktop/my-secrets-backup.enc
```

## キーボードショートカット（デスクトップアプリ）

| 操作 | macOS | Windows/Linux |
|------|-------|---------------|
| Vaultをロック | `Cmd+L` | `Ctrl+L` |
| 新規シークレット | `Cmd+N` | `Ctrl+N` |
| 検索 | `Cmd+F` | `Ctrl+F` |
| 値をコピー | `Cmd+C` | `Ctrl+C` |
| 終了 | `Cmd+Q` | `Ctrl+Q` |

## よくある質問

### マスターパスワードを忘れた場合は？

残念ながら、復元オプションはありません。マスターパスワードはシークレットを復号する唯一の方法です。以下を推奨します:
- 書き留めて安全な場所に保管
- 覚えやすいパスフレーズを使用

### シークレットはクラウドに同期される？

いいえ。secretctlは完全にローカルです。シークレットは暗号化ファイルとしてあなたのコンピューターにのみ保存されます。

### 複数のコンピューターで使える？

はい、ただしVaultを手動で転送する必要があります:
1. コンピューターAでバックアップを作成
2. バックアップファイルをコンピューターBにコピー
3. コンピューターBでバックアップを復元

### 安全？

はい。secretctlは以下を使用:
- AES-256-GCM暗号化（銀行や政府と同じ）
- Argon2id鍵導出（パスワードクラッキングから保護）
- ローカルのみの保存（ネットワークへの露出なし）

## 次のステップ

- [デスクトップアプリガイド](/docs/guides/desktop/) - 完全なデスクトップアプリドキュメント
- [CLIガイド](/docs/guides/cli/) - その他のCLIコマンドを学ぶ
- [セキュリティ概要](/docs/security/) - データがどのように保護されているか理解
- [FAQ](/docs/help/faq) - その他のよくある質問
