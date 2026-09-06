# Tasks: bump-vulnerable-deps

## 1. Bump

- [x] 1.1 `go get github.com/apache/thrift@v0.24.0 google.golang.org/grpc@v1.83.1 golang.org/x/image@v0.45.0 golang.org/x/crypto@v0.55.0 && go mod tidy`
- [x] 1.2 Confirm `go.mod` still says `go 1.25.12`

## 2. Verification

- [x] 2.1 `go build ./...`；`go test ./...` 全綠
- [x] 2.2 `govulncheck ./...` 不再列出 GO-2026-6222、GO-2026-6303；thrift／grpc 版本符合 #201／#202
- [x] 2.3 `openspec validate bump-vulnerable-deps --strict` 通過

## 3. Close-out

- [x] 3.1 Commit references #201 #202 #204 and notes #203 partial; #203 stays open for v0.56.0
