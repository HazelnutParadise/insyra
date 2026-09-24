# Tasks: bump-grpc-xds-advisory

- [x] 1.1 `go get google.golang.org/grpc@v1.83.2`；`go mod tidy`
- [x] 1.2 確認 `go.mod` 的 `go` 指令仍是 1.25.12
- [x] 2.1 `go build ./...`、`go test ./...` 全綠
- [x] 2.2 `govulncheck ./...` 不再回報 grpc
- [x] 3.1 `openspec validate bump-grpc-xds-advisory --strict` 通過
- [x] 3.2 合併進 `0.4`
