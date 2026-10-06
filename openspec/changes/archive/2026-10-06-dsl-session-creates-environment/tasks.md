# Tasks: dsl-session-creates-environment

## 1. 測試先行

- [x] 1.1 `engine/dsl/manager_test.go`：不存在的環境會被建立、存在的會沿用（第二個 session 拿得到 `x`）、不合法的名稱報錯且不留下環境、兩個 goroutine 同時開同一個不存在的環境都成功；在舊程式上前兩項與並行那項失敗
- [x] 1.2 `internal/dsl/env/create_race_test.go`：並行 `Create` 只有一個成功、寫預設檔案不覆寫已存在的 `state.json`、並行 `EnsureDefaultEnvironment` 全部成功；在舊程式上前兩項失敗（一輪有 2 個 `Create` 成功，`state.json` 被蓋成空的）

## 2. 實作

- [x] 2.1 `internal/dsl/session.go`：`NewSession` 遇到不存在的環境就建立，建立失敗但環境已存在時照常開啟
- [x] 2.2 `internal/dsl/env/manager.go`：`Create` 改用 `os.Mkdir` 搶建立權；`writeDefaultFiles` 只寫不存在的檔案；`EnsureDefaultEnvironment` 容許別人搶先建好
- [x] 2.3 `engine/dsl/dsl.go`：`NewSession` 說明改寫

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：說明自動建立與 CLI 維持嚴格；範例 C 拿掉 `Exists`／`Create` 的判斷，並在 scratchpad 的獨立 module 實際編譯、執行；方法表的 `Create` 寫明並行行為
- [x] 3.2 `engine/README.md`
- [x] 3.3 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增兩條；修正 `engine-dsl-manager` 那條已不成立的說法
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.5 skills：不需要修改

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、相關套件 `go test -race`、`golangci-lint run`（0 issues）、`openspec validate dsl-session-creates-environment --strict`
