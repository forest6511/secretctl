---
title: コントリビュート
description: secretctl への貢献方法。
sidebar_position: 1
---

# secretctl へのコントリビュート

secretctl への貢献に興味を持っていただきありがとうございます！このプロジェクトはオープンソースであり、あらゆる種類の貢献を歓迎します。

## 貢献の方法

### バグ報告

バグを見つけましたか？以下の情報を含めて[issue を開いてください](https://github.com/forest6511/secretctl/issues/new)：
- 問題の明確な説明
- 再現手順
- 期待される動作と実際の動作
- 環境（OS、Go バージョン、secretctl バージョン）

### 機能提案

新機能のアイデアがありますか？以下を記載した issue を開いてください：
- 解決しようとしているユースケース
- 機能の動作方法
- 検討した代替案

### コードの提出

コードを貢献する準備ができましたか？[開発セットアップ](/docs/contributing/development-setup)ガイドを参照してください。

1. リポジトリをフォーク
2. 機能ブランチを作成（`git checkout -b feature/your-feature`）
3. テスト付きで変更を実施
4. `go test ./...` と `golangci-lint run` を実行
5. プルリクエストを提出

### ドキュメントの改善

ドキュメントの改善はいつでも歓迎です：
- タイポの修正や不明瞭なセクションの明確化
- 例やユースケースの追加
- ドキュメントの翻訳

## 行動規範

- 敬意を持ち、建設的であること
- 人ではなく問題に焦点を当てること
- 他の人が学び成長できるよう支援すること

## はじめに

- [開発セットアップ](/docs/contributing/development-setup) - 開発環境のセットアップ
- [アーキテクチャ概要](/docs/architecture) - システム設計の理解
- [GitHub Issues](https://github.com/forest6511/secretctl/issues) - 取り組む issue を見つける

## プルリクエストガイドライン

### 提出前に

- すべてのテストを実行: `go test ./...`
- リンターを実行: `golangci-lint run`
- 必要に応じてドキュメントを更新
- 新機能にはテストを追加

### PR の説明

以下を含めてください：
- 変更内容
- なぜ必要か
- テスト方法
- 関連する issue 番号

### レビュープロセス

1. メンテナーが PR をレビュー
2. フィードバックに対応
3. 承認後、メンテナーがマージ

## ライセンス

貢献することにより、あなたの貢献が [Apache 2.0 ライセンス](https://github.com/forest6511/secretctl/blob/main/LICENSE)の下でライセンスされることに同意したものとみなされます。
