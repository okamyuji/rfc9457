#!/usr/bin/env bash

# check.sh 品質チェックを実行するスクリプト

set -euo pipefail

# 色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 結果カウンター
PASSED=0
FAILED=0

# ログ関数
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

# チェック関数
run_check() {
    local name="$1"
    local cmd="$2"

    log_info "$name を実行中..."
    if eval "$cmd" > /dev/null 2>&1; then
        log_pass "$name"
        return 0
    else
        log_fail "$name"
        eval "$cmd" 2>&1 || true
        return 1
    fi
}

# メイン処理
main() {
    echo "========================================"
    echo "  RFC 9457 Go Project Quality Checks"
    echo "========================================"
    echo ""

    # go fmt
    run_check "go fmt" "test -z \"\$(go fmt ./... 2>&1)\"" || true

    # go vet
    run_check "go vet" "go vet ./..." || true

    # staticcheck
    if command -v staticcheck &> /dev/null; then
        run_check "staticcheck" "staticcheck ./..." || true
    else
        log_info "staticcheck がインストールされていません。スキップします。"
    fi

    # golangci-lint
    if command -v golangci-lint &> /dev/null; then
        run_check "golangci-lint" "golangci-lint run" || true
    else
        log_info "golangci-lint がインストールされていません。スキップします。"
    fi

    # go test
    log_info "go test を実行中..."
    if go test -race -coverprofile=coverage.out ./... > /dev/null 2>&1; then
        log_pass "go test"

        # カバレッジ確認
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        log_info "カバレッジ: ${COVERAGE}%"

        # 80%以上か確認
        if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
            log_pass "カバレッジ 80%以上 (${COVERAGE}%)"
        else
            log_fail "カバレッジ 80%未満 (${COVERAGE}%)"
        fi
    else
        log_fail "go test"
        go test -race ./... 2>&1 || true
    fi

    # ビルド確認
    run_check "go build" "go build -o /dev/null ./cmd/server/" || true

    # サマリー
    echo ""
    echo "========================================"
    echo "  結果サマリー"
    echo "========================================"
    echo -e "  ${GREEN}PASSED${NC}: $PASSED"
    echo -e "  ${RED}FAILED${NC}: $FAILED"
    echo "========================================"

    if [ "$FAILED" -gt 0 ]; then
        exit 1
    fi
    exit 0
}

main "$@"
