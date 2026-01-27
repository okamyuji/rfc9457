// Package problem RFC 9457 Problem Detailsのドメインモデルを提供する
package problem

import (
	"encoding/json"
	"net/http"
)

// Problem RFC 9457に準拠したProblem Detailsエンティティ
type Problem struct {
	Type       string                 `json:"type"`
	Status     int                    `json:"status"`
	Title      string                 `json:"title"`
	Detail     string                 `json:"detail,omitempty"`
	Instance   string                 `json:"instance,omitempty"`
	Extensions map[string]interface{} `json:"-"`
}

// Option Problemの設定を行う関数型
type Option func(*Problem)

// New 指定されたステータスコードで新しいProblemを生成する
func New(status int, opts ...Option) *Problem {
	p := &Problem{
		Type:       "about:blank",
		Status:     status,
		Title:      GetStatusTitle(status),
		Extensions: make(map[string]interface{}),
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
func WithExtension(key string, value interface{}) Option {
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
	m := make(map[string]interface{})

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

// GetStatusTitle HTTPステータスコードに対応するタイトルを返す
func GetStatusTitle(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "Unknown Status"
	}
	return text
}
