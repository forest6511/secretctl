---
title: トラブルシューティング
description: secretctl の一般的な問題と解決策。
sidebar_position: 2
---

# トラブルシューティング

一般的な問題とその解決策。

## インストールの問題

### "command not found: secretctl"

バイナリが PATH にありません。

**解決策:**

```bash
# インストール場所を確認
ls -la /usr/local/bin/secretctl

# またはインストールディレクトリを PATH に追加
export PATH=$PATH:/path/to/secretctl
```

### secretctl 実行時に Permission denied

バイナリに実行権限がありません。

**解決策:**

```bash
chmod +x /usr/local/bin/secretctl
```

## Vault の問題

### "vault not initialized"

Vault を作成する前に secretctl を使用しようとしています。

**解決策:**

```bash
secretctl init
```

### "failed to unlock vault: invalid password"

入力したパスワードが間違っています。

**解決策:**

- パスワードを再確認
- パスワードは大文字小文字を区別
- パスワードを忘れた場合、バックアップから復元する必要があります

### "vault already exists"

既に存在する Vault を初期化しようとしています。

**解決策:**

```bash
# 新規に開始する場合（警告: 既存のシークレットを破壊）
rm -rf ~/.secretctl
secretctl init

# または別の Vault 場所を使用
secretctl init --vault-dir=/path/to/new/vault
```

## MCP サーバーの問題

### Claude Code が secretctl を認識しない

**チェック 1:** バイナリパスを確認

```bash
which secretctl
# 出力: /usr/local/bin/secretctl
```

**チェック 2:** Claude Code 設定で絶対パスを使用

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/usr/local/bin/secretctl",
      "args": ["mcp-server"]
    }
  }
}
```

**チェック 3:** 設定変更後に Claude Code を再起動

### "no password provided"

MCP サーバーは Vault をアンロックするためにパスワードが必要です。

**解決策:**

Claude Code 設定で `SECRETCTL_PASSWORD` 環境変数を設定:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

### MCP サーバー接続タイムアウト

サーバーの起動に時間がかかりすぎています。

**解決策:**

1. MCP サーバーを手動でテスト:
   ```bash
   SECRETCTL_PASSWORD=your-password secretctl mcp-server 2>&1 | head -5
   ```

2. Vault の問題を確認:
   ```bash
   secretctl list
   ```

## デスクトップアプリの問題

### アプリが起動しない

**macOS:** システム環境設定 > セキュリティとプライバシーでアプリを許可する必要があるかもしれません。

**Windows:** Windows Defender でアプリを許可する必要があるかもしれません。

**Linux:** バイナリに実行権限があるか確認:
```bash
chmod +x secretctl-desktop
```

### "Failed to connect to backend"

Go バックエンドがフロントエンドと通信していません。

**解決策:**

1. アプリを完全に閉じる
2. 古いロックファイルを削除:
   ```bash
   rm -f ~/.secretctl/*.lock
   ```
3. アプリを再起動

### セッションタイムアウトが厳しすぎる

アプリはデフォルトで15分間の非アクティブ後にロックされます。

**解決策:**

これはセキュリティ機能であり、無効化できません。アプリをアクティブに保つか、必要に応じてアンロックしてください。

## シークレット管理の問題

### "key already exists"

既に存在するキーでシークレットを作成しようとしています。

**解決策:**

```bash
# 既存のシークレットを更新
echo "new-value" | secretctl set existing-key

# または先に削除してから作成
secretctl delete existing-key
echo "new-value" | secretctl set existing-key
```

### `secretctl run` でシークレットが見つからない

キーパターンがどのシークレットにもマッチしない可能性があります。

**解決策:**

```bash
# すべてのシークレットを一覧して利用可能なキーを確認
secretctl list

# パターンがマッチするか確認
secretctl list | grep "aws"

# 正しいパターンを使用
secretctl run -k "aws/*" -- your-command
```

### 出力サニタイズが有効な出力を削除する

サニタイザーがコマンド出力でシークレットのようなパターンを検出しました。

**解決策:**

これはセキュリティ機能です。サニタイズなしで出力を見る必要がある場合:

```bash
# CLI を直接使用（MCP 経由ではなく）
secretctl get MY_SECRET

# またはファイルにエクスポート
secretctl export --format=env > .env
```

## バックアップ＆リストアの問題

### "invalid backup file"

バックアップファイルが破損しているか、有効な secretctl バックアップではありません。

**解決策:**

1. ファイルが存在し内容があることを確認:
   ```bash
   ls -la backup.enc
   ```

2. 最初に `--verify-only` で試す:
   ```bash
   secretctl restore backup.enc --verify-only
   ```

### リストア中に "password mismatch"

リストアに使用したパスワードがバックアップのパスワードと一致しません。

**解決策:**

バックアップ作成時に使用したのと同じパスワードを使用してください。

## パフォーマンスの問題

### Vault 操作が遅い

SQLite データベースが断片化している可能性があります。

**解決策:**

1. Vault をロック
2. バックアップを作成
3. バックアップからリストア（新しいデータベースを作成）

```bash
secretctl lock
secretctl backup -o backup.enc
rm -rf ~/.secretctl
secretctl restore backup.enc
```

## さらなるヘルプ

問題がここに記載されていない場合:

1. 類似の問題がないか [GitHub Issues](https://github.com/forest6511/secretctl/issues) を確認
2. よくある質問は [FAQ](/docs/help/faq) を参照
3. 以下を含めて新しい Issue を開く:
   - secretctl バージョン (`secretctl --version`)
   - オペレーティングシステムとバージョン
   - 再現手順
   - エラーメッセージ（シークレットは編集）
