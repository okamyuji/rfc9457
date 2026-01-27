package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang-rfc9457/internal/adapter/presenter"
	"golang-rfc9457/internal/domain/problem"
	problemuc "golang-rfc9457/internal/usecase/problem"
)

func TestProblemHandler_WriteProblem_正常にレスポンスを書き込む(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	rec := httptest.NewRecorder()
	p := problem.New(http.StatusNotFound,
		problem.WithDetail("リソースが見つかりません"),
	)

	handler.WriteProblem(rec, p)

	if rec.Code != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusNotFound)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/problem+json")
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	if result["status"] != float64(http.StatusNotFound) {
		t.Errorf("status = %v, want %d", result["status"], http.StatusNotFound)
	}
}

func TestProblemHandler_WriteProblem_Retry_Afterヘッダーを設定(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	rec := httptest.NewRecorder()
	p := problem.New(http.StatusTooManyRequests,
		problem.WithExtension("retry_after", 3600),
	)

	handler.WriteProblem(rec, p)

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter != "3600" {
		t.Errorf("Retry-After = %q, want %q", retryAfter, "3600")
	}
}

func TestProblemHandler_HandleNotFound_404エラーを返す(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()

	handler.HandleNotFound(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	if result["instance"] != "/not-found" {
		t.Errorf("instance = %v, want %q", result["instance"], "/not-found")
	}
}

func TestProblemHandler_HandleMethodNotAllowed_405エラーを返す(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	req := httptest.NewRequest(http.MethodPost, "/resource", nil)
	rec := httptest.NewRecorder()

	handler.HandleMethodNotAllowed(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestProblemHandler_HandleInternalServerError_500エラーを返す(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()

	handler.HandleInternalServerError(rec, req, "テストエラー")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	// detailには内部詳細を含めない（セキュリティ）
	if result["detail"] == "テストエラー" {
		t.Error("Internal error details should not be exposed")
	}
}

func TestProblemHandler_HandleValidationError_バリデーションエラーを返す(t *testing.T) {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	handler := NewProblemHandler(uc, pres)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	rec := httptest.NewRecorder()

	errors := []problemuc.ValidationErrorDetail{
		{Field: "email", Code: "INVALID", Message: "無効なメール"},
	}

	handler.HandleValidationError(rec, req, errors)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Response body is not valid JSON: %v", err)
	}

	validationErrors, ok := result["validation_errors"].([]any)
	if !ok || len(validationErrors) != 1 {
		t.Errorf("validation_errors = %v, want slice with 1 element", result["validation_errors"])
	}
}
