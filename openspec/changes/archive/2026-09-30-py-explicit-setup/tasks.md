# Tasks: py-explicit-setup

## 1. 測試先行

- [x] 1.1 `py/setup_test.go`：`Setup` 在未準備的環境執行一次 uv sync 並標記完成，之後的 `pyEnvInit` 不再呼叫 uv；已準備好時再呼叫 `Setup` 不呼叫 uv；同步失敗時回傳錯誤、不標記完成，下一次 `Setup` 重試；nil context 回傳 `errNilContext`、已取消的 context 回傳 `Canceled`，且都不碰環境目錄。在舊程式上確認失敗（`Setup` 不存在）

## 2. 實作

- [x] 2.1 `py/py.go`：`Setup(ctx)`，先檢查 context，再呼叫 `pyEnvInit(ctx)`；1.1 通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/py.md`：首次使用一節提到可以先呼叫 `Setup`；Environment Utilities 新增 `Setup` 一節
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### py` 段落末尾新增條目
- [x] 3.3 `api-review.md`：PY-1 標為已修正
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目，記下擁有者的裁定
- [x] 3.5 檢查 `skills/insyra/SKILL.md`：不列 `py` 的 API，本變更不需修改

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate py-explicit-setup --strict`
