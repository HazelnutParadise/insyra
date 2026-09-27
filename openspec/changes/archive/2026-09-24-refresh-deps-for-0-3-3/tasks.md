# Tasks: refresh-deps-for-0-3-3

- [x] 1.1 `go get -u -t ./... go@1.25.12`；`go mod tidy`
- [x] 1.2 確認 `go.mod` 的 `go` 指令仍是 1.25.12，沒有新增 `toolchain` 行
- [x] 1.3 `accel/internal/wgpu` 改用 `gputypes.BufferUsage`、`gputypes.BindGroupLayoutEntry`（`gogpu/wgpu` v0.34.1 拿掉了這兩個別名）
- [x] 1.4 `chromedp` 停在 v0.12.1，`cdproto` 用它要求的版本
- [x] 2.1 `go build ./...`、`go vet ./...`、`go test ./...` 全綠，`golangci-lint run` 0 issues
- [x] 2.2 `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel/...` 在 Apple M3（Metal）上全綠
- [x] 2.3 CI 的 govulncheck v1.3.0 用 Go 1.25.14 跑完不當掉，只回報 GO-2026-6452（excelize，上游沒有修正版）
- [x] 2.4 `datafetch.YFinance` 對 AAPL 的 History、Dividends、Quote、Info、IncomeStatement，新舊依賴輸出一致
- [x] 3.1 `AGENTS.md` Follow-ups 列出被 Go 1.25 擋住的模組和 chromedp 的上限
- [x] 3.2 `openspec validate refresh-deps-for-0-3-3 --strict` 通過
