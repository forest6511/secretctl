---
title: よくある質問
description: secretctl についてのよくある質問。
sidebar_position: 1
---

# よくある質問

## 一般

### secretctl とは？

secretctl は開発者向けに設計されたローカルファーストのシングルバイナリシークレットマネージャーです。CLI、デスクトップアプリ、AI（MCP）連携を提供しながら、シークレットを暗号化して外部サービスに露出させません。

### なぜ別のシークレットマネージャーが必要なのですか？

クラウドベースのソリューションとは異なり、secretctl は:
- シークレットをローカルに保存（外部サーバーなし）
- オフラインで動作
- AI安全なシークレット注入を提供（シークレットは LLM に露出しない）
- 依存関係のないシングルバイナリとして配布

### secretctl は .env ファイルとどう違いますか？

| 項目 | .env ファイル | secretctl |
|------|-------------|-----------|
| **ストレージ** | ディスク上に平文 | AES-256-GCM 暗号化 |
| **Git 安全性** | 誤ってコミットしやすい | 平文の機密情報なし |
| **アクセス制御** | ファイルアクセス権を持つ誰でも | マスターパスワード必要 |
| **AI 連携** | シークレットを直接露出 | AI安全設計（平文を露出しない） |
| **監査証跡** | なし | HMAC チェーンによる改ざん検出可能なログ |
| **メタデータ** | なし | タグ、メモ、有効期限、URL |

secretctl は .env ファイルの利便性と本物のセキュリティを提供します。

### secretctl は無料ですか？

はい、secretctl はオープンソースであり、個人および商用利用で無料です。

### Homebrew や Scoop でインストールできますか？

はい！secretctl はパッケージマネージャーをサポートしています:

**macOS/Linux (Homebrew):**
```bash
brew install forest6511/tap/secretctl
```

**Windows (Scoop):**
```bash
scoop bucket add secretctl https://github.com/forest6511/scoop-bucket
scoop install secretctl
```

または、[GitHub Releases](https://github.com/forest6511/secretctl/releases) からバイナリをダウンロードできます。

## セキュリティ

### secretctl はどの暗号化を使用していますか？

secretctl は以下を使用:
- **AES-256-GCM** 認証付き暗号化
- **Argon2id** 鍵導出（OWASP 推奨パラメータ: 64MB メモリ、3イテレーション）
- **SQLite** 暗号化ストレージ

### MCP 経由で平文のシークレットを取得できないのはなぜですか？

これは意図的です。MCP サーバーは AI エージェントが平文シークレットを受け取らない「AI安全設計」セキュリティモデルに従います。代わりに:
- `secret_run` はシークレットを環境変数として注入
- `secret_get_masked` はマスクされた値を返す（例: `****WXYZ`）
- 出力は偶発的なシークレット露出を防ぐためにサニタイズ

これは 1Password の「Access Without Exposure」哲学に沿っています。

### シークレットはどこに保存されますか？

すべてのデータは `~/.secretctl/` にローカル保存:
- `vault.db` - 暗号化された SQLite データベース
- `audit/` - 月次 JSONL 監査ログを含むディレクトリ（例: `2025-01.jsonl`）

### Vault をバックアップできますか？

はい、`~/.secretctl/` ディレクトリをバックアップできます。バックアップには暗号化されたデータが含まれるため、シークレットにアクセスするにはマスターパスワードが必要です。

## CLI の使用

### どうやって始めればいいですか？

```bash
# 新しい Vault を初期化
secretctl init

# シークレットを追加
secretctl set my-api-key

# シークレットを取得
secretctl get my-api-key

# シークレットを注入してコマンドを実行
secretctl run -k "api/*" -- ./my-app
```

### `get` と `run` の違いは何ですか？

- `get` はシークレット値を直接出力（人間用）
- `run` はシークレットを環境変数としてサブプロセスに注入（自動化用）

コマンド履歴やログにシークレットを露出させずにプログラムに渡したい場合は `run` を使用します。

### ワイルドカードは使えますか？

はい、ワイルドカードは `run`、`export`、`delete` で動作します:

```bash
# aws/* にマッチするすべてのシークレットを注入
secretctl run -k "aws/*" -- ./deploy.sh

# すべてのデータベースシークレットをエクスポート
secretctl export -k "db/*" -f env
```

## デスクトップアプリ

### デスクトップアプリはどうやってインストールしますか？

現在、ビルド済みバイナリはまだ利用できません。ソースからビルドしてください:

```bash
cd desktop
wails build
```

詳細は[デスクトップアプリガイド](/docs/guides/desktop)を参照してください。

### デスクトップアプリは CLI とシークレットを共有しますか？

はい、両方とも `~/.secretctl/` の同じ Vault を使用します。一方で作成したシークレットはもう一方ですぐに利用可能です。

### なぜ Vault は自動ロックしますか？

セキュリティのため、Vault は15分間の非アクティブ後に自動的にロックされます。マウスまたはキーボードのアクティビティでタイマーがリセットされます。

## MCP 連携

### Claude Code で secretctl を使うにはどうすればいいですか？

Claude Code 設定に MCP サーバーを追加:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"]
    }
  }
}
```

### どの MCP ツールが利用可能ですか？

| ツール | 説明 |
|--------|------|
| `secret_list` | メタデータ付きですべてのシークレットキーを一覧 |
| `secret_exists` | シークレットが存在するか確認 |
| `secret_get_masked` | マスクされた値を取得（例: `****WXYZ`） |
| `secret_run` | シークレットを環境変数としてコマンドを実行 |
| `secret_list_fields` | マルチフィールドシークレットのフィールド名を一覧 |
| `secret_get_field` | 非機密フィールドの値のみを取得 |
| `secret_run_with_bindings` | 定義済み環境バインディングで実行 |

### なぜ `secret_get` MCP ツールがないのですか？

設計上、secretctl は AI エージェントに平文シークレットを露出しません。代わりに `secret_run` を使用してシークレットをサブプロセスに注入してください。

## トラブルシューティング

### "vault not initialized" エラー

`secretctl init` を実行してマスターパスワード付きの新しい Vault を作成してください。

### "decryption failed" エラー

これは通常、マスターパスワードが間違っていることを意味します。パスワードは `secretctl init` 時に設定され、すべての操作に必要です。

### Vault をリセットするにはどうすればいいですか？

マスターパスワードを忘れた場合、Vault を削除してやり直す必要があります:

```bash
rm -rf ~/.secretctl
secretctl init
```

**警告**: これは保存されているすべてのシークレットを永久に削除します。

### 監査ログに "chain broken" 警告が表示される

これは監査ログが改ざんされた可能性を示しています。シークレットは安全なままですが、原因を調査する必要があります。

## ロードマップ

### どの機能が利用可能ですか？

**Phase 2.5（マルチフィールドシークレット）** - ✅ リリース済み:
- シークレットごとに複数のフィールドを保存（例: ユーザー名 + パスワード + ホスト）
- 一般的なシークレットタイプ用の定義済みテンプレート（Login、Database、API、SSH）
- MCP 連携用のフィールドレベル感度制御
- `secret_run` 用の環境変数バインディング

**CLI でマルチフィールドシークレットを使用:**

```bash
# マルチフィールドシークレットを作成
secretctl set db/prod --field host=db.example.com --field user=myuser --field password=secret123

# 特定のフィールドを取得
secretctl get db/prod --field host

# バインディング付きで実行
secretctl run -k db/prod -- ./my-app
```

**デスクトップアプリでテンプレートを使用:**
1. 「シークレットを追加」をクリック
2. テンプレートを選択（Login、Database、API、SSH）
3. フィールドとバインディングは自動設定
4. 値を入力して保存

最新の開発状況は[プロジェクトロードマップ](https://github.com/forest6511/secretctl)を参照してください。
