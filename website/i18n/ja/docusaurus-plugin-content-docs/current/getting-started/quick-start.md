---
title: クイックスタート
description: 5分以内に最初のシークレットを作成。
sidebar_position: 3
---

# クイックスタート

5分でsecretctlを使い始めましょう。

## 1. Vaultを初期化

```bash
secretctl init
```

マスターパスワードの作成を求められます。このパスワードはすべてのシークレットを暗号化します。

:::caution
マスターパスワードを忘れないでください！紛失した場合、復元できません。
:::

## 2. 最初のシークレットを追加

```bash
echo "sk-your-api-key" | secretctl set api/openai
```

または対話形式で（値の入力を求められます）：

```bash
secretctl set api/openai
```

## 3. シークレットを取得

```bash
secretctl get api/openai
```

## 4. コマンドでシークレットを使用

```bash
secretctl run -k "api/*" -- your-command
```

シェル履歴に残さずに、シークレットを環境変数として注入します。

## 次のステップ

- [シークレット管理](/docs/guides/cli/managing-secrets) - すべてのシークレット操作を学ぶ
- [コマンド実行](/docs/guides/cli/running-commands) - 高度な環境変数注入
- [MCP連携](/docs/guides/mcp/) - AIエージェントと連携
