// Package problem Problemに関するユースケースを提供する
package problem

import (
	"fmt"
	"net/http"

	"golang-rfc9457/internal/domain/problem"
)

// ProblemRepository Problemの永続化を抽象化するインターフェース
type ProblemRepository interface {
	Save(p *problem.Problem) error
}

// UseCase Problem関連のユースケースを実行する
type UseCase struct {
	repo ProblemRepository
}

// UseCaseOption UseCaseの設定を行う関数型
type UseCaseOption func(*UseCase)

// NewUseCase 新しいUseCaseインスタンスを生成する
func NewUseCase(opts ...UseCaseOption) *UseCase {
	uc := &UseCase{}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}

// WithRepository リポジトリを設定する
func WithRepository(repo ProblemRepository) UseCaseOption {
	return func(uc *UseCase) {
		uc.repo = repo
	}
}

// CreateProblemInput Problem生成のための入力データ
type CreateProblemInput struct {
	Status     int
	Type       string
	Title      string
	Detail     string
	Instance   string
	Extensions map[string]any
}

// CreateProblem 入力データからProblemを生成する
func (uc *UseCase) CreateProblem(input CreateProblemInput) *problem.Problem {
	opts := []problem.Option{}

	if input.Type != "" {
		opts = append(opts, problem.WithType(input.Type))
	}
	if input.Title != "" {
		opts = append(opts, problem.WithTitle(input.Title))
	}
	if input.Detail != "" {
		opts = append(opts, problem.WithDetail(input.Detail))
	}
	if input.Instance != "" {
		opts = append(opts, problem.WithInstance(input.Instance))
	}
	for k, v := range input.Extensions {
		opts = append(opts, problem.WithExtension(k, v))
	}

	return problem.New(input.Status, opts...)
}

// ValidationErrorDetail バリデーションエラーの詳細
type ValidationErrorDetail struct {
	Field       string         `json:"field"`
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Pointer     string         `json:"pointer,omitempty"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// CreateValidationError バリデーションエラー用のProblemを生成する
func (uc *UseCase) CreateValidationError(instance string, errors []ValidationErrorDetail) *problem.Problem {
	return problem.New(http.StatusBadRequest,
		problem.WithType("https://api.example.com/problems/validation-error"),
		problem.WithTitle("バリデーションエラー"),
		problem.WithDetail("1つ以上のフィールドが検証に失敗しました"),
		problem.WithInstance(instance),
		problem.WithExtension("validation_errors", errors),
		problem.WithExtension("total_errors", len(errors)),
	)
}

// RateLimitInput レート制限エラー生成のための入力データ
type RateLimitInput struct {
	Instance      string
	Limit         int
	Remaining     int
	ResetAt       string
	RetryAfter    int
	WindowSeconds int
}

// CreateRateLimitError レート制限エラー用のProblemを生成する
func (uc *UseCase) CreateRateLimitError(input RateLimitInput) *problem.Problem {
	detail := fmt.Sprintf("%d秒あたり%dリクエストの制限を超えました", input.WindowSeconds, input.Limit)

	return problem.New(http.StatusTooManyRequests,
		problem.WithType("https://api.example.com/problems/rate-limit-exceeded"),
		problem.WithTitle("レート制限超過"),
		problem.WithDetail(detail),
		problem.WithInstance(input.Instance),
		problem.WithExtension("limit", input.Limit),
		problem.WithExtension("remaining", input.Remaining),
		problem.WithExtension("reset_at", input.ResetAt),
		problem.WithExtension("retry_after", input.RetryAfter),
		problem.WithExtension("window_seconds", input.WindowSeconds),
	)
}

// SaveProblem Problemをリポジトリに保存する
func (uc *UseCase) SaveProblem(p *problem.Problem) error {
	if uc.repo == nil {
		return nil
	}
	return uc.repo.Save(p)
}
