#!/usr/bin/env bash

# e2e_test.sh RFC 9457 Problem Details サーバーのE2Eテスト

set -euo pipefail

# 色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 設定
export PORT="${PORT:-18080}"
BASE_URL="http://localhost:${PORT}"
SERVER_PID=""
PASSED=0
FAILED=0
TOTAL=0

# ログ関数
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    PASSED=$((PASSED + 1))
    TOTAL=$((TOTAL + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED=$((FAILED + 1))
    TOTAL=$((TOTAL + 1))
}

log_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# クリーンアップ関数
cleanup() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        log_info "サーバーを停止中 (PID: $SERVER_PID)..."
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

trap cleanup EXIT

# サーバー起動
start_server() {
    log_info "サーバーをビルド中..."
    go build -o /tmp/rfc9457-server ./cmd/server/

    log_info "サーバーを起動中 (ポート: $PORT)..."
    /tmp/rfc9457-server &
    SERVER_PID=$!

    # サーバー起動待機
    for i in {1..30}; do
        if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
            log_info "サーバーが起動しました (PID: $SERVER_PID)"
            return 0
        fi
        sleep 0.1
    done

    log_fail "サーバーの起動に失敗しました"
    exit 1
}

# HTTPリクエストを実行してJSONレスポンスを検証
assert_json_field() {
    local json="$1"
    local field="$2"
    local expected="$3"
    local actual

    actual=$(echo "$json" | jq -r ".$field" 2>/dev/null || echo "")

    if [ "$actual" = "$expected" ]; then
        return 0
    else
        echo "  Expected $field: $expected, Got: $actual"
        return 1
    fi
}

# HTTPステータスコードを検証
assert_status() {
    local actual="$1"
    local expected="$2"

    if [ "$actual" = "$expected" ]; then
        return 0
    else
        echo "  Expected status: $expected, Got: $actual"
        return 1
    fi
}

# Content-Typeを検証
assert_content_type() {
    local headers="$1"
    local expected="$2"

    if echo "$headers" | grep -qi "content-type: $expected"; then
        return 0
    else
        echo "  Expected Content-Type: $expected"
        echo "  Got headers: $headers"
        return 1
    fi
}

# ヘッダーの存在を検証
assert_header_exists() {
    local headers="$1"
    local header_name="$2"

    if echo "$headers" | grep -qi "^$header_name:"; then
        return 0
    else
        echo "  Expected header: $header_name"
        return 1
    fi
}

# ヘッダー値を検証
assert_header_value() {
    local headers="$1"
    local header_name="$2"
    local expected_value="$3"
    local actual_value

    actual_value=$(echo "$headers" | grep -i "^$header_name:" | cut -d: -f2 | tr -d ' \r\n')

    if [ "$actual_value" = "$expected_value" ]; then
        return 0
    else
        echo "  Expected $header_name: $expected_value, Got: $actual_value"
        return 1
    fi
}

# テスト関数: 基本テスト
test_case() {
    local name="$1"
    local test_func="$2"
    local result

    # set -e を一時的に無効化してテストを実行
    set +e
    $test_func
    result=$?
    set -e

    if [ $result -eq 0 ]; then
        log_pass "$name"
    else
        log_fail "$name"
    fi
}

#####################################
# 正常系テスト
#####################################

test_root_endpoint() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/")
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    assert_status "$status" "200" && \
    assert_json_field "$body" "message" "RFC 9457 Problem Details Demo Server"
}

test_health_endpoint() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    assert_status "$status" "200" && \
    assert_json_field "$body" "status" "ok"
}

test_demo_not_found() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - "$BASE_URL/demo/not-found")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | sed '$d')

    assert_status "$status" "404" && \
    assert_content_type "$headers" "application/problem+json" && \
    assert_json_field "$body" "status" "404" && \
    assert_json_field "$body" "type" "about:blank" && \
    assert_json_field "$body" "title" "Not Found"
}

test_demo_validation_error() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - "$BASE_URL/demo/validation-error")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | sed '$d')

    assert_status "$status" "400" && \
    assert_content_type "$headers" "application/problem+json" && \
    assert_json_field "$body" "status" "400" && \
    assert_json_field "$body" "title" "バリデーションエラー" && \
    [ "$(echo "$body" | jq '.validation_errors | length')" = "2" ]
}

test_demo_rate_limit() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - "$BASE_URL/demo/rate-limit")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | sed '$d')

    assert_status "$status" "429" && \
    assert_content_type "$headers" "application/problem+json" && \
    assert_header_exists "$headers" "Retry-After" && \
    assert_header_value "$headers" "Retry-After" "3600" && \
    assert_json_field "$body" "status" "429" && \
    assert_json_field "$body" "limit" "100" && \
    assert_json_field "$body" "retry_after" "3600"
}

test_demo_internal_error() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - "$BASE_URL/demo/internal-error")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | sed '$d')

    assert_status "$status" "500" && \
    assert_content_type "$headers" "application/problem+json" && \
    assert_json_field "$body" "status" "500" && \
    assert_json_field "$body" "title" "Internal Server Error"
}

#####################################
# 異常系テスト
#####################################

test_nonexistent_path() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - "$BASE_URL/nonexistent/path/here")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | sed '$d')

    assert_status "$status" "404" && \
    assert_content_type "$headers" "application/problem+json" && \
    assert_json_field "$body" "status" "404" && \
    assert_json_field "$body" "instance" "/nonexistent/path/here"
}

test_deep_nonexistent_path() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/a/b/c/d/e/f/g/h/i/j")
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    assert_status "$status" "404" && \
    assert_json_field "$body" "status" "404"
}

test_post_to_get_endpoint() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/health")
    status=$(echo "$response" | tail -n1)

    # Go 1.22+ ServeMux は POST を GET only エンドポイントに対して 405 ではなく 404 を返すことがある
    # 実装に依存するので 404 または 405 を許容
    [ "$status" = "404" ] || [ "$status" = "405" ]
}

test_invalid_method() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/")
    status=$(echo "$response" | tail -n1)

    # Go 1.22+ ServeMux の挙動に合わせる
    [ "$status" = "404" ] || [ "$status" = "405" ]
}

#####################################
# 境界値テスト
#####################################

test_root_path_only() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/")
    status=$(echo "$response" | tail -n1)

    assert_status "$status" "200"
}

test_path_with_trailing_slash() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health/")
    status=$(echo "$response" | tail -n1)

    # trailing slash は別パスとして扱われるため 404
    assert_status "$status" "404"
}

test_empty_query_string() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health?")
    status=$(echo "$response" | tail -n1)

    assert_status "$status" "200"
}

test_with_query_parameters() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health?foo=bar&baz=qux")
    status=$(echo "$response" | tail -n1)

    assert_status "$status" "200"
}

#####################################
# エッジケーステスト
#####################################

test_special_characters_in_path() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/path%20with%20spaces")
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    assert_status "$status" "404" && \
    assert_json_field "$body" "status" "404"
}

test_unicode_in_path() {
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/日本語パス")
    status=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    assert_status "$status" "404" && \
    assert_json_field "$body" "status" "404"
}

test_very_long_path() {
    local long_path
    long_path=$(printf 'a%.0s' {1..500})
    local response
    local status
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/$long_path")
    status=$(echo "$response" | tail -n1)

    # 404 または 414 (URI Too Long) を許容
    [ "$status" = "404" ] || [ "$status" = "414" ]
}

test_accept_header_json() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - -H "Accept: application/json" "$BASE_URL/demo/not-found")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)

    # Problem Details は Accept に関係なく application/problem+json を返す
    assert_status "$status" "404" && \
    assert_content_type "$headers" "application/problem+json"
}

test_accept_header_xml() {
    local response
    local status
    local headers
    response=$(curl -s -w "\n%{http_code}" -D - -H "Accept: application/xml" "$BASE_URL/demo/not-found")
    headers=$(echo "$response" | sed '/^\r$/q')
    status=$(echo "$response" | tail -n1)

    # このサーバーは JSON のみサポート
    assert_status "$status" "404" && \
    assert_content_type "$headers" "application/problem+json"
}

test_head_request() {
    local status
    local headers
    headers=$(curl -s -I "$BASE_URL/health")
    status=$(echo "$headers" | head -n1 | awk '{print $2}')

    assert_status "$status" "200"
}

test_concurrent_requests() {
    local pids=()
    local results=()

    # 10個の並行リクエスト
    for i in {1..10}; do
        curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" &
        pids+=($!)
    done

    # 全リクエストを待機
    for pid in "${pids[@]}"; do
        wait "$pid"
        results+=($?)
    done

    # すべて成功したか確認
    for result in "${results[@]}"; do
        if [ "$result" != "0" ]; then
            return 1
        fi
    done
    return 0
}

test_validation_error_fields() {
    local response
    local body
    response=$(curl -s "$BASE_URL/demo/validation-error")
    body="$response"

    # 各フィールドが存在することを確認
    [ "$(echo "$body" | jq -r '.validation_errors[0].field')" = "email" ] && \
    [ "$(echo "$body" | jq -r '.validation_errors[0].code')" = "INVALID_FORMAT" ] && \
    [ "$(echo "$body" | jq -r '.validation_errors[1].field')" = "password" ] && \
    [ "$(echo "$body" | jq -r '.validation_errors[1].code')" = "TOO_SHORT" ] && \
    [ "$(echo "$body" | jq -r '.total_errors')" = "2" ]
}

test_rate_limit_all_fields() {
    local response
    local body
    response=$(curl -s "$BASE_URL/demo/rate-limit")
    body="$response"

    # すべての拡張フィールドが存在することを確認
    [ "$(echo "$body" | jq -r '.limit')" = "100" ] && \
    [ "$(echo "$body" | jq -r '.remaining')" = "0" ] && \
    [ "$(echo "$body" | jq -r '.reset_at')" = "2025-10-29T15:00:00Z" ] && \
    [ "$(echo "$body" | jq -r '.retry_after')" = "3600" ] && \
    [ "$(echo "$body" | jq -r '.window_seconds')" = "3600" ]
}

test_problem_type_uri() {
    local response
    local body
    response=$(curl -s "$BASE_URL/demo/validation-error")
    body="$response"

    # type が適切な URI 形式であることを確認
    [ "$(echo "$body" | jq -r '.type')" = "https://api.example.com/problems/validation-error" ]
}

#####################################
# メイン処理
#####################################

main() {
    echo "========================================"
    echo "  RFC 9457 E2E Tests"
    echo "========================================"

    # 依存関係チェック
    if ! command -v jq &> /dev/null; then
        echo "エラー: jq がインストールされていません"
        exit 1
    fi

    if ! command -v curl &> /dev/null; then
        echo "エラー: curl がインストールされていません"
        exit 1
    fi

    # サーバー起動
    start_server

    # 正常系テスト
    log_section "正常系テスト"
    test_case "ルートエンドポイント" test_root_endpoint
    test_case "ヘルスチェックエンドポイント" test_health_endpoint
    test_case "デモ: Not Found" test_demo_not_found
    test_case "デモ: バリデーションエラー" test_demo_validation_error
    test_case "デモ: レート制限エラー" test_demo_rate_limit
    test_case "デモ: 内部サーバーエラー" test_demo_internal_error

    # 異常系テスト
    log_section "異常系テスト"
    test_case "存在しないパス" test_nonexistent_path
    test_case "深い存在しないパス" test_deep_nonexistent_path
    test_case "GETエンドポイントへのPOST" test_post_to_get_endpoint
    test_case "無効なHTTPメソッド" test_invalid_method

    # 境界値テスト
    log_section "境界値テスト"
    test_case "ルートパスのみ" test_root_path_only
    test_case "末尾スラッシュ付きパス" test_path_with_trailing_slash
    test_case "空のクエリ文字列" test_empty_query_string
    test_case "クエリパラメータ付き" test_with_query_parameters

    # エッジケーステスト
    log_section "エッジケーステスト"
    test_case "パス内の特殊文字" test_special_characters_in_path
    test_case "パス内のUnicode" test_unicode_in_path
    test_case "非常に長いパス" test_very_long_path
    test_case "Accept: application/json ヘッダー" test_accept_header_json
    test_case "Accept: application/xml ヘッダー" test_accept_header_xml
    test_case "HEADリクエスト" test_head_request
    test_case "並行リクエスト" test_concurrent_requests

    # データ検証テスト
    log_section "データ検証テスト"
    test_case "バリデーションエラーの全フィールド" test_validation_error_fields
    test_case "レート制限エラーの全フィールド" test_rate_limit_all_fields
    test_case "Problem Type URI形式" test_problem_type_uri

    # サマリー
    echo ""
    echo "========================================"
    echo "  結果サマリー"
    echo "========================================"
    echo -e "  ${GREEN}PASSED${NC}: $PASSED / $TOTAL"
    echo -e "  ${RED}FAILED${NC}: $FAILED / $TOTAL"
    echo "========================================"

    if [ "$FAILED" -gt 0 ]; then
        exit 1
    fi
    exit 0
}

# スクリプトのディレクトリに移動
cd "$(dirname "$0")/.."

main "$@"
