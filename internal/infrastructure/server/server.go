// Package server HTTPサーバーのインフラストラクチャ層
package server

import (
	"context"
	"encoding/json"
	"net/http"

	"golang-rfc9457/internal/adapter/handler"
	problemuc "golang-rfc9457/internal/usecase/problem"
)

// Server HTTPサーバーをラップする構造体
type Server struct {
	httpServer     *http.Server
	problemHandler *handler.ProblemHandler
}

// ServerOption Serverの設定を行う関数型
type ServerOption func(*Server)

// NewServer 新しいServerインスタンスを生成する
func NewServer(h *handler.ProblemHandler, opts ...ServerOption) *Server {
	s := &Server{
		httpServer: &http.Server{
			Addr: ":8080",
		},
		problemHandler: h,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.httpServer.Handler = s.setupRoutes()

	return s
}

// WithAddr サーバーのアドレスを設定する
func WithAddr(addr string) ServerOption {
	return func(s *Server) {
		s.httpServer.Addr = addr
	}
}

// Addr サーバーのアドレスを返す
func (s *Server) Addr() string {
	return s.httpServer.Addr
}

// Handler HTTPハンドラーを返す
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// ListenAndServe サーバーを起動する
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown サーバーを正常にシャットダウンする
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes ルーティングを設定する
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleRoot)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /demo/not-found", s.handleDemoNotFound)
	mux.HandleFunc("GET /demo/validation-error", s.handleDemoValidationError)
	mux.HandleFunc("GET /demo/rate-limit", s.handleDemoRateLimit)
	mux.HandleFunc("GET /demo/internal-error", s.handleDemoInternalError)

	return s.notFoundMiddleware(mux)
}

// notFoundMiddleware 存在しないパスに対してProblem Details形式で404を返す
func (s *Server) notFoundMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(rec, r)

		// ServeMuxが404を返した場合、Problem Details形式で上書きする
		if rec.status == http.StatusNotFound {
			s.problemHandler.HandleNotFound(w, r)
		}
	})
}

// responseRecorder レスポンスの状態を記録し、404時は書き込みを抑制する
type responseRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader ステータスコードを記録し、404の場合は実際の書き込みを遅延する
func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	if code != http.StatusNotFound {
		r.ResponseWriter.WriteHeader(code)
	}
}

// Write 404以外の場合のみデータを書き込む
func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == http.StatusNotFound {
		return len(b), nil // 404の場合は何も書き込まない
	}
	return r.ResponseWriter.Write(b)
}

// handleRoot ルートエンドポイント
func (s *Server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "RFC 9457 Problem Details Demo Server",
		"version": "1.0.0",
	})
}

// handleHealth ヘルスチェックエンドポイント
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleDemoNotFound 404エラーのデモ
func (s *Server) handleDemoNotFound(w http.ResponseWriter, r *http.Request) {
	s.problemHandler.HandleNotFound(w, r)
}

// handleDemoValidationError バリデーションエラーのデモ
func (s *Server) handleDemoValidationError(w http.ResponseWriter, r *http.Request) {
	errors := []problemuc.ValidationErrorDetail{
		{
			Field:   "email",
			Code:    "INVALID_FORMAT",
			Message: "メールアドレスの形式が正しくありません",
			Pointer: "/email",
		},
		{
			Field:   "password",
			Code:    "TOO_SHORT",
			Message: "パスワードは8文字以上である必要があります",
			Pointer: "/password",
		},
	}
	s.problemHandler.HandleValidationError(w, r, errors)
}

// handleDemoRateLimit レート制限エラーのデモ
func (s *Server) handleDemoRateLimit(w http.ResponseWriter, r *http.Request) {
	uc := problemuc.NewUseCase()
	p := uc.CreateRateLimitError(problemuc.RateLimitInput{
		Instance:      r.URL.Path,
		Limit:         100,
		Remaining:     0,
		ResetAt:       "2025-10-29T15:00:00Z",
		RetryAfter:    3600,
		WindowSeconds: 3600,
	})
	s.problemHandler.WriteProblem(w, p)
}

// handleDemoInternalError 内部サーバーエラーのデモ
func (s *Server) handleDemoInternalError(w http.ResponseWriter, r *http.Request) {
	s.problemHandler.HandleInternalServerError(w, r, "デモ用エラー")
}
