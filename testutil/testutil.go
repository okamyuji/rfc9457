// Package testutil テスト用のユーティリティを提供する
package testutil

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// AssertJSONField JSONレスポンスの特定フィールドの値を検証する
func AssertJSONField(t *testing.T, rec *httptest.ResponseRecorder, field string, expected any) {
	t.Helper()

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	actual := result[field]

	// float64とintの比較を考慮
	switch exp := expected.(type) {
	case int:
		if actual != float64(exp) {
			t.Errorf("%s = %v, want %v", field, actual, expected)
		}
	default:
		if actual != expected {
			t.Errorf("%s = %v, want %v", field, actual, expected)
		}
	}
}

// AssertStatusCode HTTPステータスコードを検証する
func AssertStatusCode(t *testing.T, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()

	if rec.Code != expected {
		t.Errorf("StatusCode = %d, want %d", rec.Code, expected)
	}
}

// AssertContentType Content-Typeヘッダーを検証する
func AssertContentType(t *testing.T, rec *httptest.ResponseRecorder, expected string) {
	t.Helper()

	contentType := rec.Header().Get("Content-Type")
	if contentType != expected {
		t.Errorf("Content-Type = %q, want %q", contentType, expected)
	}
}
