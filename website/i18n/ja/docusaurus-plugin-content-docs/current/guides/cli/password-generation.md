---
title: パスワード生成
description: 暗号学的に安全なランダム生成を使用して安全なパスワードを生成。
sidebar_position: 5
---

# パスワード生成

`generate` コマンドは Go の `crypto/rand` パッケージを使用して暗号学的に安全なランダムパスワードを作成します。

## 前提条件

- [secretctl をインストール](/docs/getting-started/installation)

## 基本的な使い方

```bash
secretctl generate [flags]
```

### デフォルトパスワード

すべての文字タイプを含む24文字のパスワードを生成:

```bash
secretctl generate
```

**出力:**
```
x9K#mP2!nQ7@wR4$tY6&jL
```

デフォルトのパスワードには以下が含まれます:
- 小文字（a-z）
- 大文字（A-Z）
- 数字（0-9）
- 記号（`!@#$%^&*()_+-=[]{}|;:,.<>?`）

## コマンドオプション

| フラグ | 短縮形 | 説明 | デフォルト |
|-------|--------|------|----------|
| `--length` | `-l` | パスワード長（8-256） | 24 |
| `--count` | `-n` | パスワード数（1-100） | 1 |
| `--no-symbols` | | 記号を除外 | false |
| `--no-numbers` | | 数字を除外 | false |
| `--no-uppercase` | | 大文字を除外 | false |
| `--no-lowercase` | | 小文字を除外 | false |
| `--exclude` | | 除外する文字 | "" |
| `--copy` | `-c` | クリップボードにコピー | false |

## パスワード長のカスタマイズ

### 長いパスワード

```bash
# 高セキュリティ用の32文字パスワード
secretctl generate -l 32

# 暗号化キー用の64文字パスワード
secretctl generate -l 64
```

### 短いパスワード

```bash
# 最小8文字のパスワード
secretctl generate -l 8
```

:::caution
16文字未満のパスワードはブルートフォース攻撃に対して脆弱な可能性があります。機密アカウントには長いパスワードを使用してください。
:::

## 複数パスワードの生成

一度に複数のパスワードを生成:

```bash
# 5つのパスワードを生成
secretctl generate -n 5
```

**出力:**
```
kL9#mN2!pQ7@rS4$
tU6&vW8*xY0^zA2%
bC4(dE6)fG8+hI0-
jK2=lM4[nO6]pQ8{
rS0|tU2;vW4:xY6,
```

以下の用途に便利:
- チームオンボーディング用のパスワード作成
- テスト認証情報の生成
- サービス間のパスワードローテーション

## 文字セットのカスタマイズ

### 英数字のみ

記号を受け付けないシステム用:

```bash
secretctl generate --no-symbols
```

### 文字のみ

コードや識別子用:

```bash
secretctl generate --no-symbols --no-numbers
```

### 数字のみ

PIN や数値コード用:

```bash
secretctl generate --no-symbols --no-uppercase --no-lowercase -l 6
```

**出力:**
```
847291
```

### 大文字のみ

システムコードやライセンスキー用:

```bash
secretctl generate --no-symbols --no-numbers --no-lowercase
```

## 紛らわしい文字の除外

混同しやすい文字を削除:

```bash
# 0/O, 1/l/I など混同しやすい文字を除外
secretctl generate --exclude "0O1lI"
```

以下の用途に便利:
- 声に出して読み上げたり手動で入力するパスワード
- QR コードや印刷物
- 手動入力でのユーザーエラー削減

### 一般的な除外セット

```bash
# 紛らわしい文字を除外
secretctl generate --exclude "0O1lI"

# シェルで問題となる文字を除外
secretctl generate --exclude '$`"'\''\\!'

# XML/HTML 特殊文字を除外
secretctl generate --exclude "<>&'\""
```

## クリップボード連携

生成したパスワードを直接クリップボードにコピー:

```bash
secretctl generate -c
```

**出力:**
```
kL9#mN2!pQ7@rS4$tU6&vW8*
WARNING: Password copied to clipboard is accessible by all processes
         Clipboard will not be automatically cleared. Overwrite manually when done.
Password copied to clipboard
```

### プラットフォームサポート

| プラットフォーム | クリップボードツール |
|-----------------|-------------------|
| macOS | `pbcopy`（組み込み） |
| Linux | `xclip` または `xsel` |
| Windows | `clip`（組み込み） |

:::caution
クリップボードは実行中のすべてのプロセスからアクセス可能です。使用後は他のコンテンツをコピーしてクリアしてください。
:::

### Linux セットアップ

Linux でクリップボードサポートをインストール:

```bash
# Debian/Ubuntu
sudo apt install xclip

# Fedora
sudo dnf install xclip

# Arch
sudo pacman -S xclip
```

## 実践的な例

### データベースパスワード

データベース用の強力な32文字パスワード:

```bash
secretctl generate -l 32 | secretctl set DB_PASSWORD
```

### API キー形式

API キーに適した英数字文字列を生成:

```bash
secretctl generate -l 40 --no-symbols
```

### WiFi パスワード

共有しやすい人間が読めるパスワード:

```bash
secretctl generate -l 16 --exclude "0O1lI"
```

### SSH キーパスフレーズ

SSH キー保護用の強力なパスフレーズ:

```bash
secretctl generate -l 24 -c
```

### チームオンボーディング

新しいチームメンバー用の一時パスワードを生成:

```bash
# オンボーディング用に10個のパスワードを生成
secretctl generate -n 10 -l 16 --no-symbols
```

### AWS シークレットキー形式

AWS スタイルのシークレットアクセスキーをシミュレート:

```bash
secretctl generate -l 40 --no-symbols
```

### バックアップ暗号化キー

バックアップ暗号化用の高エントロピーキーを生成:

```bash
secretctl generate -l 64 --no-symbols
```

## パスワード強度

### エントロピー計算

パスワード強度はエントロピーのビット数で測定されます:

| 構成 | 文字セットサイズ | 24文字のエントロピー |
|------|-----------------|---------------------|
| すべての文字 | 94 | ～157ビット |
| 記号なし | 62 | ～143ビット |
| 文字のみ | 52 | ～137ビット |
| 英数小文字 | 36 | ～124ビット |

エントロピーが高い = より強いパスワード。

### 推奨長

| 用途 | 最小長 | 推奨 |
|------|--------|------|
| 個人アカウント | 12 | 16+ |
| データベースパスワード | 24 | 32+ |
| API キー/トークン | 32 | 40+ |
| 暗号化キー | 32 | 64+ |
| マスターパスワード | 16 | 24+ |

## 他のコマンドとの組み合わせ

### 生成して保存

```bash
# 新しいパスワードを生成して保存
secretctl generate | secretctl set SERVICE_PASSWORD \
  --notes="Auto-generated on $(date)" \
  --expires="90d"
```

### エクスポート用に生成

```bash
# 複数のパスワードを生成してエクスポート
secretctl generate -n 5 | while read pw; do
  echo "Temp password: $pw"
done
```

### パスワードローテーション

```bash
# 新しいパスワードを生成して既存のシークレットを更新
secretctl generate -l 32 | secretctl set DB_PASSWORD \
  --notes="Rotated on $(date +%Y-%m-%d)"
```

## セキュリティ上の考慮事項

### 暗号化セキュリティ

secretctl は Go の `crypto/rand` パッケージを使用:
- オペレーティングシステムの暗号学的乱数生成器を使用
- セキュリティに敏感なアプリケーションに適している
- `math/rand` のような擬似乱数生成器は決して使用しない

### クリップボードセキュリティ

`--copy` 使用時:
- パスワードは実行中のすべてのプロセスからアクセス可能
- クリップボードは自動的にクリアされない
- 使用後は常にクリップボードを上書き

### ログ記録を避ける

生成したパスワードをログに記録しないよう注意:

```bash
# 悪い例: パスワードがシェル履歴に表示される
echo $(secretctl generate)

# 良い例: 直接パイプするかクリップボードを使用
secretctl generate | secretctl set MY_SECRET
secretctl generate -c
```

### シェル履歴

`generate` コマンド自体は、コマンドラインにパスワードが含まれないため、シェル履歴に対して安全です。

## トラブルシューティング

### "clipboard tool not found" エラー

Linux では、クリップボードツールをインストール:

```bash
# xclip をインストール
sudo apt install xclip

# または xsel をインストール
sudo apt install xsel
```

### "password length must be at least 8" エラー

最小パスワード長は8文字:

```bash
# これは失敗
secretctl generate -l 4

# 最小8を使用
secretctl generate -l 8
```

### "character set is empty" エラー

少なくとも1つの文字タイプが有効であることを確認:

```bash
# これは失敗 - すべての文字タイプが除外
secretctl generate --no-symbols --no-numbers --no-uppercase --no-lowercase

# 少なくとも1つのタイプを含める
secretctl generate --no-symbols --no-numbers
```

### クリップボードが動作しない

クリップボードコマンドが利用可能か確認:

```bash
# macOS
which pbcopy

# Linux
which xclip || which xsel

# Windows (PowerShell)
Get-Command clip
```

## 次のステップ

- [シークレットの管理](/docs/guides/cli/managing-secrets) - 生成したパスワードを保存
- [コマンドの実行](/docs/guides/cli/running-commands) - コマンドでシークレットを使用
- [CLI コマンドリファレンス](/docs/reference/cli-commands) - 完全なコマンドリファレンス
