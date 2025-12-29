# Multi-Field Secrets 機能拡張提案

## 調査サマリー

### 競合のデータモデル比較

| ツール | アイテム型 | カスタムフィールド | フィールドタイプ |
|--------|----------|------------------|-----------------|
| **1Password** | 20+種類 (Login, Database, API Credential, SSH Key等) | あり | Text, Concealed, URL, Date, Menu, Reference |
| **Bitwarden** | 4種類 (Login, Card, Identity, Secure Note) | あり | Text, Hidden, Boolean, Linked |
| **KeePass** | 1種類 (Entry) | あり (Custom Strings) | Text only |
| **HashiCorp Vault** | なし (KV store) | N/A | Nested JSON |
| **secretctl現状** | なし | なし | 単一値のみ |

Sources:
- [1Password Item Categories](https://support.1password.com/item-categories/)
- [1Password Item Fields](https://developer.1password.com/docs/cli/item-fields/)
- [Bitwarden Custom Fields](https://bitwarden.com/help/custom-fields/)
- [KeePass Field References](https://keepass.info/help/base/fieldrefs.html)

---

### 1Passwordのアイテム型詳細

| 型 | 主要フィールド |
|----|--------------|
| Login | username, password, url, totp |
| Database | type, server, port, database, username, password |
| API Credential | username, credential, hostname, type, validFrom, expires |
| SSH Key | public_key, private_key, passphrase |
| Server | url, username, password, admin_console_url |

---

### MCP統合の考慮事項

1. **HashiCorp Vault MCP**: Nested JSON構造 `{ "host": "...", "password": "..." }`
2. **Keeper Secrets Manager MCP**: Search by title, URL, username等
3. **MCP 2025-06-18仕様**: Structured data対応強化

**Best Practice**:
- フィールド単位でのアクセス制御
- 機密フィールド (password等) はOption D+で保護
- 非機密フィールド (host, port等) はAIに返却可能

---

## 提案: Phase 2.5 (Multi-Field Secrets)

### なぜPhase 3の前に必要か

```
Phase 0-2 (完了)        Phase 2.5 (新規)         Phase 3+ (将来)
─────────────────      ─────────────────        ─────────────────
単一値シークレット  →   マルチフィールド     →   チーム共有
                        アイテム型                クラウド同期

現状: key=value         提案: key={fields}       将来: shared vault
```

**理由**:
1. **ユーザー要求**: DB接続情報など、複数値が必要なユースケースが多い
2. **競合との差別化**: 1Password/Bitwardenレベルの機能が必要
3. **MCP強化**: フィールド単位でOption D+を適用可能に
4. **Phase 3への基盤**: チーム共有時にアイテム型は必須

---

## データモデル拡張

### 現状

```go
type Secret struct {
    Key       string    `json:"key"`
    Value     string    `json:"value"`      // 単一値
    Notes     string    `json:"notes"`
    URL       string    `json:"url"`
    Tags      []string  `json:"tags"`
    ExpiresAt *time.Time `json:"expiresAt"`
}
```

### 提案

```go
type Secret struct {
    Key       string            `json:"key"`
    Type      SecretType        `json:"type"`       // NEW: アイテム型
    Fields    map[string]Field  `json:"fields"`     // NEW: マルチフィールド
    Notes     string            `json:"notes"`
    URL       string            `json:"url"`
    Tags      []string          `json:"tags"`
    ExpiresAt *time.Time        `json:"expiresAt"`
}

type SecretType string

const (
    SecretTypePassword SecretType = "password"  // 従来互換
    SecretTypeLogin    SecretType = "login"
    SecretTypeDatabase SecretType = "database"
    SecretTypeAPI      SecretType = "api"
    SecretTypeSSH      SecretType = "ssh"
    SecretTypeCustom   SecretType = "custom"
)

type Field struct {
    Value      string    `json:"value"`
    Sensitive  bool      `json:"sensitive"`  // true = Option D+で保護
    Label      string    `json:"label"`      // 表示名
}
```

---

## アイテム型定義

### Login (Webサービス認証)

| フィールド | Sensitive | 説明 |
|-----------|-----------|------|
| username | false | ユーザー名/Email |
| password | **true** | パスワード |
| url | false | ログインURL |
| totp_secret | **true** | TOTP秘密鍵 |

### Database (データベース接続)

| フィールド | Sensitive | 説明 |
|-----------|-----------|------|
| type | false | postgres, mysql, etc. |
| host | false | ホスト名 |
| port | false | ポート番号 |
| database | false | データベース名 |
| username | false | DB ユーザー名 |
| password | **true** | DB パスワード |
| ssl_mode | false | SSL設定 |

### API (API認証)

| フィールド | Sensitive | 説明 |
|-----------|-----------|------|
| api_key | **true** | APIキー |
| api_secret | **true** | APIシークレット |
| endpoint | false | APIエンドポイント |
| description | false | 説明 |

### SSH (SSH接続)

| フィールド | Sensitive | 説明 |
|-----------|-----------|------|
| host | false | ホスト名 |
| port | false | ポート (default: 22) |
| username | false | ユーザー名 |
| private_key | **true** | 秘密鍵 |
| passphrase | **true** | パスフレーズ |

### Custom (カスタム)

| フィールド | Sensitive | 説明 |
|-----------|-----------|------|
| (任意) | (指定可能) | ユーザー定義 |

---

## MCP拡張 (Option D+ 準拠)

### 既存ツール変更

```
secret_list()
  → アイテム型情報を追加
  → { "key": "db/prod", "type": "database", "tags": [...] }

secret_get_masked(key)
  → 全フィールドをマスク表示
  → { "host": "db.example.com", "password": "****5678" }

secret_exists(key)
  → 変更なし
```

### 新規ツール

```
secret_get_field(key, field)
  → 非Sensitiveフィールドのみ返却
  → Sensitiveフィールドはエラー or マスク

secret_run_with_fields(key, command, field_mapping)
  → フィールドを環境変数にマッピング
  → { "DB_HOST": "host", "DB_PASS": "password" }
```

### MCP例: PostgreSQL接続

```
User: "本番DBに接続してテーブル一覧を取得して"

Claude: secret_get_field("db/production", "host")
        → "db.example.com"  (non-sensitive: OK)

        secret_get_field("db/production", "password")
        → ERROR: "Field 'password' is sensitive"

        secret_run_with_fields("db/production", "psql -c '\\dt'", {
            "PGHOST": "host",
            "PGPORT": "port",
            "PGDATABASE": "database",
            "PGUSER": "username",
            "PGPASSWORD": "password"
        })
        → (テーブル一覧が返る、パスワードはAIに見えない)
```

---

## CLI拡張

```bash
# アイテム作成
secretctl set db/prod --type=database
# → インタラクティブにフィールド入力

secretctl set db/prod --type=database \
  --field host=db.example.com \
  --field port=5432 \
  --field database=myapp \
  --field username=admin \
  --field password  # パスワードは標準入力から

# フィールド取得
secretctl get db/prod              # 全フィールド表示
secretctl get db/prod --field=host # 特定フィールドのみ

# 環境変数注入
secretctl run -k db/prod -- psql
# → PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD を注入
```

---

## Desktop App UI

```
┌─────────────────────────────────────────────────────────────────────┐
│ 🔐 db/production                                    [Edit] [Delete] │
├─────────────────────────────────────────────────────────────────────┤
│ Type: Database (PostgreSQL)                                         │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ Connection                                                      │ │
│ │   Host:     db.example.com                          [Copy]      │ │
│ │   Port:     5432                                    [Copy]      │ │
│ │   Database: myapp_prod                              [Copy]      │ │
│ │   SSL Mode: require                                 [Copy]      │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ Authentication                              🔒 Protected Fields │ │
│ │   Username: app_user                                [Copy]      │ │
│ │   Password: ••••••••••••                    [Show] [Copy]       │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ Tags: [production] [database] [critical]                            │
│ Notes: Primary production database - handle with care               │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ 📋 Quick Actions                                                │ │
│ │   [Copy Connection String]  [Export as .env]  [Test Connection] │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ Created: 2025-01-15  Updated: 2025-12-20  Expires: Never            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 移行戦略

### 既存データの自動変換

```
現状:
  Key: "AWS_SECRET_KEY"
  Value: "sk-xxxxx"

変換後:
  Key: "AWS_SECRET_KEY"
  Type: "password"  (デフォルト型)
  Fields: {
    "value": { Value: "sk-xxxxx", Sensitive: true }
  }
```

### 後方互換性

```bash
# 従来の使い方（引き続き動作）
secretctl set MY_SECRET
secretctl get MY_SECRET
secretctl run -k MY_SECRET -- cmd

# 新しい使い方
secretctl set MY_SECRET --type=login --field username=...
```

---

## 実装フェーズ

### Phase 2.5a: データモデル拡張 (基盤)

| 優先度 | タスク | 工数目安 |
|--------|--------|---------|
| P0 | Secret構造体拡張 | S |
| P0 | SQLiteスキーマ移行 | M |
| P0 | 既存データ自動変換 | S |
| P1 | アイテム型定義 (login, database, api, ssh) | S |
| P1 | フィールドバリデーション | S |

### Phase 2.5b: CLI/MCP拡張

| 優先度 | タスク | 工数目安 |
|--------|--------|---------|
| P0 | CLI `set --type --field` 対応 | M |
| P0 | CLI `get --field` 対応 | S |
| P1 | MCP `secret_get_field` ツール | M |
| P1 | MCP `secret_run_with_fields` ツール | M |
| P2 | 環境変数マッピング自動化 | M |

### Phase 2.5c: Desktop App対応

| 優先度 | タスク | 工数目安 |
|--------|--------|---------|
| P0 | アイテム型選択UI | M |
| P0 | フィールド編集フォーム | L |
| P1 | フィールドグループ表示 | M |
| P2 | Quick Actions (接続文字列コピー等) | M |

---

## 更新ロードマップ案

```
┌─────────────────────────────────────────────────────────────────────┐
│                    secretctl ロードマップ (更新版)                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Phase 0-2: ローカル完結版                       ✅ v0.6.0 完了     │
│  ─────────────────────────                                          │
│  - 単一値シークレット管理                                           │
│  - CLI + MCP + Desktop App                                          │
│  - Option D+ (AIに平文非公開)                                       │
│                                                                     │
│  Phase 2.5: Multi-Field Secrets                  📋 NEW (提案)      │
│  ──────────────────────────────                                     │
│  - アイテム型 (login, database, api, ssh, custom)                   │
│  - マルチフィールド対応                                             │
│  - フィールド単位のOption D+                                        │
│  - MCP拡張 (secret_get_field, secret_run_with_fields)              │
│  - Desktop App UI刷新                                               │
│                                                                     │
│  Phase 3: Team Edition                           📋 将来 (変更なし) │
│  ─────────────────────                                              │
│  - クラウド同期サービス                                             │
│  - チームVault共有                                                  │
│  - 監査ログエンタープライズ拡張                                     │
│                                                                     │
│  Phase 4: Enterprise                             📋 将来 (変更なし) │
│  ───────────────────                                                │
│  - SSO/SAML/OIDC統合                                                │
│  - RBAC                                                             │
│  - コンプライアンス監査ログ                                         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## PostgreSQL MCPユースケースへの回答

### 質問: 「ClaudeからPostgreSQLに接続したい」

### 回答: Phase 2.5で実現

```bash
# 1. DB接続情報を保存
secretctl set db/postgres/dev --type=database \
  --field type=postgres \
  --field host=localhost \
  --field port=5432 \
  --field database=myapp \
  --field username=dev_user \
  --field password  # 標準入力から

# 2. MCPで使用
# Claude: "開発DBのテーブル一覧を見せて"
# → secret_run_with_fields("db/postgres/dev", "psql -c '\\dt'", {...})
# → AIはパスワードを見ずにコマンド実行
```

### 責務の明確化

```
secretctl = Secrets Manager (認証情報管理)
          + Environment Injector (環境変数注入)

          ≠ Database Client (DB操作は外部ツール)
```

---

## 次のステップ

1. **この提案のレビュー**: 方向性の確認
2. **Phase 2.5の優先度決定**: P0タスクから着手
3. **詳細設計**: データモデル、API仕様の確定
4. **実装開始**: Phase 2.5a (データモデル拡張) から

---

## Sources

- [1Password Item Categories](https://support.1password.com/item-categories/)
- [1Password Item Fields](https://developer.1password.com/docs/cli/item-fields/)
- [Bitwarden Custom Fields](https://bitwarden.com/help/custom-fields/)
- [KeePass Field References](https://keepass.info/help/base/fieldrefs.html)
- [HashiCorp Vault MCP Server](https://developer.hashicorp.com/vault/docs/mcp-server/overview)
- [Keeper Secrets Manager MCP](https://docs.keeper.io/en/keeperpam/secrets-manager/integrations/model-context-protocol-mcp-for-ai-agents-node)
- [Astrix - State of MCP Server Security 2025](https://astrix.security/learn/blog/state-of-mcp-server-security-2025/)
- [WorkOS - Best Practices for MCP Secrets Management](https://workos.com/guide/best-practices-for-mcp-secrets-management)
