// Package rfc9457 RFC 9457 Problem Details for HTTP APIsの公開APIを提供する
package rfc9457

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ContentType RFC 9457で定義されたProblem DetailsのContent-Type
const ContentType = "application/problem+json"

// Problem RFC 9457に準拠したProblem Detailsエンティティ
type Problem struct {
	Type       string         `json:"type"`
	Status     int            `json:"status"`
	Title      string         `json:"title"`
	Detail     string         `json:"detail,omitempty"`
	Instance   string         `json:"instance,omitempty"`
	Extensions map[string]any `json:"-"`
}

// Option Problemの設定を行う関数型
type Option func(*Problem)

// New 指定されたステータスコードで新しいProblemを生成する
func New(status int, opts ...Option) *Problem {
	p := &Problem{
		Type:       "about:blank",
		Status:     status,
		Title:      http.StatusText(status),
		Extensions: make(map[string]any),
	}

	if p.Title == "" {
		p.Title = "Unknown Status"
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithType Problem TypeのURIを設定する
func WithType(t string) Option {
	return func(p *Problem) {
		p.Type = t
	}
}

// WithTitle 問題タイプの短い要約を設定する
func WithTitle(title string) Option {
	return func(p *Problem) {
		p.Title = title
	}
}

// WithDetail この発生に固有の説明を設定する
func WithDetail(detail string) Option {
	return func(p *Problem) {
		p.Detail = detail
	}
}

// WithInstance 問題発生を識別するURIを設定する
func WithInstance(instance string) Option {
	return func(p *Problem) {
		p.Instance = instance
	}
}

// WithExtension 拡張フィールドを追加する
func WithExtension(key string, value any) Option {
	return func(p *Problem) {
		p.Extensions[key] = value
	}
}

// Error errorインターフェースを実装する
func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Title + ": " + p.Detail
	}
	return p.Title
}

// MarshalJSON 拡張フィールドをトップレベルに展開してJSONにシリアライズする
func (p *Problem) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	m["type"] = p.Type
	m["status"] = p.Status
	m["title"] = p.Title

	if p.Detail != "" {
		m["detail"] = p.Detail
	}
	if p.Instance != "" {
		m["instance"] = p.Instance
	}

	for k, v := range p.Extensions {
		m[k] = v
	}

	return json.Marshal(m)
}

// WriteProblem ProblemをHTTPレスポンスとして書き込む
func WriteProblem(w http.ResponseWriter, p *Problem) {
	// Retry-Afterヘッダーの処理
	if retryAfter, ok := p.Extensions["retry_after"]; ok {
		switch v := retryAfter.(type) {
		case int:
			w.Header().Set("Retry-After", strconv.Itoa(v))
		case int64:
			w.Header().Set("Retry-After", strconv.FormatInt(v, 10))
		case float64:
			w.Header().Set("Retry-After", strconv.FormatInt(int64(v), 10))
		case string:
			w.Header().Set("Retry-After", v)
		}
	}

	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)

	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
}

// ValidationError バリデーションエラーの詳細
type ValidationError struct {
	Field       string         `json:"field"`
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Pointer     string         `json:"pointer,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// NewValidationError バリデーションエラー用のProblemを生成する
func NewValidationError(instance string, errors []ValidationError) *Problem {
	return New(http.StatusBadRequest,
		WithType("https://api.example.com/problems/validation-error"),
		WithTitle("バリデーションエラー"),
		WithDetail("1つ以上のフィールドが検証に失敗しました"),
		WithInstance(instance),
		WithExtension("validation_errors", errors),
		WithExtension("total_errors", len(errors)),
	)
}

// NewRateLimitError レート制限エラー用のProblemを生成する
func NewRateLimitError(instance string, limit, remaining int, resetAt string, retryAfter int) *Problem {
	return New(http.StatusTooManyRequests,
		WithType("https://api.example.com/problems/rate-limit-exceeded"),
		WithTitle("レート制限超過"),
		WithDetail("レート制限を超過しました"),
		WithInstance(instance),
		WithExtension("limit", limit),
		WithExtension("remaining", remaining),
		WithExtension("reset_at", resetAt),
		WithExtension("retry_after", retryAfter),
	)
}
