#!/bin/bash
# secretctl Demo GIF 作成スクリプト
# 使用方法: ./create-demo-gif.sh

set -e

DEMO_DIR="/tmp/secretctl-demo-recording"
VAULT_DIR="/tmp/secretctl-demo"
PASSWORD="demo123456"
SECRETCTL="${DEMO_DIR}/secretctl"
OUTPUT_GIF="${DEMO_DIR}/demo.gif"
OUTPUT_CAST="${DEMO_DIR}/demo.cast"

echo "=== secretctl Demo GIF 作成 ==="

# 1. 必要なツールの確認
echo "[1/5] ツール確認..."
for cmd in asciinema agg expect; do
    if ! command -v $cmd &> /dev/null; then
        echo "エラー: $cmd がインストールされていません"
        echo "  brew install $cmd"
        exit 1
    fi
done
echo "  ✓ asciinema, agg, expect"

# 2. secretctl ビルド（必要な場合）
if [ ! -f "$SECRETCTL" ]; then
    echo "[2/5] secretctl ビルド..."
    REPO_DIR="/Users/hisaoyoshitome/Workspace/secretctl"
    if [ -f "$REPO_DIR/cmd/secretctl/main.go" ]; then
        (cd "$REPO_DIR" && go build -o "$SECRETCTL" ./cmd/secretctl)
        echo "  ✓ ビルド完了"
    else
        echo "エラー: secretctl ソースが見つかりません"
        exit 1
    fi
else
    echo "[2/5] secretctl 既存 ✓"
fi

# 3. Vault 初期化
echo "[3/5] Vault 初期化..."
rm -rf "$VAULT_DIR"
rm -rf ~/.secretctl
mkdir -p "$VAULT_DIR"

SECRETCTL_VAULT_DIR="$VAULT_DIR" expect << INIT_EOF
spawn $SECRETCTL init
expect "Enter master password:"
send "$PASSWORD\r"
expect "Confirm master password:"
send "$PASSWORD\r"
expect eof
INIT_EOF

echo "  ✓ Vault 初期化完了: $VAULT_DIR"

# 4. asciinema 録画
echo "[4/5] デモ録画..."
rm -f "$OUTPUT_CAST"

# expect スクリプトを作成
cat > "${DEMO_DIR}/run-demo.exp" << 'EXPECT_EOF'
#!/usr/bin/expect -f

set timeout 60
set password "demo123456"
set secretctl "/tmp/secretctl-demo-recording/secretctl"
set vault_dir "/tmp/secretctl-demo"
set env(SECRETCTL_VAULT_DIR) "/tmp/secretctl-demo"

spawn asciinema rec /tmp/secretctl-demo-recording/demo.cast --overwrite

# Wait for shell to start
sleep 1.5

# Setup clean prompt
send "export SECRETCTL_VAULT_DIR=/tmp/secretctl-demo\r"
sleep 0.3
send "export PS1='$ '\r"
sleep 0.3
send "clear\r"
sleep 0.5

# Title
send "echo '# secretctl - The simplest AI-ready secrets manager'\r"
sleep 1.5

# Set secret using --field (no pipe needed)
send "$secretctl set OPENAI_API_KEY --field value=sk-proj-abc123xyz\r"
expect "Enter master password:"
send "$password\r"
expect "saved"
sleep 1.2

# List
send "$secretctl list\r"
expect "Enter master password:"
send "$password\r"
expect "OPENAI_API_KEY"
sleep 1.2

# Get
send "$secretctl get OPENAI_API_KEY\r"
expect "Enter master password:"
send "$password\r"
expect "sk-proj-abc123xyz"
sleep 1.2

# Run with sanitization demo
send "echo '# Secrets injected as env vars - output is sanitized:'\r"
sleep 0.8
send "$secretctl run -k OPENAI_API_KEY -- sh -c 'echo My key is \$OPENAI_API_KEY'\r"
expect "Enter master password:"
send "$password\r"
expect -re "(REDACTED|My key)"
sleep 1.5

# Delete
send "$secretctl delete OPENAI_API_KEY\r"
expect "Enter master password:"
send "$password\r"
expect -re "(deleted|removed|Secret)"
sleep 1.2

# Ending
send "echo '# Local-first. AI-safe. Just works.'\r"
sleep 0.5
send "echo 'https://github.com/forest6511/secretctl'\r"
sleep 2

send "exit\r"
expect eof
EXPECT_EOF

chmod +x "${DEMO_DIR}/run-demo.exp"

# 環境変数を設定して expect 実行
SECRETCTL_VAULT_DIR="$VAULT_DIR" "${DEMO_DIR}/run-demo.exp"

echo "  ✓ 録画完了"

# 5. GIF 変換
echo "[5/5] GIF 変換..."
agg "$OUTPUT_CAST" "$OUTPUT_GIF" \
    --font-size 18 \
    --theme monokai \
    --cols 80 \
    --rows 24

echo ""
echo "=== 完了 ==="
echo "GIF: $OUTPUT_GIF"
echo "サイズ: $(du -h "$OUTPUT_GIF" | cut -f1)"
echo ""
echo "確認: open $OUTPUT_GIF"
