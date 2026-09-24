# Tasks: hold-grpc-below-the-xds-advisory

- [x] 1.1 `go get google.golang.org/grpc@v1.83.2`；`go mod tidy`，確認沒有其他模組跟著動
- [x] 1.2 確認 `go.mod` 的 `go` 指令仍是 1.25.12
- [x] 2.1 `go build ./...`、`go vet ./...`、`go test ./...` 全綠，`golangci-lint run` 0 issues
- [x] 2.2 `go.mod` 的 156 個模組逐一比對 GitHub advisory 資料庫：0 個落在範圍內（改之前是 grpc 1 個）
- [x] 2.3 govulncheck v1.3.0 用 Go 1.25.14 只回報 GO-2026-6452
- [x] 3.1 `AGENTS.md`：升級規則加上 advisory 比對，grpc 列入暫緩清單，excelize follow-up 記下 GO-2026-6452 是資料庫錯誤
- [x] 3.2 `openspec validate hold-grpc-below-the-xds-advisory --strict` 通過
