package testutil

import (
	"net/http/httptest"
	"testing"
)

func TestAssertJSONField_正しい値を検証(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Body.WriteString(`{"status": 404, "title": "Not Found"}`)

	// これはパニックしないはず
	AssertJSONField(t, rec, "status", 404)
	AssertJSONField(t, rec, "title", "Not Found")
}

func TestAssertStatusCode_正しいステータスを検証(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Code = 404

	// これはパニックしないはず
	AssertStatusCode(t, rec, 404)
}

func TestAssertContentType_正しいContent_Typeを検証(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/problem+json")

	// これはパニックしないはず
	AssertContentType(t, rec, "application/problem+json")
}
