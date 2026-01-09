---
title: バックアップとリストア
description: secretctl の Vault を安全にバックアップおよびリストアする方法。
sidebar_position: 4
---

# バックアップとリストア

secretctl は、データ損失からシークレットを保護し、Vault の移行を可能にする暗号化バックアップとリストア機能を提供します。

## 概要

- **暗号化バックアップ** — 各バックアップごとに新しいソルトを使用した AES-256-GCM 暗号化
- **整合性検証** — HMAC-SHA256 で改ざんを検出
- **アトミックリストア** — 部分的な状態を防ぐ全か無かのリストア
- **キーファイルサポート** — パスワードプロンプトなしで自動バックアップを有効化

## バックアップの作成

### 基本的なバックアップ

マスターパスワードで暗号化バックアップを作成:

```bash
secretctl backup -o vault-backup.enc
```

### 監査ログを含める

監査履歴を含む完全なバックアップ:

```bash
secretctl backup -o full-backup.enc --with-audit
```

### 別のバックアップパスワードを使用

バックアップに別のパスワードを使用（プロンプト表示）:

```bash
secretctl backup -o backup.enc --backup-password
```

### キーファイルを使用（自動化用）

自動バックアップスクリプト用に、パスワードプロンプトの代わりにキーファイルを使用:

```bash
# キーファイルを生成（一度だけ）
head -c 32 /dev/urandom > backup.key
chmod 600 backup.key

# キーファイルを使用してバックアップ
secretctl backup -o backup.enc --key-file=backup.key
```

:::warning
キーファイルはマスターパスワードと同様に安全に保管してください。キーファイルを持っている人は誰でもバックアップを復号できます。
:::

### 外部ツールへのパイプ

暗号化ツールへのパイプ用に stdout にバックアップ:

```bash
# GPG で暗号化
secretctl backup --stdout | gpg --encrypt -r you@email.com > backup.gpg

# 圧縮して暗号化
secretctl backup --stdout | gzip | gpg --encrypt > backup.gz.gpg
```

## バックアップのリストア

### まずバックアップを検証

リストア前に必ずバックアップの整合性を検証:

```bash
secretctl restore backup.enc --verify-only
```

出力にはバックアップメタデータが表示されます:
```
Backup verification successful!
  Version: 1
  Created: 2025-01-15 10:30:00
  Secrets: 42
  Includes Audit: true
```

### リストアのプレビュー（ドライラン）

変更を加えずにリストア内容を確認:

```bash
secretctl restore backup.enc --dry-run
```

### 空の Vault へのリストア

新規リストア（既存の Vault がない場合）:

```bash
secretctl restore backup.enc
```

### 競合の処理

既存の Vault にリストアする場合:

```bash
# 既存のキーをスキップ（新しいもののみ追加）
secretctl restore backup.enc --on-conflict=skip

# 既存のキーを上書き
secretctl restore backup.enc --on-conflict=overwrite

# 競合時にエラー（デフォルト）
secretctl restore backup.enc --on-conflict=error
```

### 監査ログ付きでリストア

バックアップから監査ログをリストア（既存を上書き）:

```bash
secretctl restore backup.enc --with-audit
```

### キーファイルを使用

キーファイルで作成されたバックアップの場合:

```bash
secretctl restore backup.enc --key-file=backup.key
```

## バックアップ形式

secretctl バックアップは安全なバージョン付き形式を使用:

| コンポーネント | 説明 |
|--------------|------|
| マジックナンバー | 8バイト識別子（`SCTL_BKP`） |
| ヘッダー | JSON メタデータ（バージョン、タイムスタンプ、暗号化モード） |
| 暗号化ペイロード | AES-256-GCM 暗号化 Vault データ |
| HMAC | HMAC-SHA256 整合性チェック |

### 鍵導出

パスワードベースのバックアップの場合:

1. バックアップごとに新しい32バイトソルトを生成
2. Argon2id KDF（64MB メモリ、3イテレーション、4スレッド）
3. HKDF-SHA256 で暗号化と MAC 用の別々の鍵を導出

## ベストプラクティス

### 定期的なバックアップ

cron を使用して自動バックアップをスケジュール:

```bash
# 毎日午前2時にバックアップ
0 2 * * * /usr/local/bin/secretctl backup \
  -o /backup/vault-$(date +\%Y\%m\%d).enc \
  --key-file=/etc/secretctl/backup.key
```

### バックアップローテーション

複数世代のバックアップを保持:

```bash
#!/bin/bash
BACKUP_DIR=/backup/secretctl
MAX_BACKUPS=7

# 新しいバックアップを作成
secretctl backup -o "$BACKUP_DIR/vault-$(date +%Y%m%d-%H%M%S).enc" \
  --key-file=/etc/secretctl/backup.key

# 古いバックアップを削除
ls -t "$BACKUP_DIR"/vault-*.enc | tail -n +$((MAX_BACKUPS + 1)) | xargs rm -f
```

### オフサイトストレージ

バックアップを複数の場所に保存:

```bash
# クラウドストレージにバックアップ
secretctl backup --stdout | aws s3 cp - s3://my-backups/vault-backup.enc

# リモートサーバーにバックアップ
secretctl backup --stdout | ssh backup-server "cat > /backup/vault.enc"
```

### リストアテスト

定期的にバックアップがリストアできることを検証:

```bash
# 一時的な場所にリストア
SECRETCTL_VAULT_DIR=/tmp/test-restore secretctl restore backup.enc

# シークレットを検証
SECRETCTL_VAULT_DIR=/tmp/test-restore secretctl list

# クリーンアップ
rm -rf /tmp/test-restore
```

## トラブルシューティング

### "Backup integrity check failed"

バックアップファイルが破損または改ざんされている可能性があります。別のバックアップからリストアしてください。

### "Invalid password or corrupted data"

パスワードが間違っているか、バックアップが破損しています。以下を試してください:

1. 正しいパスワードを使用していることを確認
2. キーファイルを使用している場合、元のファイルであることを確認
3. バックアップファイルが転送中に切り詰められていないか確認

### "Vault is locked by another process"

別の secretctl プロセスが Vault にアクセスしています。完了を待つか、古いロックファイルを確認してください。

## セキュリティ上の考慮事項

- **バックアップパスワード**は保存されません。覚えておく必要があります
- **キーファイル**はバックアップとは別に保存すべきです
- **監査ログ**にはアクセスパターンに関する機密情報が含まれる可能性があります
- **バックアップファイル**は制限されたパーミッション（0600）を持つべきです
