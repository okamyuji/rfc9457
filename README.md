# golang-rfc9457

RFC 9457 (Problem Details for HTTP APIs) のGo実装ライブラリです。

## 概要

HTTP APIのエラーレスポンスを標準化された `application/problem+json` 形式で返すためのライブラリです。

## インストール

```bash
go get golang-rfc9457
```

## 基本的な使い方

### Problem Detailsの作成

```go
import (
    "net/http"
    "golang-rfc9457/pkg/rfc9457"
)

// 基本的なエラー
p := rfc9457.New(http.StatusNotFound,
    rfc9457.WithDetail("指定されたユーザーが見つかりません"),
    rfc9457.WithInstance("/users/12345"),
)

// HTTPレスポンスとして書き込み
rfc9457.WriteProblem(w, p)
```

### カスタムProblem Type

```go
p := rfc9457.New(http.StatusForbidden,
    rfc9457.WithType("https://api.example.com/problems/out-of-credit"),
    rfc9457.WithTitle("クレジット不足"),
    rfc9457.WithDetail("現在の残高は30ですが、この操作には50が必要です"),
    rfc9457.WithExtension("balance", 30),
    rfc9457.WithExtension("required", 50),
)
```

### バリデーションエラー

```go
errors := []rfc9457.ValidationError{
    {Field: "email", Code: "INVALID_FORMAT", Message: "メールアドレスの形式が正しくありません"},
    {Field: "password", Code: "TOO_SHORT", Message: "パスワードは8文字以上必要です"},
}

p := rfc9457.NewValidationError("/api/users", errors)
rfc9457.WriteProblem(w, p)
```

### レート制限エラー

```go
p := rfc9457.NewRateLimitError(
    "/api/search",      // instance
    100,                // limit
    0,                  // remaining
    "2025-01-27T15:00:00Z", // reset_at
    3600,               // retry_after (秒)
)
rfc9457.WriteProblem(w, p)
// Retry-After ヘッダーも自動的に設定されます
```

## レスポンス例

```json
{
  "type": "https://api.example.com/problems/out-of-credit",
  "status": 403,
  "title": "クレジット不足",
  "detail": "現在の残高は30ですが、この操作には50が必要です",
  "instance": "/account/12345",
  "balance": 30,
  "required": 50
}
```

## RFC 9457 標準メンバー

| フィールド | 説明 |
| --------- | ---- |
| `type` | 問題タイプを識別するURI（デフォルト: `about:blank`） |
| `status` | HTTPステータスコード |
| `title` | 問題タイプの短い要約 |
| `detail` | この発生に固有の説明 |
| `instance` | 問題発生を識別するURI |

上記以外のフィールドは**拡張メンバー**として自由に追加できます。

## デモサーバー

```bash
# サーバー起動
go run ./cmd/server/

# または環境変数でポート指定
PORT=9090 go run ./cmd/server/
```

### エンドポイント

| パス | 説明 |
| --- | ---- |
| `GET /` | サーバー情報 |
| `GET /health` | ヘルスチェック |
| `GET /demo/not-found` | 404エラーのデモ |
| `GET /demo/validation-error` | バリデーションエラーのデモ |
| `GET /demo/rate-limit` | レート制限エラーのデモ |
| `GET /demo/internal-error` | 内部エラーのデモ |

## 開発

### 品質チェック

```bash
./check.sh
```

以下を実行します

- `go fmt` - フォーマット
- `go vet` - 静的解析
- `staticcheck` - 追加静的解析
- `golangci-lint` - 総合リンター
- `go test` - テスト（カバレッジ80%以上必須）
- `go build` - ビルド確認

### E2Eテスト

```bash
./scripts/e2e_test.sh
```

### アーキテクチャ

DDD + Clean Architectureで設計されています。

```shell
├── cmd/server/          # エントリーポイント
├── internal/
│   ├── domain/problem/  # ドメイン層
│   ├── usecase/problem/ # ユースケース層
│   ├── adapter/
│   │   ├── handler/     # HTTPハンドラー
│   │   └── presenter/   # レスポンス整形
│   └── infrastructure/server/  # HTTPサーバー
├── pkg/rfc9457/         # 公開API
└── testutil/            # テストユーティリティ
```

## ライセンス

MIT License
