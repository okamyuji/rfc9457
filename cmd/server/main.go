// Package main RFC 9457 Problem Details Demo Serverのエントリーポイント
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-rfc9457/internal/adapter/handler"
	"golang-rfc9457/internal/adapter/presenter"
	"golang-rfc9457/internal/infrastructure/server"
	problemuc "golang-rfc9457/internal/usecase/problem"
)

// Config アプリケーションの設定
type Config struct {
	Addr   string
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultConfig デフォルト設定を返す（PORT環境変数をサポート）
func DefaultConfig() *Config {
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	return &Config{
		Addr:   addr,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// App アプリケーションの依存性を保持する
type App struct {
	server         *server.Server
	problemHandler *handler.ProblemHandler
	useCase        *problemuc.UseCase
	presenter      *presenter.ProblemPresenter
}

// NewApp 依存性注入を行いAppインスタンスを生成する
func NewApp(opts ...server.ServerOption) *App {
	// Domain層: 依存なし

	// UseCase層: Domain層に依存
	uc := problemuc.NewUseCase()

	// Adapter層: UseCase層に依存
	pres := presenter.NewProblemPresenter()
	h := handler.NewProblemHandler(uc, pres)

	// Infrastructure層: Adapter層に依存
	srv := server.NewServer(h, opts...)

	return &App{
		server:         srv,
		problemHandler: h,
		useCase:        uc,
		presenter:      pres,
	}
}

// Run アプリケーションを実行する
func Run(ctx context.Context, cfg *Config) error {
	app := NewApp(server.WithAddr(cfg.Addr))

	_, _ = fmt.Fprintf(cfg.Stdout, "RFC 9457 Problem Details Demo Server starting on %s\n", app.server.Addr())

	// サーバーをgoroutineで起動
	errCh := make(chan error, 1)
	go func() {
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// シャットダウンシグナルまたはコンテキストのキャンセルを待機
	select {
	case <-ctx.Done():
		_, _ = fmt.Fprintln(cfg.Stdout, "Shutting down server...")
	case err := <-errCh:
		return err
	}

	// グレースフルシャットダウン
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.server.Shutdown(shutdownCtx); err != nil {
		_, _ = fmt.Fprintf(cfg.Stderr, "Server shutdown error: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintln(cfg.Stdout, "Server stopped")
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := DefaultConfig()

	if err := Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
