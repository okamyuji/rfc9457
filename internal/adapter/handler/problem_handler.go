// Package handler HTTPリクエストを処理するハンドラー層
package handler

import (
	"net/http"

	"golang-rfc9457/internal/adapter/presenter"
	"golang-rfc9457/internal/domain/problem"
	problemuc "golang-rfc9457/internal/usecase/problem"
)

// ProblemHandlerInterface Problemハンドラーのインターフェース
type ProblemHandlerInterface interface {
	WriteProblem(w http.ResponseWriter, p *problem.Problem)
	HandleNotFound(w http.ResponseWriter, r *http.Request)
	HandleMethodNotAllowed(w http.ResponseWriter, r *http.Request)
	HandleInternalServerError(w http.ResponseWriter, r *http.Request, detail string)
	HandleValidationError(w http.ResponseWriter, r *http.Request, errors []problemuc.ValidationErrorDetail)
}

// ProblemHandler RFC 9457 Problem Detailsのハンドラー
type ProblemHandler struct {
	useCase   *problemuc.UseCase
	presenter *presenter.ProblemPresenter
}

// NewProblemHandler 新しいProblemHandlerインスタンスを生成する
func NewProblemHandler(uc *problemuc.UseCase, pres *presenter.ProblemPresenter) *ProblemHandler {
	return &ProblemHandler{
		useCase:   uc,
		presenter: pres,
	}
}

// WriteProblem ProblemをHTTPレスポンスとして書き込む
func (h *ProblemHandler) WriteProblem(w http.ResponseWriter, p *problem.Problem) {
	headers := h.presenter.GetHeaders(p)
	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.Header().Set("Content-Type", h.presenter.GetContentType())
	w.WriteHeader(p.Status)

	data, err := h.presenter.ToJSON(p)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
}

// HandleNotFound 404 Not Foundエラーを処理する
func (h *ProblemHandler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	p := h.useCase.CreateProblem(problemuc.CreateProblemInput{
		Status:   http.StatusNotFound,
		Detail:   "リクエストされたリソースは存在しません",
		Instance: r.URL.Path,
	})
	h.WriteProblem(w, p)
}

// HandleMethodNotAllowed 405 Method Not Allowedエラーを処理する
func (h *ProblemHandler) HandleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	p := h.useCase.CreateProblem(problemuc.CreateProblemInput{
		Status:   http.StatusMethodNotAllowed,
		Detail:   "このリソースでは" + r.Method + "メソッドは許可されていません",
		Instance: r.URL.Path,
	})
	h.WriteProblem(w, p)
}

// HandleInternalServerError 500 Internal Server Errorを処理する
// detail引数はログ用で、レスポンスには含めない（セキュリティ）
func (h *ProblemHandler) HandleInternalServerError(w http.ResponseWriter, r *http.Request, _ string) {
	p := h.useCase.CreateProblem(problemuc.CreateProblemInput{
		Status:   http.StatusInternalServerError,
		Detail:   "予期しないエラーが発生しました",
		Instance: r.URL.Path,
	})
	h.WriteProblem(w, p)
}

// HandleValidationError バリデーションエラーを処理する
func (h *ProblemHandler) HandleValidationError(w http.ResponseWriter, r *http.Request, errors []problemuc.ValidationErrorDetail) {
	p := h.useCase.CreateValidationError(r.URL.Path, errors)
	h.WriteProblem(w, p)
}
