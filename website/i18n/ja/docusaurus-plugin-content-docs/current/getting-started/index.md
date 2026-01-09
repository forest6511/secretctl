---
title: はじめに
description: 5分でsecretctlを使い始めましょう。
sidebar_position: 1
---

# secretctl をはじめよう

**secretctl** は最もシンプルなAI対応シークレットマネージャーです。MCP（Model Context Protocol）によるAIエージェント連携をネイティブサポートし、安全でローカル完結の資格情報管理を提供します。

## パスを選択

<div className="row">
  <div className="col col--6">
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>開発者向け</h3>
      </div>
      <div className="card__body">
        <p>Claude Codeとの連携、CI/CD自動化、MCPによるAI安全なシークレットアクセス。</p>
        <ul>
          <li>MCPサーバー設定</li>
          <li>Claude Code 連携</li>
          <li>環境変数注入</li>
          <li>API自動化</li>
        </ul>
      </div>
      <div className="card__footer">
        <a className="button button--primary button--block" href="./for-developers">
          開発者ガイド →
        </a>
      </div>
    </div>
  </div>
  <div className="col col--6">
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>一般ユーザー向け</h3>
      </div>
      <div className="card__body">
        <p>デスクトップアプリまたは基本的なCLIでシンプルで安全なパスワード管理。</p>
        <ul>
          <li>デスクトップアプリ設定</li>
          <li>パスワード整理</li>
          <li>バックアップと復元</li>
          <li>技術知識不要</li>
        </ul>
      </div>
      <div className="card__footer">
        <a className="button button--secondary button--block" href="./for-users">
          ユーザーガイド →
        </a>
      </div>
    </div>
  </div>
</div>

## なぜ secretctl？

- **ローカル完結**: シークレットはあなたのマシンから出ることはありません
- **AI対応**: AI安全設計の内蔵MCPサーバー（AIエージェントは平文を見ません）
- **シンプル**: 単一バイナリ、サーバー不要
- **安全**: Argon2id鍵導出によるAES-256-GCM暗号化

## クイックリンク

- [インストール](/docs/getting-started/installation) - システムにsecretctlをインストール
- [クイックスタート](/docs/getting-started/quick-start) - 5分で最初のシークレットを作成
- [コアコンセプト](/docs/getting-started/concepts) - Vault、シークレット、暗号化を理解

## クイックスタート（5分）

### オプション 1: デスクトップアプリ

1. [GitHub Releases](https://github.com/forest6511/secretctl/releases) からダウンロード
2. アプリを開いてVaultを作成
3. ビジュアルでシークレット管理を開始

[デスクトップガイドへ →](/docs/guides/desktop/)

### オプション 2: CLI

```bash
# ダウンロードしてインストール
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl

# Vaultを初期化
secretctl init

# 最初のシークレットを追加
echo "sk-..." | secretctl set OPENAI_API_KEY
```

[CLIガイドへ →](/docs/guides/cli/)

### オプション 3: AI/MCP 連携

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

[MCPガイドへ →](/docs/guides/mcp/)
