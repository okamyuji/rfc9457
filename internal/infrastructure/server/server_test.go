package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang-rfc9457/internal/adapter/handler"
	"golang-rfc9457/internal/adapter/presenter"
	problemuc "golang-rfc9457/internal/usecase/problem"
)

func createTestHandler() *handler.ProblemHandler {
	uc := problemuc.NewUseCase()
	pres := presenter.NewProblemPresenter()
	return handler.NewProblemHandler(uc, pres)
}

func TestNewServer_デフォルト設定でサーバーを生成(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.Addr() != ":8080" {
		t.Errorf("Addr = %q, want %q", srv.Addr(), ":8080")
	}
}

func TestNewServer_カスタムアドレスでサーバーを生成(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h, WithAddr(":9090"))

	if srv.Addr() != ":9090" {
		t.Errorf("Addr = %q, want %q", srv.Addr(), ":9090")
	}
}

func TestServer_Handler_ルートエンドポイントが動作(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestServer_Handler_存在しないパスは404(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h)

	req := httptest.NewRequest(http.MethodGet, "/not-exist", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

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

func TestServer_Handler_ヘルスチェックエンドポイント(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestServer_Handler_エラーデモエンドポイント(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h)

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/demo/not-found", http.StatusNotFound},
		{"/demo/validation-error", http.StatusBadRequest},
		{"/demo/rate-limit", http.StatusTooManyRequests},
		{"/demo/internal-error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("StatusCode = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/problem+json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/problem+json")
			}
		})
	}
}

func TestServer_Shutdown_正常にシャットダウン(t *testing.T) {
	h := createTestHandler()
	srv := NewServer(h, WithAddr(":0"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}
