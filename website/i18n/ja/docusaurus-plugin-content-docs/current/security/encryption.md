---
title: 暗号化詳細
description: 暗号実装と仕様。
sidebar_position: 3
---

# 暗号化詳細

このガイドでは、secretctl で使用される暗号アルゴリズム、パラメータ、実装について詳細な情報を提供します。

## 暗号仕様

### サマリー

| コンポーネント | アルゴリズム | パラメータ |
|--------------|------------|-----------|
| 対称暗号 | AES-256-GCM | 256ビットキー、96ビット nonce |
| 鍵導出 | Argon2id | 64MB メモリ、3イテレーション、4スレッド |
| Salt | ランダム | 128ビット（16バイト） |
| Nonce | ランダム | 暗号化ごとに96ビット（12バイト） |
| HMAC（監査ログ） | HMAC-SHA256 | 256ビットキー |

### 標準準拠

| 仕様 | リファレンス |
|------|------------|
| AES-GCM | NIST SP 800-38D |
| Argon2 | RFC 9106 |
| OWASP | パスワードストレージチートシート |
| 鍵導出 | HKDF (RFC 5869) |

## AES-256-GCM

### なぜ AES-256-GCM か？

| プロパティ | メリット |
|-----------|---------|
| **認証付き暗号化** | 改ざんを自動検出 |
| **NIST 推奨** | 政府および業界標準 |
| **ハードウェアアクセラレーション** | 現代の CPU で高速（AES-NI） |
| **十分な検証** | 数十年の暗号解析 |

### 実装

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
)

func encrypt(plaintext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    // ランダム nonce を生成
    nonce := make([]byte, gcm.NonceSize()) // 12 バイト
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    // 暗号化と認証
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, errors.New("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

### Blob フォーマット

すべての暗号化データは一貫したフォーマットを使用:

```
┌──────────────┬─────────────────────┬────────────────┐
│ Nonce (12B)  │ 暗号文              │ GCM Tag (16B)  │
└──────────────┴─────────────────────┴────────────────┘
```

**適用対象:**
- `encrypted_dek` (vault_keys テーブル)
- `encrypted_key` (secrets テーブル)
- `encrypted_value` (secrets テーブル)
- `encrypted_metadata` (secrets テーブル)

### このフォーマットの利点

| 利点 | 説明 |
|------|------|
| 自己完結型 | 各 blob は独立して復号可能 |
| 別の nonce カラム不要 | シンプルなデータベーススキーマ |
| 業界標準 | libsodium sealed box と同じ |

## Argon2id 鍵導出

### なぜ Argon2id か？

| プロパティ | メリット |
|-----------|---------|
| **メモリハード** | GPU/ASIC に高コスト |
| **時間ハード** | 複数イテレーションが必要 |
| **並列性** | 複数コアを活用 |
| **RFC 標準** | RFC 9106 (2021) |

Argon2id は Argon2i（サイドチャネル耐性）と Argon2d（GPU 耐性）を組み合わせています。

### パラメータ

```go
import "golang.org/x/crypto/argon2"

const (
    memory      = 64 * 1024  // 64 MB
    iterations  = 3
    parallelism = 4
    keyLength   = 32         // 256 ビット
    saltLength  = 16         // 128 ビット
)

func deriveKey(password string, salt []byte) []byte {
    return argon2.IDKey(
        []byte(password),
        salt,
        iterations,
        memory,
        parallelism,
        keyLength,
    )
}
```

### OWASP 準拠

これらのパラメータは [OWASP パスワードストレージチートシート](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) に準拠:

| パラメータ | 値 | OWASP 推奨 |
|-----------|-----|-----------|
| メモリ | 64 MB | 64 MB 最小 |
| 時間 | 3 | 3 イテレーション |
| スレッド | 4 | 4（現代のマルチコア） |

### 環境別パフォーマンス

| 環境 | メモリ | アンロック時間 | ステータス |
|------|--------|--------------|----------|
| 現代の PC/Mac | 8GB+ | 0.5-1秒 | 最適 |
| 低スペックラップトップ | 4GB | 1-2秒 | 良好 |
| Raspberry Pi 4 | 2-8GB | 1-3秒 | 許容 |
| CI 環境 | 2-4GB | 1-2秒 | 良好 |
| Docker（制限あり） | 可変 | 可変 | 64MB+ 必要 |

:::warning Docker メモリ
64MB 未満のメモリを持つコンテナは鍵導出中に失敗します。コンテナに十分なメモリが割り当てられていることを確認してください。
:::

## Nonce 管理

### 一意性の保証

AES-GCM は一意の nonce が必要です。同じキーで nonce を再利用するとセキュリティが損なわれます（Forbidden Attack）。

### 戦略: ランダム Nonce

```go
func generateNonce() ([]byte, error) {
    nonce := make([]byte, 12) // 96 ビット
    if _, err := rand.Read(nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }
    return nonce, nil
}
```

### 衝突確率

| Nonce 長 | 衝突発生 | 確率 |
|---------|---------|------|
| 96ビット | 2^32 暗号化後 | 2^-33 ≈ 86億分の1 |

個人使用では、40億回の暗号化に達することは実質的に不可能です。

### ランダム性ソース

```go
import "crypto/rand"
```

Go の `crypto/rand` は以下を使用:
- **Linux**: `/dev/urandom` (CSPRNG)
- **macOS**: `getentropy()` (カーネルエントロピー)
- **Windows**: `CryptGenRandom()` (CryptoAPI)

すべて暗号学的に安全な疑似乱数生成器です。

## 鍵階層詳細

### 3層構造

```
┌─────────────────────────────────────────────────────────────────────┐
│                         鍵階層                                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  第1層: マスターパスワード（ユーザー入力）                              │
│          │                                                          │
│          │ 保存されない、導出にのみ使用                                │
│          │                                                          │
│          ▼                                                          │
│  第2層: マスターキー（導出）                                           │
│          │                                                          │
│          │ = Argon2id(password, salt)                               │
│          │ セッション中のみメモリに存在                                │
│          │                                                          │
│          ▼                                                          │
│  第3層: データ暗号化キー (DEK)                                        │
│          │                                                          │
│          │ 保存形式: AES-GCM(DEK, MasterKey)                        │
│          │ すべてのシークレットの暗号化に使用                          │
│          │                                                          │
│          ▼                                                          │
│  シークレット: AES-GCM(secret, DEK)                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### なぜこの設計か？

| 機能 | メリット |
|------|---------|
| **パスワードローテーション** | すべてのシークレットを再暗号化せずにパスワード変更 |
| **防御を深める** | 1つの層の侵害ですべてが露出しない |
| **セッション分離** | ロック後にマスターキーをクリア |

### パスワードローテーションフロー

```
1. Vault をアンロック（旧マスターキーを導出）
2. 旧マスターキーで DEK を復号
3. 新しい salt を生成
4. 新パスワードから新マスターキーを導出
5. 新マスターキーで DEK を再暗号化
6. 新しい暗号化された DEK と salt を保存
7. シークレットは変更なし（同じ DEK で暗号化されたまま）
```

## データベーススキーマ

### テーブル

```sql
-- 暗号化された DEK ストレージ
CREATE TABLE vault_keys (
    id INTEGER PRIMARY KEY,
    encrypted_dek BLOB NOT NULL,  -- nonce || AES-GCM 暗号文
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- シークレットストレージ
CREATE TABLE secrets (
    id INTEGER PRIMARY KEY,
    key_hash TEXT UNIQUE NOT NULL,     -- SHA-256(key) ルックアップ用
    encrypted_key BLOB NOT NULL,       -- nonce || 暗号化されたキー名
    encrypted_value BLOB NOT NULL,     -- nonce || 暗号化された値
    encrypted_metadata BLOB,           -- nonce || 暗号化された JSON（オプション）
    tags TEXT,                         -- カンマ区切り、平文
    expires_at TIMESTAMP,              -- クエリ用平文
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- HMAC チェーン付き監査ログ
CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY,
    action TEXT NOT NULL,              -- get, set, delete, list
    key_hash TEXT,                     -- アクセスしたキーの SHA-256
    source TEXT NOT NULL,              -- cli, mcp, ui
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    prev_hash TEXT NOT NULL,           -- 前のレコードハッシュ
    record_hash TEXT NOT NULL          -- このレコードの HMAC
);
```

### なぜ key_hash か？

平文のキー名の代わりに `SHA-256(key)` を保存:
- 復号せずにルックアップが可能
- 保存時のキー名を保護
- 一方向（ハッシュからキー名を導出できない）

## メタデータ暗号化

### 構造

```go
type SecretMetadata struct {
    Version int    `json:"v"`              // スキーマバージョン (1)
    Notes   string `json:"notes,omitempty"` // 最大 10KB
    URL     string `json:"url,omitempty"`   // 最大 2048 文字
}
```

### ストレージルール

| 条件 | encrypted_metadata |
|------|-------------------|
| notes="" AND url="" | NULL（暗号化なし） |
| notes OR url に値あり | nonce + AES-GCM(JSON) |

### MCP アクセス制限

AI安全設計はメタデータにも拡張:

| データ | MCP アクセス |
|--------|------------|
| `key` (名前) | あり（secret_list 経由） |
| `tags` | あり（平文、検索用） |
| `expires_at` | あり（平文、クエリ用） |
| `has_notes` | あり（ブールフラグのみ） |
| `has_url` | あり（ブールフラグのみ） |
| `notes` 内容 | **なし** |
| `url` 内容 | **なし** |
| `value` | **なし** |

**ルール**: 暗号化されたカラム = MCP アクセス禁止。

## 監査ログ整合性

### HMAC チェーン

```go
// マスターキーから監査キーを導出
auditKey := hkdf.Expand(sha256.New, masterKey, []byte("audit-log-v1"), 32)

// レコード HMAC を計算
func computeRecordHMAC(record AuditRecord, prevHash string, key []byte) string {
    data := fmt.Sprintf("%d|%s|%s|%s|%s|%s",
        record.ID,
        record.Action,
        record.KeyHash,
        record.Source,
        record.Timestamp.Format(time.RFC3339Nano),
        prevHash,
    )
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(data))
    return hex.EncodeToString(mac.Sum(nil))
}
```

### 検証

```go
func verifyChain(records []AuditRecord, key []byte) error {
    for i, r := range records {
        // HMAC を検証
        var prevHash string
        if i > 0 {
            prevHash = records[i-1].RecordHash
        }
        expected := computeRecordHMAC(r, prevHash, key)
        if r.RecordHash != expected {
            return fmt.Errorf("record %d: HMAC mismatch", r.ID)
        }

        // チェーンを検証
        if i > 0 && r.PrevHash != records[i-1].RecordHash {
            return fmt.Errorf("record %d: chain broken", r.ID)
        }
    }
    return nil
}
```

## 使用ライブラリ

### Go 標準ライブラリ

```go
import (
    "crypto/aes"           // AES 暗号化
    "crypto/cipher"        // GCM モード
    "crypto/hmac"          // HMAC
    "crypto/rand"          // セキュアランダム
    "crypto/sha256"        // SHA-256 ハッシュ
)
```

### golang.org/x/crypto

```go
import (
    "golang.org/x/crypto/argon2"  // 鍵導出
    "golang.org/x/crypto/hkdf"    // 鍵拡張
)
```

### なぜこれらのライブラリか？

| ライブラリ | 理由 |
|-----------|------|
| Go 標準ライブラリ | 実績あり、監査済み、メンテナンス済み |
| golang.org/x/crypto | 公式 Go 暗号拡張 |

サードパーティの暗号ライブラリは使用せず、サプライチェーンリスクを最小化。

## セキュリティ考慮事項

### 保護されるもの

| 資産 | 保護 |
|------|------|
| シークレット値 | AES-256-GCM 暗号化 |
| キー名 | AES-256-GCM 暗号化 |
| メタデータ | AES-256-GCM 暗号化 |
| マスターパスワード | 保存されない |
| 監査整合性 | HMAC チェーン |

### 保護されないもの

| 資産 | 理由 |
|------|------|
| タグ | 検索用に平文で保存 |
| 有効期限 | クエリ用に平文で保存 |
| レコード数 | DB から観察可能 |
| アクセスパターン | 監査ログで保護 |

### バックアップ考慮事項

```bash
# バックアップ安全（暗号化済み）
~/.secretctl/vault.db
~/.secretctl/vault.salt
~/.secretctl/vault.meta

# リストアに必要
# - 上記3ファイルすべて
# - マスターパスワード（保存されていない）
```

:::warning パスワード紛失
マスターパスワードを忘れた場合、シークレットは復元できません。これは設計によるものです - バックドアはありません。
:::

## 次のステップ

- [セキュリティ概要](/docs/security/) - 高レベルセキュリティアーキテクチャ
- [仕組み](/docs/security/how-it-works) - セキュリティデータフロー
- [MCP ツールリファレンス](/docs/reference/mcp-tools) - AI 連携セキュリティ
