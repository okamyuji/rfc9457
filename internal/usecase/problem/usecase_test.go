package problem

import (
	"net/http"
	"testing"

	"golang-rfc9457/internal/domain/problem"
)

func TestUseCase_CreateProblem_基本的なProblemを生成(t *testing.T) {
	uc := NewUseCase()

	input := CreateProblemInput{
		Status: http.StatusNotFound,
	}

	result := uc.CreateProblem(input)

	if result.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", result.Status, http.StatusNotFound)
	}
	if result.Type != "about:blank" {
		t.Errorf("Type = %q, want %q", result.Type, "about:blank")
	}
}

func TestUseCase_CreateProblem_カスタム値を設定(t *testing.T) {
	uc := NewUseCase()

	input := CreateProblemInput{
		Status:   http.StatusForbidden,
		Type:     "https://api.example.com/problems/out-of-credit",
		Title:    "クレジットが不足しています",
		Detail:   "現在の残高は30です",
		Instance: "/account/12345",
		Extensions: map[string]any{
			"balance": 30,
		},
	}

	result := uc.CreateProblem(input)

	if result.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", result.Status, http.StatusForbidden)
	}
	if result.Type != input.Type {
		t.Errorf("Type = %q, want %q", result.Type, input.Type)
	}
	if result.Title != input.Title {
		t.Errorf("Title = %q, want %q", result.Title, input.Title)
	}
	if result.Detail != input.Detail {
		t.Errorf("Detail = %q, want %q", result.Detail, input.Detail)
	}
	if result.Instance != input.Instance {
		t.Errorf("Instance = %q, want %q", result.Instance, input.Instance)
	}
	if result.Extensions["balance"] != 30 {
		t.Errorf("Extensions[balance] = %v, want 30", result.Extensions["balance"])
	}
}

func TestUseCase_CreateValidationError_複数のエラーを含むProblemを生成(t *testing.T) {
	uc := NewUseCase()

	errors := []ValidationErrorDetail{
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

	result := uc.CreateValidationError("/api/users", errors)

	if result.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", result.Status, http.StatusBadRequest)
	}
	if result.Type != "https://api.example.com/problems/validation-error" {
		t.Errorf("Type = %q, want validation-error type", result.Type)
	}

	validationErrors, ok := result.Extensions["validation_errors"].([]ValidationErrorDetail)
	if !ok {
		t.Fatalf("Extensions[validation_errors] is not []ValidationErrorDetail")
	}
	if len(validationErrors) != 2 {
		t.Errorf("validation_errors length = %d, want 2", len(validationErrors))
	}
}

func TestUseCase_CreateRateLimitError_レート制限エラーを生成(t *testing.T) {
	uc := NewUseCase()

	input := RateLimitInput{
		Instance:      "/api/search",
		Limit:         100,
		Remaining:     0,
		ResetAt:       "2025-10-29T15:00:00Z",
		RetryAfter:    3600,
		WindowSeconds: 3600,
	}

	result := uc.CreateRateLimitError(input)

	if result.Status != http.StatusTooManyRequests {
		t.Errorf("Status = %d, want %d", result.Status, http.StatusTooManyRequests)
	}
	if result.Extensions["limit"] != 100 {
		t.Errorf("Extensions[limit] = %v, want 100", result.Extensions["limit"])
	}
	if result.Extensions["remaining"] != 0 {
		t.Errorf("Extensions[remaining] = %v, want 0", result.Extensions["remaining"])
	}
	if result.Extensions["retry_after"] != 3600 {
		t.Errorf("Extensions[retry_after] = %v, want 3600", result.Extensions["retry_after"])
	}
}

// ProblemRepository リポジトリインターフェースのテスト用モック
type mockProblemRepository struct {
	problems []*problem.Problem
}

func (m *mockProblemRepository) Save(p *problem.Problem) error {
	m.problems = append(m.problems, p)
	return nil
}

func TestUseCase_SaveProblem_リポジトリに保存(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewUseCase(WithRepository(repo))

	p := problem.New(http.StatusNotFound)
	err := uc.SaveProblem(p)

	if err != nil {
		t.Errorf("SaveProblem failed: %v", err)
	}
	if len(repo.problems) != 1 {
		t.Errorf("repository should have 1 problem, got %d", len(repo.problems))
	}
}
