package problem

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNewProblem_デフォルト値が設定される(t *testing.T) {
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

func TestNewProblem_カスタム値を設定できる(t *testing.T) {
	p := New(http.StatusForbidden,
		WithType("https://api.example.com/problems/out-of-credit"),
		WithTitle("クレジットが不足しています"),
		WithDetail("現在の残高は30ですが、この操作には50が必要です。"),
		WithInstance("/account/12345/transactions/abc"),
	)

	if p.Type != "https://api.example.com/problems/out-of-credit" {
		t.Errorf("Type = %q, want %q", p.Type, "https://api.example.com/problems/out-of-credit")
	}
	if p.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want %d", p.Status, http.StatusForbidden)
	}
	if p.Title != "クレジットが不足しています" {
		t.Errorf("Title = %q, want %q", p.Title, "クレジットが不足しています")
	}
	if p.Detail != "現在の残高は30ですが、この操作には50が必要です。" {
		t.Errorf("Detail = %q, want expected value", p.Detail)
	}
	if p.Instance != "/account/12345/transactions/abc" {
		t.Errorf("Instance = %q, want %q", p.Instance, "/account/12345/transactions/abc")
	}
}

func TestProblem_拡張フィールドを追加できる(t *testing.T) {
	p := New(http.StatusForbidden,
		WithExtension("balance", 30),
		WithExtension("accounts", []string{"/account/12345", "/account/67890"}),
	)

	if p.Extensions["balance"] != 30 {
		t.Errorf("Extensions[balance] = %v, want 30", p.Extensions["balance"])
	}
	accounts, ok := p.Extensions["accounts"].([]string)
	if !ok || len(accounts) != 2 {
		t.Errorf("Extensions[accounts] = %v, want slice with 2 elements", p.Extensions["accounts"])
	}
}

func TestProblem_JSONシリアライズ_標準フィールドのみ(t *testing.T) {
	p := New(http.StatusNotFound)

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if result["type"] != "about:blank" {
		t.Errorf("type = %v, want %q", result["type"], "about:blank")
	}
	if result["status"] != float64(http.StatusNotFound) {
		t.Errorf("status = %v, want %v", result["status"], http.StatusNotFound)
	}
	if result["title"] != "Not Found" {
		t.Errorf("title = %v, want %q", result["title"], "Not Found")
	}
	// detail と instance は空なので含まれない
	if _, exists := result["detail"]; exists {
		t.Error("detail should not be present when empty")
	}
	if _, exists := result["instance"]; exists {
		t.Error("instance should not be present when empty")
	}
}

func TestProblem_JSONシリアライズ_拡張フィールド含む(t *testing.T) {
	p := New(http.StatusForbidden,
		WithType("https://api.example.com/problems/out-of-credit"),
		WithDetail("残高不足"),
		WithExtension("balance", 30),
		WithExtension("currency", "USD"),
	)

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// 拡張フィールドがトップレベルに展開される
	if result["balance"] != float64(30) {
		t.Errorf("balance = %v, want 30", result["balance"])
	}
	if result["currency"] != "USD" {
		t.Errorf("currency = %v, want %q", result["currency"], "USD")
	}
}

func TestProblem_Error_エラーインターフェースを実装(t *testing.T) {
	p := New(http.StatusNotFound,
		WithTitle("Not Found"),
		WithDetail("リソースが見つかりません"),
	)

	var err error = p
	expected := "Not Found: リソースが見つかりません"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestProblem_Error_Detailなしの場合(t *testing.T) {
	p := New(http.StatusNotFound)

	var err error = p
	expected := "Not Found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestGetStatusTitle_既知のステータスコード(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{http.StatusBadRequest, "Bad Request"},
		{http.StatusUnauthorized, "Unauthorized"},
		{http.StatusForbidden, "Forbidden"},
		{http.StatusNotFound, "Not Found"},
		{http.StatusConflict, "Conflict"},
		{http.StatusTooManyRequests, "Too Many Requests"},
		{http.StatusInternalServerError, "Internal Server Error"},
		{http.StatusBadGateway, "Bad Gateway"},
		{http.StatusServiceUnavailable, "Service Unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GetStatusTitle(tt.status)
			if got != tt.want {
				t.Errorf("GetStatusTitle(%d) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGetStatusTitle_未知のステータスコード(t *testing.T) {
	got := GetStatusTitle(999)
	if got != "Unknown Status" {
		t.Errorf("GetStatusTitle(999) = %q, want %q", got, "Unknown Status")
	}
}
