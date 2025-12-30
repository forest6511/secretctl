#!/bin/bash
# CLI Smoke Test Script for secretctl
# Run this on each OS after downloading the release binary

set -euo pipefail

BINARY="${1:-./secretctl}"
TEST_VAULT_DIR="${TEST_VAULT_DIR:-/tmp/secretctl-smoke-test-$$}"
TEST_PASSWORD="smoke-test-password-123"

echo "=== secretctl CLI Smoke Test ==="
echo "Binary: $BINARY"
echo "Test vault: $TEST_VAULT_DIR"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "=== Cleanup ==="
    rm -rf "$TEST_VAULT_DIR"
    echo "Removed test vault directory"
}
trap cleanup EXIT

# Export environment
export SECRETCTL_VAULT_DIR="$TEST_VAULT_DIR"
export SECRETCTL_PASSWORD="$TEST_PASSWORD"

# Test 1: Version
echo "1. Testing version..."
VERSION=$($BINARY version 2>&1 || true)
echo "   Version: $VERSION"
if [[ -z "$VERSION" ]]; then
    echo "   FAIL: No version output"
    exit 1
fi
echo "   PASS"

# Test 2: Init
echo ""
echo "2. Testing init..."
$BINARY init
if [[ ! -d "$TEST_VAULT_DIR" ]]; then
    echo "   FAIL: Vault directory not created"
    exit 1
fi
echo "   PASS"

# Test 3: Set secret
echo ""
echo "3. Testing set..."
$BINARY set test/smoke-key "smoke-value-123"
echo "   PASS"

# Test 4: Get secret
echo ""
echo "4. Testing get..."
VALUE=$($BINARY get test/smoke-key)
if [[ "$VALUE" != "smoke-value-123" ]]; then
    echo "   FAIL: Expected 'smoke-value-123', got '$VALUE'"
    exit 1
fi
echo "   PASS"

# Test 5: List secrets
echo ""
echo "5. Testing list..."
LIST=$($BINARY list)
if ! echo "$LIST" | grep -q "test/smoke-key"; then
    echo "   FAIL: Secret not found in list"
    exit 1
fi
echo "   PASS"

# Test 6: Run command (AI-Safe Access - env injection)
echo ""
echo "6. Testing run command (AI-Safe Access)..."
$BINARY set test/env-key "env-secret-456"
if [[ "$(uname)" == "MINGW"* ]] || [[ "$(uname)" == "MSYS"* ]]; then
    # Windows
    RUN_OUTPUT=$($BINARY run -k "test/env-key" -- cmd /c "echo %TEST_ENV_KEY%")
else
    # Unix
    RUN_OUTPUT=$($BINARY run -k "test/env-key" -- printenv TEST_ENV_KEY)
fi
if [[ "$RUN_OUTPUT" != "env-secret-456" ]]; then
    echo "   FAIL: Expected 'env-secret-456', got '$RUN_OUTPUT'"
    exit 1
fi
echo "   PASS"

# Test 7: Delete secret
echo ""
echo "7. Testing delete..."
$BINARY delete test/smoke-key
$BINARY delete test/env-key
LIST_AFTER=$($BINARY list 2>&1 || true)
if echo "$LIST_AFTER" | grep -q "test/smoke-key"; then
    echo "   FAIL: Secret still exists after delete"
    exit 1
fi
echo "   PASS"

# Test 8: Generate password
echo ""
echo "8. Testing generate..."
GEN_OUTPUT=$($BINARY generate --length 16)
if [[ ${#GEN_OUTPUT} -lt 16 ]]; then
    echo "   FAIL: Generated password too short"
    exit 1
fi
echo "   PASS"

echo ""
echo "=== All CLI Smoke Tests PASSED ==="
