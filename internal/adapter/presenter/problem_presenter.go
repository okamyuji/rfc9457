// Package presenter HTTPレスポンスの整形を担当するプレゼンター層
package presenter

import (
	"encoding/json"
	"strconv"

	"golang-rfc9457/internal/domain/problem"
)

// ProblemPresenterInterface Problemのプレゼンテーションを抽象化するインターフェース
type ProblemPresenterInterface interface {
	ToJSON(p *problem.Problem) ([]byte, error)
	GetContentType() string
	GetHeaders(p *problem.Problem) map[string]string
}

// ProblemPresenter Problemをレスポンス形式に変換する
type ProblemPresenter struct{}

// NewProblemPresenter 新しいProblemPresenterインスタンスを生成する
func NewProblemPresenter() *ProblemPresenter {
	return &ProblemPresenter{}
}

// ToJSON ProblemをJSON形式にシリアライズする
func (pr *ProblemPresenter) ToJSON(p *problem.Problem) ([]byte, error) {
	return json.Marshal(p)
}

// GetContentType RFC 9457で定義されたContent-Typeを返す
func (pr *ProblemPresenter) GetContentType() string {
	return "application/problem+json"
}

// GetHeaders Problemに基づいて追加のHTTPヘッダーを生成する
func (pr *ProblemPresenter) GetHeaders(p *problem.Problem) map[string]string {
	headers := make(map[string]string)

	if retryAfter, ok := p.Extensions["retry_after"]; ok {
		switch v := retryAfter.(type) {
		case int:
			headers["Retry-After"] = strconv.Itoa(v)
		case int64:
			headers["Retry-After"] = strconv.FormatInt(v, 10)
		case float64:
			headers["Retry-After"] = strconv.FormatInt(int64(v), 10)
		case string:
			headers["Retry-After"] = v
		}
	}

	return headers
}
