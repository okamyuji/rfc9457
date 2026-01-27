package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRun_正常に起動してシャットダウン(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		Addr:   ":0",
		Stdout: &stdout,
		Stderr: &stderr,
	}

	err := Run(ctx, cfg)

	// コンテキストがキャンセルされたことによるシャットダウンはエラーではない
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Run failed: %v", err)
	}
}

func TestRun_起動メッセージを出力(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		Addr:   ":0",
		Stdout: &stdout,
		Stderr: &stderr,
	}

	_ = Run(ctx, cfg)

	output := stdout.String()
	if !strings.Contains(output, "RFC 9457") {
		t.Errorf("起動メッセージにRFC 9457が含まれていません: %q", output)
	}
}

func TestRun_デフォルト設定(t *testing.T) {
	// PORT環境変数をクリア
	t.Setenv("PORT", "")

	cfg := DefaultConfig()

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.Stdout == nil {
		t.Error("Stdout should not be nil")
	}
	if cfg.Stderr == nil {
		t.Error("Stderr should not be nil")
	}
}

func TestRun_PORT環境変数でアドレスを設定(t *testing.T) {
	t.Setenv("PORT", "9999")

	cfg := DefaultConfig()

	if cfg.Addr != ":9999" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9999")
	}
}

func TestNewApp_依存性注入が正しく行われる(t *testing.T) {
	app := NewApp()

	if app.server == nil {
		t.Error("server should not be nil")
	}
	if app.problemHandler == nil {
		t.Error("problemHandler should not be nil")
	}
}
