---
title: コアコンセプト
description: Vault、シークレット、secretctlがデータを安全に保つ仕組みを理解。
sidebar_position: 4
---

# コアコンセプト

## Vault

Vaultはすべてのシークレットを保存する暗号化されたSQLiteデータベースです。デフォルトでは `~/.secretctl/vault.db` に配置されます。

## シークレット

シークレットはVaultに保存されるキー・バリューペアです。キーは整理のためにパス形式の記法を使用します：

```
api/openai
db/production/password
aws/access-key
```

## 暗号化

secretctlは業界標準の暗号化を使用します：

- **AES-256-GCM** でシークレット値を暗号化
- **Argon2id** でマスターパスワードから暗号化キーを導出

## AI安全設計（AIセキュリティ）

secretctlをAIエージェント（MCP）と使用する場合、シークレットは平文で公開されることはありません：

- `secret_run`: シークレットを環境変数として注入
- `secret_get_masked`: `****WXYZ` のようなマスク値を返却
- MCPで平文を取得する `secret_get` は存在しません

[AI安全設計について詳しく →](/docs/guides/mcp/security-model)
