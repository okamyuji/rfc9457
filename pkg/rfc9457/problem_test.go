package rfc9457

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_デフォルト値が設定される(t *testing.T) {
	p := New(http.StatusNotFound)

	if p.Type != "about:blank" {
		t.Errorf("Type = %q, want %q", p.Type, "about:blank")
	}
	if p.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", p.Status, http.StatusNotFound)
	}
	if p.Title != "Not Found" {
		t.Errorf("Title = %q, want %q", p.Title, "Not Found")
	}
}

func TestNew_カスタム値を設定(t *testing.T) {
	p := New(http.StatusForbidden,
		WithType("https://api.example.com/problems/out-of-credit"),
		WithTitle("クレジットが不足しています"),
		WithDetail("現在の残高は30です"),
		WithInstance("/account/12345"),
		WithExtension("balance", 30),
	)

	if p.Type != "https://api.example.com/problems/out-of-credit" {
		t.Errorf("Type = %q, want expected value", p.Type)
	}
	if p.Title != "クレジットが不足しています" {
		t.Errorf("Title = %q, want expected value", p.Title)
	}
	if p.Extensions["balance"] != 30 {
		t.Errorf("Extensions[balance] = %v, want 30", p.Extensions["balance"])
	}
}

func TestProblem_Error_インターフェース実装(t *testing.T) {
	p := New(http.StatusNotFound, WithDetail("リソースが見つかりません"))

	var err error = p
	if err.Error() != "Not Found: リソースが見つかりません" {
		t.Errorf("Error() = %q, want expected value", err.Error())
	}
}

func TestProblem_MarshalJSON_拡張フィールドがトップレベル(t *testing.T) {
	p := New(http.StatusForbidden, WithExtension("balance", 30))

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["balance"] != float64(30) {
		t.Errorf("balance = %v, want 30", result["balance"])
	}
}

func TestWriteProblem_レスポンスを書き込む(t *testing.T) {
	rec := httptest.NewRecorder()
	p := New(http.StatusNotFound, WithDetail("リソースが見つかりません"))

	WriteProblem(rec, p)

	if rec.Code != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusNotFound)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != ContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, ContentType)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	if result["status"] != float64(http.StatusNotFound) {
		t.Errorf("status = %v, want %d", result["status"], http.StatusNotFound)
	}
}

func TestWriteProblem_Retry_Afterヘッダーを設定(t *testing.T) {
	rec := httptest.NewRecorder()
	p := New(http.StatusTooManyRequests, WithExtension("retry_after", 3600))

	WriteProblem(rec, p)

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter != "3600" {
		t.Errorf("Retry-After = %q, want %q", retryAfter, "3600")
	}
}

func TestContentType_正しい値(t *testing.T) {
	if ContentType != "application/problem+json" {
		t.Errorf("ContentType = %q, want %q", ContentType, "application/problem+json")
	}
}

func TestNewValidationError_バリデーションエラーを生成(t *testing.T) {
	errors := []ValidationError{
		{Field: "email", Code: "INVALID", Message: "無効なメール", Pointer: "/email"},
	}

	p := NewValidationError("/api/users", errors)

	if p.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", p.Status, http.StatusBadRequest)
	}

	validationErrors, ok := p.Extensions["validation_errors"].([]ValidationError)
	if !ok {
		t.Fatalf("Extensions[validation_errors] is not []ValidationError")
	}
	if len(validationErrors) != 1 {
		t.Errorf("validation_errors length = %d, want 1", len(validationErrors))
	}
}

func TestNewRateLimitError_レート制限エラーを生成(t *testing.T) {
	p := NewRateLimitError("/api/search", 100, 0, "2025-10-29T15:00:00Z", 3600)

	if p.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", p.Status, http.StatusTooManyRequests)
	}
	if p.Extensions["limit"] != 100 {
		t.Errorf("Extensions[limit] = %v, want 100", p.Extensions["limit"])
	}
	if p.Extensions["retry_after"] != 3600 {
		t.Errorf("Extensions[retry_after] = %v, want 3600", p.Extensions["retry_after"])
	}
}
