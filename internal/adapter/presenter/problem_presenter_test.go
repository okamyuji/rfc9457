package presenter

import (
	"encoding/json"
	"net/http"
	"testing"

	"golang-rfc9457/internal/domain/problem"
)

func TestProblemPresenter_ToJSON_正常にJSONを生成(t *testing.T) {
	presenter := NewProblemPresenter()
	p := problem.New(http.StatusNotFound)

	result, err := presenter.ToJSON(p)

	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("ToJSON returned empty result")
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if parsed["status"] != float64(http.StatusNotFound) {
		t.Errorf("status = %v, want %d", parsed["status"], http.StatusNotFound)
	}
}

func TestProblemPresenter_ToJSON_拡張フィールドを含む(t *testing.T) {
	presenter := NewProblemPresenter()
	p := problem.New(http.StatusForbidden,
		problem.WithExtension("balance", 30),
	)

	result, err := presenter.ToJSON(p)

	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if parsed["balance"] != float64(30) {
		t.Errorf("balance = %v, want 30", parsed["balance"])
	}
}

func TestProblemPresenter_GetContentType_application_problem_json(t *testing.T) {
	presenter := NewProblemPresenter()

	contentType := presenter.GetContentType()

	if contentType != "application/problem+json" {
		t.Errorf("GetContentType() = %q, want %q", contentType, "application/problem+json")
	}
}

func TestProblemPresenter_GetHeaders_Retry_Afterヘッダーを含む(t *testing.T) {
	presenter := NewProblemPresenter()
	p := problem.New(http.StatusTooManyRequests,
		problem.WithExtension("retry_after", 3600),
	)

	headers := presenter.GetHeaders(p)

	if headers["Retry-After"] != "3600" {
		t.Errorf("Retry-After = %q, want %q", headers["Retry-After"], "3600")
	}
}

func TestProblemPresenter_GetHeaders_Retry_Afterなしの場合(t *testing.T) {
	presenter := NewProblemPresenter()
	p := problem.New(http.StatusNotFound)

	headers := presenter.GetHeaders(p)

	if _, exists := headers["Retry-After"]; exists {
		t.Error("Retry-After should not be present")
	}
}
