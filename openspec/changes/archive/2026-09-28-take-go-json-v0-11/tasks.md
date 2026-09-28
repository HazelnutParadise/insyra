# Tasks: take-go-json-v0-11

## 1. Upgrade

- [x] 1.1 `go get github.com/goccy/go-json@v0.11.1`；`go mod tidy`；`go` 指令仍是 1.25.12，沒有新增模組
- [x] 1.2 GitHub advisory database 查 go-json：v0.11.1 不在任何範圍內

## 2. Tests

- [x] 2.1 新測試涵蓋 spec 的五個情境，在 v0.10.6 上失敗（`1e400`、UTF-8、`1e-7`、`01`），在 v0.11.1 上通過；trailing comma 兩版都拒收，列為回歸保護
- [x] 2.2 `go build ./...`、`go vet ./...`、`go test ./...` 全綠，`golangci-lint run` 0 issues
- [x] 2.3 govulncheck v1.3.0 用 Go 1.25.14 跑完不當掉，結果與 refresh-deps-for-0-3-4 相同

## 3. Records

- [x] 3.1 `Docs/DataTable.md` 的 `ReadJSON` 與 `ToJSON` 說明新行為
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `### Core` 加上行為改變（含 BREAKING）與測速結果
- [x] 3.3 `AGENTS.md`：刪除 go-json 那條 Follow-up，更新數字轉文字那條裡 `ToJSON` 的寫法
- [x] 3.4 `openspec validate take-go-json-v0-11 --strict` 通過
