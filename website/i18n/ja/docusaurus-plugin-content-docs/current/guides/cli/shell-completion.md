---
title: シェル補完
description: secretctl コマンドのシェル補完を有効にする。
sidebar_position: 5
---

# シェル補完

secretctl は bash、zsh、fish、PowerShell のシェル補完をサポートしています。

## クイックインストール

### Bash

```bash
# ~/.bashrc に追加
echo 'source <(secretctl completion bash)' >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
# ~/.zshrc に追加
echo 'source <(secretctl completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
secretctl completion fish > ~/.config/fish/completions/secretctl.fish
```

### PowerShell

```powershell
# PowerShell プロファイルに追加
Add-Content $PROFILE 'secretctl completion powershell | Out-String | Invoke-Expression'
```

## 補完される内容

シェル補完で以下が補完されます：

- **コマンド**: `secretctl <TAB>` で利用可能なコマンドを表示
- **サブコマンド**: `secretctl completion <TAB>` でシェルオプションを表示
- **フラグ**: `secretctl get --<TAB>` で利用可能なフラグを表示
- **シークレットキー**: 動的補完を有効にした場合（下記参照）

## 動的補完

デフォルトでは、セキュリティ上の理由からシークレットキーは補完されません。動的なシークレットキー補完を有効にするには：

```bash
export SECRETCTL_COMPLETION_ENABLED=1
```

動的補完を有効にすると：

```bash
secretctl get <TAB>
# 利用可能なシークレットキーを表示

secretctl delete <TAB>
# 利用可能なシークレットキーを表示
```

:::caution セキュリティに関する注意
動的補完は Vault がアンロックされている必要があります。Vault がロックされている場合、アンロックするまでキー補完は利用できません。
:::

### 永続的に有効にする

シェル設定に追加：

```bash
# Bash (~/.bashrc) または Zsh (~/.zshrc)
export SECRETCTL_COMPLETION_ENABLED=1
source <(secretctl completion bash)  # または zsh
```

```fish
# Fish (~/.config/fish/config.fish)
set -gx SECRETCTL_COMPLETION_ENABLED 1
```

## インストールスクリプト

secretctl には補完をインストールするヘルパースクリプトが含まれています：

### Bash

```bash
# インストールスクリプトは ~/.bash_completion.d/ に補完を追加します
./scripts/install-completion-bash.sh
```

### Zsh

```bash
# インストールスクリプトは ~/.zsh/completions/ に補完を追加します
./scripts/install-completion-zsh.zsh
```

### Fish

```bash
# インストールスクリプトは ~/.config/fish/completions/ に補完を追加します
./scripts/install-completion-fish.fish
```

## インストールの確認

インストール後、新しいターミナルを開いてテスト：

```bash
# secretctl と入力して Tab を押す
secretctl <TAB>
# 表示される: audit backup completion delete export generate get init list lock run set unlock

# サブコマンドを入力して Tab を押すとフラグが表示される
secretctl get --<TAB>
# --json, --silent などのフラグが表示される
```

## トラブルシューティング

### 補完が動作しない

1. **新しいターミナルを開く** - 補完スクリプトはシェル起動時に読み込まれます
2. **source コマンドを確認** - `source <(secretctl completion ...)` がシェル設定に含まれていることを確認
3. **secretctl が PATH にあることを確認** - `which secretctl` を実行して確認

### 動的補完が動作しない

1. **環境変数を確認** - `echo $SECRETCTL_COMPLETION_ENABLED` が `1` を表示するか確認
2. **Vault をアンロック** - 動的補完にはアンロックされた Vault が必要
3. **Vault の状態を確認** - `secretctl list` を実行してアクセスを確認

### Zsh compinit の問題

"command not found: compdef" エラーが出る場合、補完 source の前に以下を追加：

```bash
autoload -Uz compinit && compinit
source <(secretctl completion zsh)
```

## 上級者向け: カスタム補完

補完システムは Cobra の組み込み補完を使用しています。以下の方法で拡張できます：

1. secretctl をフォーク
2. `cmd/secretctl/` のコマンドに `ValidArgsFunction` を追加
3. `go build` で再ビルド

## 次のステップ

- [CLI コマンドリファレンス](/docs/reference/cli-commands) - 完全なコマンドドキュメント
- [コマンドの実行](/docs/guides/cli/running-commands) - シークレットをコマンドで使用
