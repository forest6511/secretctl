---
title: CLI ガイド
description: secretctl コマンドラインインターフェースをマスターする。
sidebar_position: 1
---

# CLI ガイド

コマンドラインから secretctl を使用する方法を学びます。

## クイックリファレンス

```bash
# Vault を初期化
secretctl init

# シークレットを管理
echo "value" | secretctl set KEY
secretctl get KEY
secretctl list
secretctl delete KEY

# シークレット付きでコマンドを実行
secretctl run -k KEY -- your-command

# シークレットをエクスポート
secretctl export -o .env

# パスワードを生成
secretctl generate
```

## ガイド

- [コマンドの実行](/docs/guides/cli/running-commands) - シークレットを環境変数としてコマンドを実行
- [パスワード生成](/docs/guides/cli/password-generation) - 安全なランダムパスワードを生成

## リファレンス

完全なコマンドドキュメントは [CLI コマンドリファレンス](/docs/reference/cli-commands) を参照してください。
